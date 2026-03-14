package context

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"ok/internal/logger"
	"ok/providers"
	"ok/app/memory"
	"ok/internal/skills"
)

type ContextBuilder struct {
	workspace     string
	skillsLoader  *skills.SkillsLoader
	personaLoader *PersonaLoader
	memory        *MemoryStore
	retriever     *memory.Retriever    // optional RAG retriever for semantic memory
	ragCache      *RAGContextCache // caches formatted RAG context blocks

	// Cache for system prompt parts to avoid rebuilding on every call.
	// This fixes issue #607: repeated reprocessing of the entire context.
	// The cache auto-invalidates when workspace source files change (mtime check).
	systemPromptMutex sync.RWMutex
	cachedParts       *promptParts // nil means no cache
	cachedAt          time.Time    // max observed mtime across tracked paths at cache build time

	// existedAtCache tracks which source file paths existed the last time the
	// cache was built. This lets sourceFilesChanged detect files that are newly
	// created (didn't exist at cache time, now exist) or deleted (existed at
	// cache time, now gone) — both of which should trigger a cache rebuild.
	existedAtCache map[string]bool

	// skillFilesAtCache snapshots the skill tree file set and mtimes at cache
	// build time. This catches nested file creations/deletions/mtime changes
	// that may not update the top-level skill root directory mtime.
	skillFilesAtCache map[string]time.Time
}

func getGlobalConfigDir() string {
	if home := os.Getenv("OK_HOME"); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ok")
}

func NewContextBuilder(workspace string) *ContextBuilder {
	// builtin skills: skills directory in current project
	// Use the skills/ directory under the current working directory
	builtinSkillsDir := strings.TrimSpace(os.Getenv("OK_BUILTIN_SKILLS"))
	if builtinSkillsDir == "" {
		wd, _ := os.Getwd()
		builtinSkillsDir = filepath.Join(wd, "skills")
	}
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	return &ContextBuilder{
		workspace:     workspace,
		skillsLoader:  skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		personaLoader: NewPersonaLoader(workspace),
		memory:        NewMemoryStore(workspace),
		ragCache:      NewRAGContextCache(30, 2*time.Minute),
	}
}

func (cb *ContextBuilder) getIdentity() string {
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))

	return fmt.Sprintf(`# ok ✓

You are ok, a personal AI assistant.

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

## Important Rules

1. **Use tools** — When you need to perform an action, call the appropriate tool. Do not simulate or narrate actions you can actually execute.

2. **Match the user's language** — Reply in the same language the user writes in. If they switch languages, follow.

3. **Be concise** — Explain what you're doing briefly. One sentence before a tool call, not a paragraph.

4. **Memory** — When the user shares something worth remembering, update %s/memory/MEMORY.md. Don't memorize routine interactions.

5. **Context summaries** — Conversation summaries provided as context are approximate. Always defer to explicit user instructions over summary content.

6. **Safety** — Never execute destructive operations without explicit user confirmation. Flag risks before acting.

7. **Heartbeats** — When you receive a heartbeat, check HEARTBEAT.md for tasks. Respond with HEARTBEAT_OK if nothing needs attention.

8. **Voice messages** — Messages containing ` + "`[voice: text]`" + ` are voice messages already transcribed to text. Treat the text inside as what the user said. Do NOT say you cannot process audio — the transcription is already done.`,
		workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

// promptParts holds the two halves of the system prompt, split by volatility
// so that LLM-side prompt caching (Anthropic ephemeral, OpenAI prefix) can
// cache the stable core independently of the more-frequently-changing memory.
type promptParts struct {
	core   string // identity + persona + skills — rarely changes
	memory string // MEMORY.md context — changes occasionally
}

// combined returns the full system prompt as a single string (used by tests
// and any code that doesn't need the split).
func (p *promptParts) combined() string {
	if p.memory == "" {
		return p.core
	}
	return p.core + "\n\n---\n\n" + p.memory
}

// buildPromptParts builds the two prompt halves from disk.
func (cb *ContextBuilder) buildPromptParts() *promptParts {
	// --- Core: identity + persona + skills (rarely changes) ---
	coreParts := []string{cb.getIdentity()}

	persona := cb.personaLoader.Load()
	if section := persona.BuildPromptSection(); section != "" {
		coreParts = append(coreParts, section)
	}

	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		coreParts = append(coreParts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary))
	}

	// --- Memory: changes more often ---
	memoryContext := cb.memory.GetMemoryContext()
	mem := ""
	if memoryContext != "" {
		mem = "# Memory\n\n" + memoryContext
	}

	return &promptParts{
		core:   strings.Join(coreParts, "\n\n---\n\n"),
		memory: mem,
	}
}

// BuildSystemPrompt returns the full system prompt as a single string.
// Kept for backward compatibility (tests, LoadBootstrapFiles, etc.).
func (cb *ContextBuilder) BuildSystemPrompt() string {
	return cb.buildPromptParts().combined()
}

// buildPromptPartsWithCache returns the cached prompt parts if available
// and source files haven't changed, otherwise builds and caches them.
// Source file changes are detected via mtime checks (cheap stat calls).
func (cb *ContextBuilder) buildPromptPartsWithCache() *promptParts {
	// Try read lock first — fast path when cache is valid
	cb.systemPromptMutex.RLock()
	if cb.cachedParts != nil && !cb.sourceFilesChangedLocked() {
		result := cb.cachedParts
		cb.systemPromptMutex.RUnlock()
		return result
	}
	cb.systemPromptMutex.RUnlock()

	// Acquire write lock for building
	cb.systemPromptMutex.Lock()
	defer cb.systemPromptMutex.Unlock()

	// Double-check: another goroutine may have rebuilt while we waited
	if cb.cachedParts != nil && !cb.sourceFilesChangedLocked() {
		return cb.cachedParts
	}

	// Snapshot the baseline (existence + max mtime) BEFORE building the prompt.
	// This way cachedAt reflects the pre-build state: if a file is modified
	// during BuildSystemPrompt, its new mtime will be > baseline.maxMtime,
	// so the next sourceFilesChangedLocked check will correctly trigger a
	// rebuild. The alternative (baseline after build) risks caching stale
	// content with a too-new baseline, making the staleness invisible.
	baseline := cb.buildCacheBaseline()
	parts := cb.buildPromptParts()
	cb.cachedParts = parts
	cb.cachedAt = baseline.maxMtime
	cb.existedAtCache = baseline.existed
	cb.skillFilesAtCache = baseline.skillFiles

	logger.DebugCF("agent", "System prompt cached",
		map[string]any{
			"core_len":   len(parts.core),
			"memory_len": len(parts.memory),
		})

	return parts
}

// BuildSystemPromptWithCache returns the full cached system prompt as a single
// string. Used by tests and any code that doesn't need the core/memory split.
func (cb *ContextBuilder) BuildSystemPromptWithCache() string {
	return cb.buildPromptPartsWithCache().combined()
}

// InvalidateCache clears the cached system prompt.
// Normally not needed because the cache auto-invalidates via mtime checks,
// but this is useful for tests or explicit reload commands.
func (cb *ContextBuilder) InvalidateCache() {
	cb.systemPromptMutex.Lock()
	defer cb.systemPromptMutex.Unlock()

	cb.cachedParts = nil
	cb.cachedAt = time.Time{}
	cb.existedAtCache = nil
	cb.skillFilesAtCache = nil

	logger.DebugCF("agent", "System prompt cache invalidated", nil)
}

// sourcePaths returns non-skill workspace source files tracked for cache
// invalidation (bootstrap files + memory). Skill roots are handled separately
// because they require both directory-level and recursive file-level checks.
func (cb *ContextBuilder) sourcePaths() []string {
	return []string{
		filepath.Join(cb.workspace, "AGENTS.md"),
		filepath.Join(cb.workspace, "SOUL.md"),
		filepath.Join(cb.workspace, "USER.md"),
		filepath.Join(cb.workspace, "IDENTITY.md"),
		filepath.Join(cb.workspace, "memory", "MEMORY.md"),
	}
}

// skillRoots returns all skill root directories that can affect
// BuildSkillsSummary output (workspace/global/builtin).
func (cb *ContextBuilder) skillRoots() []string {
	if cb.skillsLoader == nil {
		return []string{filepath.Join(cb.workspace, "skills")}
	}

	roots := cb.skillsLoader.SkillRoots()
	if len(roots) == 0 {
		return []string{filepath.Join(cb.workspace, "skills")}
	}
	return roots
}

// cacheBaseline holds the file existence snapshot and the latest observed
// mtime across all tracked paths. Used as the cache reference point.
type cacheBaseline struct {
	existed    map[string]bool
	skillFiles map[string]time.Time
	maxMtime   time.Time
}

// buildCacheBaseline records which tracked paths currently exist and computes
// the latest mtime across all tracked files + skills directory contents.
// Called under write lock when the cache is built.
func (cb *ContextBuilder) buildCacheBaseline() cacheBaseline {
	skillRoots := cb.skillRoots()

	// All paths whose existence we track: source files + all skill roots.
	allPaths := append(cb.sourcePaths(), skillRoots...)

	existed := make(map[string]bool, len(allPaths))
	skillFiles := make(map[string]time.Time)
	var maxMtime time.Time

	for _, p := range allPaths {
		info, err := os.Stat(p)
		existed[p] = err == nil
		if err == nil && info.ModTime().After(maxMtime) {
			maxMtime = info.ModTime()
		}
	}

	// Walk all skill roots recursively to snapshot skill files and mtimes.
	// Use os.Stat (not d.Info) for consistency with sourceFilesChanged checks.
	for _, root := range skillRoots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr == nil && !d.IsDir() {
				if info, err := os.Stat(path); err == nil {
					skillFiles[path] = info.ModTime()
					if info.ModTime().After(maxMtime) {
						maxMtime = info.ModTime()
					}
				}
			}
			return nil
		})
	}

	// If no tracked files exist yet (empty workspace), maxMtime is zero.
	// Use a very old non-zero time so that:
	// 1. cachedAt.IsZero() won't trigger perpetual rebuilds.
	// 2. Any real file created afterwards has mtime > cachedAt, so it
	//    will be detected by fileChangedSince (unlike time.Now() which
	//    could race with a file whose mtime <= Now).
	if maxMtime.IsZero() {
		maxMtime = time.Unix(1, 0)
	}

	return cacheBaseline{existed: existed, skillFiles: skillFiles, maxMtime: maxMtime}
}

// sourceFilesChangedLocked checks whether any workspace source file has been
// modified, created, or deleted since the cache was last built.
//
// IMPORTANT: The caller MUST hold at least a read lock on systemPromptMutex.
// Go's sync.RWMutex is not reentrant, so this function must NOT acquire the
// lock itself (it would deadlock when called from BuildSystemPromptWithCache
// which already holds RLock or Lock).
func (cb *ContextBuilder) sourceFilesChangedLocked() bool {
	if cb.cachedAt.IsZero() {
		return true
	}

	// Check tracked source files (bootstrap + memory).
	if slices.ContainsFunc(cb.sourcePaths(), cb.fileChangedSince) {
		return true
	}

	// --- Skill roots (workspace/global/builtin) ---
	//
	// For each root:
	// 1. Creation/deletion and root directory mtime changes are tracked by fileChangedSince.
	// 2. Nested file create/delete/mtime changes are tracked by the skill file snapshot.
	for _, root := range cb.skillRoots() {
		if cb.fileChangedSince(root) {
			return true
		}
	}
	if skillFilesChangedSince(cb.skillRoots(), cb.skillFilesAtCache) {
		return true
	}

	return false
}

// fileChangedSince returns true if a tracked source file has been modified,
// newly created, or deleted since the cache was built.
//
// Four cases:
//   - existed at cache time, exists now -> check mtime
//   - existed at cache time, gone now   -> changed (deleted)
//   - absent at cache time,  exists now -> changed (created)
//   - absent at cache time,  gone now   -> no change
func (cb *ContextBuilder) fileChangedSince(path string) bool {
	// Defensive: if existedAtCache was never initialized, treat as changed
	// so the cache rebuilds rather than silently serving stale data.
	if cb.existedAtCache == nil {
		return true
	}

	existedBefore := cb.existedAtCache[path]
	info, err := os.Stat(path)
	existsNow := err == nil

	if existedBefore != existsNow {
		return true // file was created or deleted
	}
	if !existsNow {
		return false // didn't exist before, doesn't exist now
	}
	return info.ModTime().After(cb.cachedAt)
}

// errWalkStop is a sentinel error used to stop filepath.WalkDir early.
// Using a dedicated error (instead of fs.SkipAll) makes the early-exit
// intent explicit and avoids the nilerr linter warning that would fire
// if the callback returned nil when its err parameter is non-nil.
var errWalkStop = errors.New("walk stop")

// skillFilesChangedSince compares the current recursive skill file tree
// against the cache-time snapshot. Any create/delete/mtime drift invalidates
// the cache.
func skillFilesChangedSince(skillRoots []string, filesAtCache map[string]time.Time) bool {
	// Defensive: if the snapshot was never initialized, force rebuild.
	if filesAtCache == nil {
		return true
	}

	// Check cached files still exist and keep the same mtime.
	for path, cachedMtime := range filesAtCache {
		info, err := os.Stat(path)
		if err != nil {
			// A previously tracked file disappeared (or became inaccessible):
			// either way, cached skill summary may now be stale.
			return true
		}
		if !info.ModTime().Equal(cachedMtime) {
			return true
		}
	}

	// Check no new files appeared under any skill root.
	changed := false
	for _, root := range skillRoots {
		if strings.TrimSpace(root) == "" {
			continue
		}

		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				// Treat unexpected walk errors as changed to avoid stale cache.
				if !os.IsNotExist(walkErr) {
					changed = true
					return errWalkStop
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if _, ok := filesAtCache[path]; !ok {
				changed = true
				return errWalkStop
			}
			return nil
		})

		if changed {
			return true
		}
		if err != nil && !errors.Is(err, errWalkStop) && !os.IsNotExist(err) {
			logger.DebugCF("agent", "skills walk error", map[string]any{"error": err.Error()})
			return true
		}
	}

	return false
}

// LoadBootstrapFiles returns persona files content. Delegates to PersonaLoader.
func (cb *ContextBuilder) LoadBootstrapFiles() string {
	return cb.personaLoader.Load().BuildPromptSection()
}

// buildDynamicContext returns a short dynamic context string with per-request info.
// This changes every request (time, session) so it is NOT part of the cached prompt.
// LLM-side KV cache reuse is achieved by each provider adapter's native mechanism:
//   - Anthropic: per-block cache_control (ephemeral) on the static SystemParts block
//   - OpenAI / Codex: prompt_cache_key for prefix-based caching
//
// See: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
// See: https://platform.openai.com/docs/guides/prompt-caching
func (cb *ContextBuilder) buildDynamicContext(channel, chatID string) string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	rt := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Current Time\n%s\n\n## Runtime\n%s", now, rt)

	if channel != "" && chatID != "" {
		fmt.Fprintf(&sb, "\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}

	return sb.String()
}

func (cb *ContextBuilder) BuildMessages(
	history []providers.Message,
	summary string,
	currentMessage string,
	media []string,
	channel, chatID string,
) []providers.Message {
	messages := []providers.Message{}

	// The system prompt is split into blocks ordered by volatility (most stable
	// first) to maximize LLM-side prompt caching:
	//   Block 0: core (identity+persona+skills) — rarely changes    → ephemeral
	//   Block 1: memory                         — changes sometimes → ephemeral
	//   Block 2: summary                        — stable per window → ephemeral
	//   Block 3: dynamic (time/session)         — changes every req → no cache
	//   Block 4: RAG                            — changes every req → no cache
	//
	// Anthropic caches by prefix: everything identical from the start is reused.
	// Putting stable content first means even when memory/summary change, the
	// core block stays cached. Up to 4 ephemeral breakpoints are allowed.
	//
	// Everything is sent as a single system message for provider compatibility:
	// - Anthropic adapter extracts messages[0] (Role=="system") and maps its content
	//   to the top-level "system" parameter in the Messages API request.
	// - Codex maps only the first system message to its instructions field.
	// - OpenAI-compat passes messages through as-is.
	parts := cb.buildPromptPartsWithCache()

	stringParts := []string{parts.core}
	contentBlocks := []providers.ContentBlock{
		{Type: "text", Text: parts.core, CacheControl: &providers.CacheControl{Type: "ephemeral"}},
	}

	// Memory block — changes occasionally (when user saves something)
	if parts.memory != "" {
		stringParts = append(stringParts, parts.memory)
		contentBlocks = append(contentBlocks, providers.ContentBlock{
			Type: "text", Text: parts.memory, CacheControl: &providers.CacheControl{Type: "ephemeral"},
		})
	}

	// Summary block — stable within a summarization window
	if summary != "" {
		summaryText := fmt.Sprintf(
			"CONTEXT_SUMMARY: The following is an approximate summary of prior conversation "+
				"for reference only. It may be incomplete or outdated — always defer to explicit instructions.\n\n%s",
			summary)
		stringParts = append(stringParts, summaryText)
		contentBlocks = append(contentBlocks, providers.ContentBlock{
			Type: "text", Text: summaryText, CacheControl: &providers.CacheControl{Type: "ephemeral"},
		})
	}

	// Dynamic context — changes every request (time, runtime, session)
	dynamicCtx := cb.buildDynamicContext(channel, chatID)
	stringParts = append(stringParts, dynamicCtx)
	contentBlocks = append(contentBlocks, providers.ContentBlock{Type: "text", Text: dynamicCtx})

	// RAG: retrieve relevant past interactions for the current message
	if cb.retriever != nil && strings.TrimSpace(currentMessage) != "" {
		ragCtx := cb.retrieveRAGContext(currentMessage)
		if ragCtx != "" {
			stringParts = append(stringParts, ragCtx)
			contentBlocks = append(contentBlocks, providers.ContentBlock{Type: "text", Text: ragCtx})
		}
	}

	fullSystemPrompt := strings.Join(stringParts, "\n\n---\n\n")

	// Log system prompt summary for debugging (debug mode only).
	// Read cachedParts under lock to avoid a data race with
	// concurrent InvalidateCache / buildPromptPartsWithCache writes.
	cb.systemPromptMutex.RLock()
	isCached := cb.cachedParts != nil
	cb.systemPromptMutex.RUnlock()

	logger.DebugCF("agent", "System prompt built",
		map[string]any{
			"core_chars":    len(parts.core),
			"memory_chars":  len(parts.memory),
			"dynamic_chars": len(dynamicCtx),
			"total_chars":   len(fullSystemPrompt),
			"has_summary":   summary != "",
			"cached":        isCached,
		})

	// Log preview of system prompt (avoid logging huge content)
	preview := fullSystemPrompt
	if len(preview) > 500 {
		preview = preview[:500] + "... (truncated)"
	}
	logger.DebugCF("agent", "System prompt preview",
		map[string]any{
			"preview": preview,
		})

	history = sanitizeHistoryForProvider(history)
	history = compactToolHistory(history)

	// Single system message containing all context — compatible with all providers.
	// SystemParts enables cache-aware adapters to set per-block cache_control;
	// Content is the concatenated fallback for adapters that don't read SystemParts.
	messages = append(messages, providers.Message{
		Role:        "system",
		Content:     fullSystemPrompt,
		SystemParts: contentBlocks,
	})

	// Add conversation history
	messages = append(messages, history...)

	// Add current user message
	if strings.TrimSpace(currentMessage) != "" {
		msg := providers.Message{
			Role:    "user",
			Content: currentMessage,
		}
		if len(media) > 0 {
			msg.Media = media
		}
		messages = append(messages, msg)
	}

	return messages
}

func sanitizeHistoryForProvider(history []providers.Message) []providers.Message {
	if len(history) == 0 {
		return history
	}

	sanitized := make([]providers.Message, 0, len(history))
	for _, msg := range history {
		switch msg.Role {
		case "system":
			// Drop system messages from history. BuildMessages always
			// constructs its own single system message (static + dynamic +
			// summary); extra system messages would break providers that
			// only accept one (Anthropic, Codex).
			logger.DebugCF("agent", "Dropping system message from history", map[string]any{})
			continue

		case "tool":
			if len(sanitized) == 0 {
				logger.DebugCF("agent", "Dropping orphaned leading tool message", map[string]any{})
				continue
			}
			// Walk backwards to find the nearest assistant message,
			// skipping over any preceding tool messages (multi-tool-call case).
			foundAssistant := false
			for i := len(sanitized) - 1; i >= 0; i-- {
				if sanitized[i].Role == "tool" {
					continue
				}
				if sanitized[i].Role == "assistant" && len(sanitized[i].ToolCalls) > 0 {
					foundAssistant = true
				}
				break
			}
			if !foundAssistant {
				logger.DebugCF("agent", "Dropping orphaned tool message", map[string]any{})
				continue
			}
			sanitized = append(sanitized, msg)

		case "assistant":
			if len(msg.ToolCalls) > 0 {
				if len(sanitized) == 0 {
					logger.DebugCF("agent", "Dropping assistant tool-call turn at history start", map[string]any{})
					continue
				}
				prev := sanitized[len(sanitized)-1]
				if prev.Role != "user" && prev.Role != "tool" {
					logger.DebugCF(
						"agent",
						"Dropping assistant tool-call turn with invalid predecessor",
						map[string]any{"prev_role": prev.Role},
					)
					continue
				}
			}
			sanitized = append(sanitized, msg)

		default:
			sanitized = append(sanitized, msg)
		}
	}

	// Second pass: ensure every assistant message with tool_calls has matching
	// tool result messages following it. This is required by strict providers
	// like DeepSeek that enforce: "An assistant message with 'tool_calls' must
	// be followed by tool messages responding to each 'tool_call_id'."
	final := make([]providers.Message, 0, len(sanitized))
	for i := 0; i < len(sanitized); i++ {
		msg := sanitized[i]
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Collect expected tool_call IDs
			expected := make(map[string]bool, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				expected[tc.ID] = false
			}

			// Check following messages for matching tool results
			toolMsgCount := 0
			for j := i + 1; j < len(sanitized); j++ {
				if sanitized[j].Role != "tool" {
					break
				}
				toolMsgCount++
				if _, exists := expected[sanitized[j].ToolCallID]; exists {
					expected[sanitized[j].ToolCallID] = true
				}
			}

			// If any tool_call_id is missing, drop this assistant message and its partial tool messages
			allFound := true
			for toolCallID, found := range expected {
				if !found {
					allFound = false
					logger.DebugCF(
						"agent",
						"Dropping assistant message with incomplete tool results",
						map[string]any{
							"missing_tool_call_id": toolCallID,
							"expected_count":       len(expected),
							"found_count":          toolMsgCount,
						},
					)
					break
				}
			}

			if !allFound {
				// Skip this assistant message and its tool messages
				i += toolMsgCount
				continue
			}
		}
		final = append(final, msg)
	}

	return final
}

// SetRetriever sets the RAG retriever for semantic memory search.
func (cb *ContextBuilder) SetRetriever(r *memory.Retriever) {
	cb.retriever = r
}

// retrieveRAGContext searches past interactions and formats them as context.
// Checks the RAG context cache first to avoid redundant embedding API calls.
func (cb *ContextBuilder) retrieveRAGContext(query string) string {
	// Check RAG context cache first
	if cb.ragCache != nil {
		if cached, ok := cb.ragCache.Get(query); ok {
			logger.DebugCF("rag", "RAG context cache hit", map[string]any{"query_len": len(query)})
			return cached
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := cb.retriever.Search(ctx, query)
	if err != nil {
		logger.WarnCF("rag", "RAG search failed", map[string]any{"error": err.Error()})
		return ""
	}

	formatted := memory.FormatContext(results)

	// Cache the result
	if cb.ragCache != nil && formatted != "" {
		cb.ragCache.Put(query, formatted)
	}

	return formatted
}

// compactToolHistory replaces old tool call pairs with compact summaries to save
// tokens. The last keepRecentMessages messages are kept intact for continuity;
// older assistant(tool_calls)+tool(result) groups are collapsed into a single
// assistant message like: "[tool: shell_exec({command:"ls"}) → 1245 chars]".
//
// This runs at read-time (BuildMessages) so the session file is never modified.
const keepRecentMessages = 10

func compactToolHistory(history []providers.Message) []providers.Message {
	if len(history) <= keepRecentMessages {
		return history
	}

	// Split into old (compactable) and recent (kept verbatim).
	cutoff := len(history) - keepRecentMessages
	old := history[:cutoff]
	recent := history[cutoff:]

	compacted := make([]providers.Message, 0, len(history))
	savedChars := 0

	for i := 0; i < len(old); i++ {
		msg := old[i]

		// Not an assistant message with tool calls → keep as-is
		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			compacted = append(compacted, msg)
			continue
		}

		// Collect the tool result messages that follow this assistant message
		toolResults := make(map[string]string) // toolCallID → content
		j := i + 1
		for j < len(old) && old[j].Role == "tool" {
			toolResults[old[j].ToolCallID] = old[j].Content
			j++
		}

		// Build compact summary
		var sb strings.Builder
		for _, tc := range msg.ToolCalls {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}

			// Compact tool call args
			args := ""
			if tc.Function != nil && tc.Function.Arguments != "" {
				args = tc.Function.Arguments
				if len(args) > 80 {
					args = args[:80] + "..."
				}
			}

			// Compact tool result
			result := toolResults[tc.ID]
			resultLen := len(result)
			resultPreview := ""
			if resultLen > 0 {
				preview := result
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				// Escape newlines for single-line display
				preview = strings.ReplaceAll(preview, "\n", " ")
				resultPreview = fmt.Sprintf(" → %s (%d chars)", preview, resultLen)
			}

			fmt.Fprintf(&sb, "[tool: %s(%s)%s]", tc.Name, args, resultPreview)
		}

		// Count savings
		origChars := len(msg.Content)
		for _, r := range toolResults {
			origChars += len(r)
		}
		savedChars += origChars - sb.Len()

		// Replace with single compacted assistant message (no tool calls)
		compacted = append(compacted, providers.Message{
			Role:    "assistant",
			Content: sb.String(),
		})

		// Skip the tool result messages we already consumed
		i = j - 1
	}

	if savedChars > 0 {
		logger.DebugCF("agent", "Tool history compacted", map[string]any{
			"old_messages":  len(old),
			"compacted_to":  len(compacted),
			"chars_saved":   savedChars,
			"recent_kept":   len(recent),
		})
	}

	compacted = append(compacted, recent...)
	return compacted
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]any {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]any{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
