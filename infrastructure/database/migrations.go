package database

import (
	"fmt"

	"github.com/renesul/ok/domain"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&domain.Session{},
		&domain.Conversation{},
		&domain.Message{},
	); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	fts5SQL := `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		conversation_id UNINDEXED,
		content,
		tokenize='unicode61'
	)`
	if err := db.Exec(fts5SQL).Error; err != nil {
		return fmt.Errorf("create fts5 table: %w", err)
	}

	embeddingsSQL := `CREATE TABLE IF NOT EXISTS message_embeddings (
		message_id INTEGER PRIMARY KEY,
		conversation_id INTEGER NOT NULL,
		embedding BLOB NOT NULL
	)`
	if err := db.Exec(embeddingsSQL).Error; err != nil {
		return fmt.Errorf("create embeddings table: %w", err)
	}

	memorySQL := `CREATE TABLE IF NOT EXISTS agent_memory (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL
	)`
	if err := db.Exec(memorySQL).Error; err != nil {
		return fmt.Errorf("create agent_memory table: %w", err)
	}

	// Memory category column
	db.Exec("ALTER TABLE agent_memory ADD COLUMN category TEXT DEFAULT 'fact'")

	// Memory embedding column (vector search)
	db.Exec("ALTER TABLE agent_memory ADD COLUMN embedding BLOB DEFAULT NULL")

	schedulerSQL := `CREATE TABLE IF NOT EXISTS scheduled_jobs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		task_type TEXT NOT NULL,
		input TEXT NOT NULL,
		interval_seconds INTEGER NOT NULL,
		enabled INTEGER DEFAULT 1,
		last_run DATETIME,
		last_status TEXT DEFAULT '',
		fail_count INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL
	)`
	if err := db.Exec(schedulerSQL).Error; err != nil {
		return fmt.Errorf("create scheduled_jobs table: %w", err)
	}

	feedbackSQL := `CREATE TABLE IF NOT EXISTS agent_feedback (
		id TEXT PRIMARY KEY,
		tool_name TEXT NOT NULL,
		task_type TEXT DEFAULT '',
		success INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		error TEXT DEFAULT '',
		created_at DATETIME NOT NULL
	)`
	if err := db.Exec(feedbackSQL).Error; err != nil {
		return fmt.Errorf("create agent_feedback table: %w", err)
	}
	db.Exec("CREATE INDEX IF NOT EXISTS idx_feedback_tool ON agent_feedback(tool_name)")
	db.Exec("ALTER TABLE agent_feedback ADD COLUMN cost INTEGER DEFAULT 1")

	configSQL := `CREATE TABLE IF NOT EXISTS agent_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`
	if err := db.Exec(configSQL).Error; err != nil {
		return fmt.Errorf("create agent_config table: %w", err)
	}

	executionsSQL := `CREATE TABLE IF NOT EXISTS agent_executions (
		id TEXT PRIMARY KEY,
		goal TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'done',
		steps TEXT,
		timeline TEXT,
		total_ms INTEGER DEFAULT 0,
		step_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if err := db.Exec(executionsSQL).Error; err != nil {
		return fmt.Errorf("create agent_executions table: %w", err)
	}

	// Seed templates
	db.Exec(`INSERT OR IGNORE INTO agent_config (key, value) VALUES ('soul', 'Voce e OK, um assistente pessoal inteligente e confiavel.

Principios:
- Seja genuinamente util. Nada de "Otima pergunta!" — va direto ao ponto.
- Tenha opiniao. Assistentes sem personalidade nao ajudam. Se te perguntarem uma preferencia, responda com honestidade.
- Resolva antes de perguntar. Esgote os recursos disponiveis antes de pedir ajuda. Traga solucoes, nao so perguntas.
- Conquiste confianca pela competencia. Mostre que sabe o que faz.

Limites:
- Assuntos confidenciais sao confidenciais, sem excecao.
- Acoes externas (emails, posts) precisam de aprovacao quando houver duvida.
- Nunca invente informacoes. Se nao sabe, diga.

Estilo:
- Pratico, direto e autentico.
- Nem corporativo nem bajulador.
- Conciso: se pode dizer em uma frase, nao use tres.
- Responde no idioma do usuario.

Use ferramentas APENAS quando necessario para tarefas praticas. Para conversas normais, responda naturalmente.')`)

	db.Exec(`INSERT OR IGNORE INTO agent_config (key, value) VALUES ('identity', 'Nome: OK
Tipo: assistente pessoal de IA
Estilo: direto, profissional, com um toque de humor seco
Especialidades: automacao, pesquisa, organizacao, analise de dados
Emoji: ⚡')`)

	db.Exec(`INSERT OR IGNORE INTO agent_config (key, value) VALUES ('user_profile', 'Arquiteto de software e entusiasta de IA.
Prefere solucoes simples e pragmaticas.
Valoriza clareza acima de tudo.
Trabalha principalmente com Go, sistemas distribuidos e automacao.
Idioma principal: portugues brasileiro.
Fuso horario: America/Sao_Paulo (GMT-3).')`)

	db.Exec(`INSERT OR IGNORE INTO agent_config (key, value) VALUES ('environment_notes', 'Sistema operacional: Linux (Debian)
Linguagem principal: Go
Banco de dados: SQLite (modernc, sem CGO)
Servidor web: Fiber
Diretorio do projeto: ~/Documentos/ok
Sandbox de arquivos: data/sandbox
Integracao LLM: OpenAI API (gpt-4.1-mini)
Embedding: text-embedding-3-small')`)

	// Execution metrics columns
	db.Exec("ALTER TABLE agent_executions ADD COLUMN tools_used TEXT DEFAULT ''")
	db.Exec("ALTER TABLE agent_executions ADD COLUMN failure_reason TEXT DEFAULT ''")

	// Default agent limits
	db.Exec(`INSERT OR IGNORE INTO agent_config (key, value) VALUES ('agent_limits', '{"max_steps":6,"max_attempts":4,"timeout_ms":120000}')`)

	// Audit table
	auditSQL := `CREATE TABLE IF NOT EXISTS agent_audit (
		id TEXT PRIMARY KEY,
		tool TEXT NOT NULL,
		input TEXT NOT NULL,
		output TEXT DEFAULT '',
		safety TEXT NOT NULL DEFAULT 'safe',
		approved INTEGER DEFAULT 1,
		created_at DATETIME NOT NULL
	)`
	if err := db.Exec(auditSQL).Error; err != nil {
		return fmt.Errorf("create agent_audit table: %w", err)
	}
	db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_tool ON agent_audit(tool)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_created ON agent_audit(created_at)")


	return nil
}
