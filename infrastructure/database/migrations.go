package database

import (
	"database/sql"
	"fmt"
	"strings"
)

func RunMigrations(db *sql.DB) error {
	// Core domain tables (replaces GORM AutoMigrate)
	sessionsSQL := `CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	)`
	if _, err := db.Exec(sessionsSQL); err != nil {
		return fmt.Errorf("create sessions table: %w", err)
	}

	conversationsSQL := `CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT 'import',
		channel TEXT NOT NULL DEFAULT 'web',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`
	if _, err := db.Exec(conversationsSQL); err != nil {
		return fmt.Errorf("create conversations table: %w", err)
	}

	messagesSQL := `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id INTEGER NOT NULL,
		role TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	)`
	if _, err := db.Exec(messagesSQL); err != nil {
		return fmt.Errorf("create messages table: %w", err)
	}
	execIgnore(db, "CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id)")

	// FTS5
	fts5SQL := `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		conversation_id UNINDEXED,
		content,
		tokenize='unicode61'
	)`
	if _, err := db.Exec(fts5SQL); err != nil {
		return fmt.Errorf("create fts5 table: %w", err)
	}

	// Embeddings
	embeddingsSQL := `CREATE TABLE IF NOT EXISTS message_embeddings (
		message_id INTEGER PRIMARY KEY,
		conversation_id INTEGER NOT NULL,
		embedding BLOB NOT NULL
	)`
	if _, err := db.Exec(embeddingsSQL); err != nil {
		return fmt.Errorf("create embeddings table: %w", err)
	}

	// Agent memory
	memorySQL := `CREATE TABLE IF NOT EXISTS agent_memory (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL
	)`
	if _, err := db.Exec(memorySQL); err != nil {
		return fmt.Errorf("create agent_memory table: %w", err)
	}
	addColumnIfNotExists(db, "ALTER TABLE agent_memory ADD COLUMN category TEXT DEFAULT 'fact'")
	addColumnIfNotExists(db, "ALTER TABLE agent_memory ADD COLUMN embedding BLOB DEFAULT NULL")

	// FTS5 for agent_memory
	memoryFtsSQL := `CREATE VIRTUAL TABLE IF NOT EXISTS agent_memory_fts USING fts5(
		content,
		category UNINDEXED,
		tokenize='unicode61'
	)`
	if _, err := db.Exec(memoryFtsSQL); err != nil {
		return fmt.Errorf("create fts5 memory table: %w", err)
	}

	// Triggers to keep FTS5 in sync with agent_memory
	db.Exec(`CREATE TRIGGER IF NOT EXISTS agent_memory_ai AFTER INSERT ON agent_memory BEGIN
		INSERT INTO agent_memory_fts(rowid, content, category) VALUES (new.rowid, new.content, new.category);
	END;`)
	db.Exec(`CREATE TRIGGER IF NOT EXISTS agent_memory_ad AFTER DELETE ON agent_memory BEGIN
		INSERT INTO agent_memory_fts(agent_memory_fts, rowid, content, category) VALUES('delete', old.rowid, old.content, old.category);
	END;`)
	db.Exec(`CREATE TRIGGER IF NOT EXISTS agent_memory_au AFTER UPDATE ON agent_memory BEGIN
		INSERT INTO agent_memory_fts(agent_memory_fts, rowid, content, category) VALUES('delete', old.rowid, old.content, old.category);
		INSERT INTO agent_memory_fts(rowid, content, category) VALUES (new.rowid, new.content, new.category);
	END;`)

	// Backfill existing data
	db.Exec(`INSERT INTO agent_memory_fts(rowid, content, category)
		SELECT rowid, content, category FROM agent_memory 
		WHERE rowid NOT IN (SELECT rowid FROM agent_memory_fts);`)

	// Scheduler
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
	if _, err := db.Exec(schedulerSQL); err != nil {
		return fmt.Errorf("create scheduled_jobs table: %w", err)
	}

	// Feedback
	feedbackSQL := `CREATE TABLE IF NOT EXISTS agent_feedback (
		id TEXT PRIMARY KEY,
		tool_name TEXT NOT NULL,
		task_type TEXT DEFAULT '',
		success INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		error TEXT DEFAULT '',
		created_at DATETIME NOT NULL
	)`
	if _, err := db.Exec(feedbackSQL); err != nil {
		return fmt.Errorf("create agent_feedback table: %w", err)
	}
	execIgnore(db, "CREATE INDEX IF NOT EXISTS idx_feedback_tool ON agent_feedback(tool_name)")
	addColumnIfNotExists(db, "ALTER TABLE agent_feedback ADD COLUMN cost INTEGER DEFAULT 1")

	// Config
	configSQL := `CREATE TABLE IF NOT EXISTS agent_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`
	if _, err := db.Exec(configSQL); err != nil {
		return fmt.Errorf("create agent_config table: %w", err)
	}

	// Executions
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
	if _, err := db.Exec(executionsSQL); err != nil {
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
	addColumnIfNotExists(db, "ALTER TABLE agent_executions ADD COLUMN tools_used TEXT DEFAULT ''")
	addColumnIfNotExists(db, "ALTER TABLE agent_executions ADD COLUMN failure_reason TEXT DEFAULT ''")

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
	if _, err := db.Exec(auditSQL); err != nil {
		return fmt.Errorf("create agent_audit table: %w", err)
	}
	execIgnore(db, "CREATE INDEX IF NOT EXISTS idx_audit_tool ON agent_audit(tool)")
	execIgnore(db, "CREATE INDEX IF NOT EXISTS idx_audit_created ON agent_audit(created_at)")

	return nil
}

func addColumnIfNotExists(db *sql.DB, query string) {
	_, err := db.Exec(query)
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		// Silently ignore duplicate column errors
	}
	_ = err
}

func execIgnore(db *sql.DB, query string) {
	db.Exec(query)
}
