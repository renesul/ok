package agent

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/embedding"
	"github.com/renesul/ok/infrastructure/llm"
	"github.com/renesul/ok/infrastructure/security"
	"go.uber.org/zap"
)

const maxContentLength = 4000

type SQLiteMemory struct {
	db       *sql.DB
	embed    *embedding.Client
	vector   *VectorStore
	scrubber *security.SecretScrubber
	log      *zap.Logger
}

func NewSQLiteMemory(db *sql.DB, log *zap.Logger) *SQLiteMemory {
	return &SQLiteMemory{
		db:  db,
		log: log.Named("agent.memory"),
	}
}

// SetEmbeddingClient configura o client de embeddings (opcional)
func (m *SQLiteMemory) SetEmbeddingClient(client *embedding.Client) {
	m.embed = client
}

// SetScrubber configura o scrubber de segredos (opcional)
func (m *SQLiteMemory) SetScrubber(s *security.SecretScrubber) {
	m.scrubber = s
}

// SetVectorStore configura o vector store para busca semantica (opcional)
func (m *SQLiteMemory) SetVectorStore(vs *VectorStore) {
	m.vector = vs
}

// DB expoe o banco para uso em transacoes externas
func (m *SQLiteMemory) DB() *sql.DB {
	return m.db
}

func (m *SQLiteMemory) Save(entry domain.MemoryEntry) error {
	return m.SaveInTx(m.db, entry)
}

// SaveInTx salva uma memory entry usando uma transacao existente
func (m *SQLiteMemory) SaveInTx(tx database.Execer, entry domain.MemoryEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	category := entry.Category
	if category == "" {
		category = "fact"
	}

	// Scrub segredos antes de persistir
	content := entry.Content
	if m.scrubber != nil {
		content = m.scrubber.Scrub(content)
	}

	_, err := tx.ExecContext(context.Background(),
		"INSERT INTO agent_memory (id, content, category, created_at) VALUES (?, ?, ?, ?)",
		entry.ID, content, category, entry.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Best-effort: indexar no vector store (com conteudo ja scrubbed)
	if m.vector != nil {
		m.vector.Save(entry.ID, content, category)
	}
	return nil
}

// SaveChunked salva conteudo grande dividindo em chunks semanticos
func (m *SQLiteMemory) SaveChunked(entry domain.MemoryEntry) error {
	if len(entry.Content) <= maxContentLength {
		return m.Save(entry)
	}

	chunks := splitChunks(entry.Content, maxContentLength)
	baseID := uuid.New().String()
	now := time.Now()

	category := entry.Category
	if category == "" {
		category = "fact"
	}

	for i, chunk := range chunks {
		chunkEntry := domain.MemoryEntry{
			ID:        fmt.Sprintf("%s_chunk_%d", baseID, i),
			Content:   chunk,
			Category:  category,
			CreatedAt: now,
		}
		if err := m.Save(chunkEntry); err != nil {
			return fmt.Errorf("save chunk %d: %w", i, err)
		}
	}

	m.log.Debug("saved chunked memory", zap.String("base_id", baseID), zap.Int("chunks", len(chunks)), zap.Int("total_len", len(entry.Content)))
	return nil
}

// SaveChunkedInTx salva conteudo grande em chunks dentro de uma transacao
func (m *SQLiteMemory) SaveChunkedInTx(tx database.Execer, entry domain.MemoryEntry) error {
	if len(entry.Content) <= maxContentLength {
		return m.SaveInTx(tx, entry)
	}

	chunks := splitChunks(entry.Content, maxContentLength)
	baseID := uuid.New().String()
	now := time.Now()

	category := entry.Category
	if category == "" {
		category = "fact"
	}

	for i, chunk := range chunks {
		chunkEntry := domain.MemoryEntry{
			ID:        fmt.Sprintf("%s_chunk_%d", baseID, i),
			Content:   chunk,
			Category:  category,
			CreatedAt: now,
		}
		if err := m.SaveInTx(tx, chunkEntry); err != nil {
			return fmt.Errorf("save chunk %d: %w", i, err)
		}
	}

	return nil
}

// splitChunks divide texto em chunks respeitando limites semanticos
func splitChunks(content string, maxLen int) []string {
	if len(content) <= maxLen {
		return []string{content}
	}

	var chunks []string

	// Tentar dividir por paragrafos
	paragraphs := strings.Split(content, "\n\n")
	var current strings.Builder

	for _, para := range paragraphs {
		if current.Len() > 0 && current.Len()+len(para)+2 > maxLen {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if len(para) > maxLen {
			// Paragrafo grande: dividir por linhas
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
			chunks = append(chunks, splitByLines(para, maxLen)...)
			continue
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

func splitByLines(text string, maxLen int) []string {
	lines := strings.Split(text, "\n")
	var chunks []string
	var current strings.Builder

	for _, line := range lines {
		if current.Len() > 0 && current.Len()+len(line)+1 > maxLen {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if len(line) > maxLen {
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
			// Hard cut respeitando UTF-8
			for len(line) > maxLen {
				cut := TruncateUTF8(line, maxLen)
				chunks = append(chunks, cut)
				line = line[len(cut):]
			}
			if len(line) > 0 {
				current.WriteString(line)
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

func (m *SQLiteMemory) Search(query string, limit int) ([]domain.MemoryEntry, error) {
	if query == "" || limit <= 0 {
		return nil, nil
	}

	m.log.Debug("search memory", zap.String("query", query), zap.Int("limit", limit))

	ftsQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
	rows, err := m.db.Query(
		`SELECT m.id, m.content, COALESCE(m.category, 'fact') as category, m.created_at 
		 FROM agent_memory m 
		 JOIN agent_memory_fts f ON m.rowid = f.rowid 
		 WHERE f.content MATCH ? 
		 ORDER BY m.created_at DESC LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search memory: %w", err)
	}
	defer rows.Close()

	var entries []domain.MemoryEntry
	for rows.Next() {
		var e domain.MemoryEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.Category, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory row: %w", err)
		}
		entries = append(entries, e)
	}

	m.log.Debug("memory results", zap.Int("count", len(entries)))
	return entries, rows.Err()
}

// SearchByCategory busca memorias filtradas por categoria
func (m *SQLiteMemory) SearchByCategory(query string, category string, limit int) ([]domain.MemoryEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	m.log.Debug("search memory by category", zap.String("query", query), zap.String("category", category), zap.Int("limit", limit))

	var q string
	var args []interface{}

	if query != "" {
		ftsQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
		q = `SELECT m.id, m.content, COALESCE(m.category, 'fact') as category, m.created_at 
			 FROM agent_memory m 
			 JOIN agent_memory_fts f ON m.rowid = f.rowid 
			 WHERE COALESCE(m.category, 'fact') = ? AND f.content MATCH ? 
			 ORDER BY m.created_at DESC LIMIT ?`
		args = []interface{}{category, ftsQuery, limit}
	} else {
		q = `SELECT id, content, COALESCE(category, 'fact') as category, created_at 
			 FROM agent_memory 
			 WHERE COALESCE(category, 'fact') = ? 
			 ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{category, limit}
	}

	rows, err := m.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search memory by category: %w", err)
	}
	defer rows.Close()

	var entries []domain.MemoryEntry
	for rows.Next() {
		var e domain.MemoryEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.Category, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory row: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func (m *SQLiteMemory) Recent(limit int) ([]domain.MemoryEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	m.log.Debug("recent memories", zap.Int("limit", limit))

	rows, err := m.db.Query(
		"SELECT id, content, COALESCE(category, 'fact') as category, created_at FROM agent_memory ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent memories: %w", err)
	}
	defer rows.Close()

	var entries []domain.MemoryEntry
	for rows.Next() {
		var e domain.MemoryEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.Category, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory row: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func (m *SQLiteMemory) DeleteRulesByContent(keyword string) (int64, error) {
	result, err := m.db.Exec(
		"DELETE FROM agent_memory WHERE category = 'rule' AND LOWER(content) LIKE ?",
		"%"+strings.ToLower(keyword)+"%",
	)
	if err != nil {
		return 0, fmt.Errorf("delete rules: %w", err)
	}
	return result.RowsAffected()
}

// SearchSemantic busca memorias usando vector store ou fallback LIKE
func (m *SQLiteMemory) SearchSemantic(ctx context.Context, query string, limit int) ([]domain.MemoryEntry, error) {
	ctx = database.Ctx(ctx)
	if m.vector != nil {
		results, err := m.vector.Search(query, limit)
		if err == nil && len(results) > 0 {
			m.log.Debug("semantic search via vectorstore", zap.Int("results", len(results)))
			return results, nil
		}
		m.log.Debug("vectorstore search failed, fallback to LIKE", zap.Error(err))
	}
	return m.Search(query, limit)
}

const condensePrompt = `You receive a history of past interactions from an AI assistant.
Extract ONLY lessons learned, user preferences and relevant facts about the project.
Ignore greetings, corrected errors and trivial interactions.
Return ONE concise paragraph.`

const minCondenseCount = 10

// CondenseOldMemories comprime memorias antigas em sinteses via LLM
func (m *SQLiteMemory) CondenseOldMemories(ctx context.Context, llmClient *llm.Client, llmConfig llm.ClientConfig) error {
	ctx = database.Ctx(ctx)
	if llmConfig.BaseURL == "" {
		return nil
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	rows, err := m.db.QueryContext(ctx,
		"SELECT id, content, COALESCE(category, 'fact') as category, created_at FROM agent_memory WHERE COALESCE(category, 'fact') IN ('fact', '') AND created_at < ? ORDER BY created_at ASC LIMIT 30",
		cutoff,
	)
	if err != nil {
		return fmt.Errorf("query old memories: %w", err)
	}
	defer rows.Close()

	var memRows []struct {
		ID, Content, Category string
		CreatedAt             time.Time
	}
	for rows.Next() {
		var r struct {
			ID, Content, Category string
			CreatedAt             time.Time
		}
		if err := rows.Scan(&r.ID, &r.Content, &r.Category, &r.CreatedAt); err != nil {
			return fmt.Errorf("scan old memory: %w", err)
		}
		memRows = append(memRows, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate old memories: %w", err)
	}

	if len(memRows) < minCondenseCount {
		m.log.Debug("condense skipped: not enough old memories", zap.Int("count", len(memRows)))
		return nil
	}

	// Juntar conteudo
	var contents []string
	var ids []string
	for _, row := range memRows {
		contents = append(contents, row.Content)
		ids = append(ids, row.ID)
	}
	batch := strings.Join(contents, "\n---\n")

	// Chamar LLM para condensar
	cfg := llmConfig
	cfg.MaxTokens = 300
	cfg.Temperature = 0.2

	messages := []llm.Message{
		{Role: "system", Content: condensePrompt},
		{Role: "user", Content: batch},
	}

	synthesis, err := llmClient.ChatCompletionSync(ctx, cfg, messages)
	if err != nil {
		return fmt.Errorf("condense llm call: %w", err)
	}

	if strings.TrimSpace(synthesis) == "" {
		m.log.Debug("condense: LLM returned empty synthesis")
		return nil
	}

	// Transacao atomica: delete velhas + insert sintese
	return database.WithTx(m.db, ctx, func(tx *sql.Tx) error {
		// Delete velhas do SQLite — build IN (?, ?, ...) dynamically
		placeholders := strings.Repeat("?,", len(ids))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, len(ids))
		for i, id := range ids {
			args[i] = id
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM agent_memory WHERE id IN ("+placeholders+")", args...); err != nil {
			return fmt.Errorf("delete old memories: %w", err)
		}

		// Delete do vector store
		if m.vector != nil {
			m.vector.Delete(ids...)
		}

		// Insert sintese
		entry := domain.MemoryEntry{
			ID:       uuid.New().String(),
			Content:  synthesis,
			Category: "synthesis",
		}
		if err := m.SaveInTx(tx, entry); err != nil {
			return fmt.Errorf("save synthesis: %w", err)
		}

		m.log.Debug("condense completed",
			zap.Int("condensed", len(ids)),
			zap.Int("synthesis_len", len(synthesis)),
		)
		return nil
	})
}

// ShouldStore decide se uma interacao merece ser salva na memoria
func ShouldStore(input, output string) bool {
	if output == "" {
		return false
	}
	if len(input)+len(output) < 10 {
		return false
	}
	// Filtrar apenas outputs que sao erros de tools (prefixo), nao conteudo que menciona erros
	lowerOutput := strings.ToLower(output)
	errorPrefixes := []string{"error:", "erro:", "timeout:", "command failed:", "execution error:"}
	for _, prefix := range errorPrefixes {
		if strings.HasPrefix(lowerOutput, prefix) {
			return false
		}
	}
	return true
}
