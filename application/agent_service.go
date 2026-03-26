package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/renesul/ok/application/engine"
	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

type AgentService struct {
	db               *sql.DB
	llmClient        *llm.Client
	llmHeavyConfig   llm.ClientConfig
	llmFastConfig    llm.ClientConfig
	planner          domain.Planner
	executor         domain.Executor
	memory           *agentpkg.SQLiteMemory
	execRepo         *agentpkg.ExecutionRepository
	configRepo       *agentpkg.ConfigRepository
	soul             string
	identity         string
	userProfile      string
	environmentNotes string
	limits           domain.AgentLimits
	cachedPrompt       string
	promptMu           sync.RWMutex
	globalEventHandler func(domain.AgentEvent)
	globalMu           sync.RWMutex
	skillRepo          domain.SkillRepository
	useNativeTools     bool
	log                *zap.Logger
}

func NewAgentService(
	db *sql.DB,
	llmClient *llm.Client,
	llmHeavyConfig llm.ClientConfig,
	llmFastConfig llm.ClientConfig,
	planner domain.Planner,
	executor domain.Executor,
	memory *agentpkg.SQLiteMemory,
	execRepo *agentpkg.ExecutionRepository,
	configRepo *agentpkg.ConfigRepository,
	skillRepo domain.SkillRepository,
	useNativeTools bool,
	log *zap.Logger,
) *AgentService {
	s := &AgentService{
		db:             db,
		llmClient:      llmClient,
		llmHeavyConfig: llmHeavyConfig,
		llmFastConfig:  llmFastConfig,
		planner:        planner,
		executor:       executor,
		memory:         memory,
		execRepo:       execRepo,
		configRepo:     configRepo,
		skillRepo:      skillRepo,
		useNativeTools: useNativeTools,
		log:            log.Named("service.agent"),
	}
	s.loadTemplates()
	return s
}

// SetGlobalEventHandler registers a handler that receives ALL agent events
// from every channel (chat, WebSocket, adapters). Used by WSHandler to broadcast.
func (s *AgentService) SetGlobalEventHandler(fn func(domain.AgentEvent)) {
	s.globalMu.Lock()
	s.globalEventHandler = fn
	s.globalMu.Unlock()
}

func (s *AgentService) emitGlobal(e domain.AgentEvent) {
	s.globalMu.RLock()
	fn := s.globalEventHandler
	s.globalMu.RUnlock()
	if fn != nil {
		fn(e)
	}
}

func (s *AgentService) Run(ctx context.Context, input string) (domain.AgentResponse, error) {
	s.log.Debug("agent start", zap.String("input", input))
	if !s.autoForgetRule(input) {
		s.autoLearnRule(input)
	}
	eng := s.buildEngine()
	buf := engine.NewBufferEmitter()
	emitter := engine.NewCallbackEmitter(func(e domain.AgentEvent) {
		buf.Forward(e)
		s.emitGlobal(e)
	})
	if err := eng.RunLoop(ctx, input, emitter); err != nil {
		return domain.AgentResponse{}, err
	}
	return buf.Response(), nil
}

func (s *AgentService) RunStream(ctx context.Context, input string, onEvent domain.EventCallback) error {
	s.log.Debug("agent stream start", zap.String("input", input))
	if !s.autoForgetRule(input) {
		s.autoLearnRule(input)
	}
	emit := func(e domain.AgentEvent) {
		if onEvent != nil {
			onEvent(e)
		}
		s.emitGlobal(e)
	}
	eng := s.buildEngine()
	emitter := engine.NewCallbackEmitter(emit)
	return eng.RunLoop(ctx, input, emitter)
}

// rulePatterns detects user intent to save a permanent rule.
var rulePatterns = []string{
	"from now on", "always ", "never ", "remember that", "remember this",
	"memorize", "learn this", "learn that", "de agora em diante",
	"sempre ", "nunca ", "lembre", "memorize", "aprenda",
}

var forgetPatterns = []string{
	"forget ", "forget that", "remove rule", "delete rule",
	"stop doing", "esqueça", "esqueca", "remova regra", "apague regra",
}

func (s *AgentService) autoForgetRule(input string) bool {
	if s.memory == nil {
		return false
	}
	lower := strings.ToLower(input)
	for _, pattern := range forgetPatterns {
		if strings.Contains(lower, pattern) {
			idx := strings.Index(lower, pattern)
			keyword := strings.TrimSpace(input[idx+len(pattern):])
			if keyword == "" {
				return true
			}
			deleted, err := s.memory.DeleteRulesByContent(keyword)
			if err != nil {
				s.log.Debug("auto-forget rule failed", zap.Error(err))
			} else if deleted > 0 {
				s.log.Debug("auto-forgot rules", zap.Int64("deleted", deleted), zap.String("keyword", keyword))
				s.invalidatePromptCache()
			}
			return true
		}
	}
	return false
}

func (s *AgentService) autoLearnRule(input string) {
	if s.memory == nil {
		return
	}
	lower := strings.ToLower(input)
	for _, pattern := range rulePatterns {
		if strings.Contains(lower, pattern) {
			if err := s.memory.Save(domain.MemoryEntry{
				Content:  input,
				Category: "rule",
			}); err != nil {
				s.log.Debug("auto-learn rule failed", zap.Error(err))
			} else {
				s.log.Debug("auto-learned rule from input", zap.String("input", input))
				s.invalidatePromptCache()
			}
			return
		}
	}
}

func (s *AgentService) invalidatePromptCache() {
	s.promptMu.Lock()
	s.cachedPrompt = ""
	s.promptMu.Unlock()
}

func (s *AgentService) buildEngine() *engine.AgentEngine {
	return engine.NewAgentEngine(
		s.db, s.llmClient, s.llmHeavyConfig, s.llmFastConfig, s.planner, s.executor,
		s.memory, s.execRepo,
		s.limits, s.useNativeTools, s.BuildSystemPrompt, s.log,
	)
}

func (s *AgentService) loadTemplates() {
	if s.configRepo == nil {
		return
	}
	ctx := context.Background()
	if v, err := s.configRepo.Get(ctx, "soul"); err == nil && v != "" {
		s.soul = v
	}
	if v, err := s.configRepo.Get(ctx, "identity"); err == nil && v != "" {
		s.identity = v
	}
	if v, err := s.configRepo.Get(ctx, "user_profile"); err == nil && v != "" {
		s.userProfile = v
	}
	if v, err := s.configRepo.Get(ctx, "environment_notes"); err == nil && v != "" {
		s.environmentNotes = v
	}
	s.limits = domain.DefaultAgentLimits()
	if v, err := s.configRepo.Get(ctx, "agent_limits"); err == nil && v != "" {
		var limits domain.AgentLimits
		if jsonErr := json.Unmarshal([]byte(v), &limits); jsonErr == nil {
			s.limits = limits
		}
	}
}

func (s *AgentService) ReloadSoul() {
	s.loadTemplates()
	s.invalidatePromptCache()
}

func (s *AgentService) ListSkills() []map[string]string {
	if s.skillRepo == nil {
		return nil
	}
	skills, err := s.skillRepo.List()
	if err != nil {
		return nil
	}
	var result []map[string]string
	for _, sk := range skills {
		result = append(result, map[string]string{
			"name":        sk.Name,
			"description": sk.Description,
		})
	}
	return result
}

func (s *AgentService) ListTools() []map[string]string {
	tools := s.planner.Tools()
	var result []map[string]string
	for _, tool := range tools {
		entry := map[string]string{
			"name":        tool.Name(),
			"description": tool.Description(),
		}
		if st, ok := tool.(domain.SafeTool); ok {
			entry["safety"] = string(st.Safety())
		}
		result = append(result, entry)
	}
	return result
}

func (s *AgentService) GetConfigRepo() *agentpkg.ConfigRepository {
	return s.configRepo
}

func (s *AgentService) BuildSystemPrompt() string {
	s.promptMu.RLock()
	if s.cachedPrompt != "" {
		cached := s.cachedPrompt
		s.promptMu.RUnlock()
		return cached
	}
	s.promptMu.RUnlock()

	now := time.Now().Format("Monday, 2 January 2006, 15:04")

	var parts []string

	soul := s.soul
	if soul == "" {
		soul = "You are a highly capable, direct, and intelligent personal autonomous agent functioning on the user's local machine."
	}
	parts = append(parts, soul)

	if s.identity != "" {
		parts = append(parts, s.identity)
	}
	if s.userProfile != "" {
		parts = append(parts, "About the user: "+s.userProfile)
	}
	if s.environmentNotes != "" {
		parts = append(parts, "Environment: "+s.environmentNotes)
	}

	if s.memory != nil {
		rules, _ := s.memory.SearchByCategory("", "rule", 20)
		if len(rules) > 0 {
			var ruleTexts []string
			for _, r := range rules {
				ruleTexts = append(ruleTexts, "- "+r.Content)
			}
			parts = append(parts, "Learned Rules (OBEY ALWAYS):\n"+strings.Join(ruleTexts, "\n"))
		}
	}

	if s.skillRepo != nil {
		skills, _ := s.skillRepo.List()
		if len(skills) > 0 {
			var skillDescs []string
			for _, sk := range skills {
				skillDescs = append(skillDescs, fmt.Sprintf("- %s: %s", sk.Name, sk.Description))
			}
			parts = append(parts, "[INSTALLED SKILLS] You have the following loadable skills:\n"+
				strings.Join(skillDescs, "\n")+
				"\nUse the 'skill_loader' tool to read a skill's rules BEFORE executing a task if needed.")
		}
	}

	parts = append(parts, "Current date and time: "+now)

	parts = append(parts, fmt.Sprintf(`Available Tools:
%s

TOOL SELECTION GUIDE (use the most specific tool for each request):
- Search internet / documentation / troubleshoot errors → web_search
- Open/navigate specific website → browser
- Search text within project files → search
- Read file (with pagination) → file_read
- Create/write new file → file_write
- Edit snippet in existing file → file_edit
- Execute JS/Python/Bash code → repl (language:"node"|"python"|"bash")
- Execute terminal commands / git / npm / tests → shell
- Fix bug → file_read + file_edit + shell (run tests)
- Write tests → file_read + file_write + shell
- Install packages → shell (npm install / pip install / go get)
- Commit/push code → shell (git add/commit/push)
- Calculate expressions → math
- Parse JSON → json_parse
- Schedule recurring tasks → schedule
- Make HTTP requests → http
- Convert base64 → base64
- Extract text from HTML → text_extract
- Read entire directory structure → folder_index
- Complex tasks (subdivide into parts) → delegate
- Memorize a rule or fact permanently → learn_rule
- Load skill instructions before executing a task → skill_loader

RULES:
- RESPECT EXACTLY what the user requested (language, format, tool, tone).
- If JavaScript is requested → repl with language:"node".
- If CSV is requested → output raw CSV, not JSON.
- If shell is requested → use shell, not repl.
- NEVER substitute the user's choice without asking.
- For normal conversation without required actions → respond directly with done=true.

To use a tool, reply EXACTLY with valid JSON ONLY:
{"thought":"your step-by-step reasoning and deduction", "tool":"name","input":"value","done":false}

To answer directly (no tool needed), reply EXACTLY with valid JSON ONLY:
{"thought":"your step-by-step reasoning and deduction", "tool":"","input":"your final answer","done":true}

IMPORTANT: ALWAYS respond in valid JSON format ONLY. No markdown wrapping.`, s.planner.ToolDescriptions()))

	prompt := joinNonEmpty(parts, "\n\n")

	s.promptMu.Lock()
	s.cachedPrompt = prompt
	s.promptMu.Unlock()

	go func() {
		time.Sleep(1 * time.Minute)
		s.promptMu.Lock()
		s.cachedPrompt = ""
		s.promptMu.Unlock()
	}()

	return prompt
}

func joinNonEmpty(parts []string, sep string) string {
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, sep)
}

func (s *AgentService) GetExecution(id string) (*domain.ExecutionRecord, error) {
	if s.execRepo == nil {
		return nil, nil
	}
	return s.execRepo.FindByID(id)
}

func (s *AgentService) GetRecentExecutions(limit int) ([]domain.ExecutionRecord, error) {
	if s.execRepo == nil {
		return nil, nil
	}
	return s.execRepo.FindRecent(limit)
}

func (s *AgentService) GetLimits() domain.AgentLimits {
	return s.limits
}

func (s *AgentService) SetLimits(ctx context.Context, limits domain.AgentLimits) error {
	data, err := json.Marshal(limits)
	if err != nil {
		return fmt.Errorf("marshal limits: %w", err)
	}
	if err := s.configRepo.Set(ctx, "agent_limits", string(data)); err != nil {
		return fmt.Errorf("save limits: %w", err)
	}
	s.limits = limits
	return nil
}

func (s *AgentService) GetMetrics() (*domain.ExecutionMetrics, error) {
	if s.execRepo == nil {
		return nil, nil
	}
	return s.execRepo.GetMetrics()
}
