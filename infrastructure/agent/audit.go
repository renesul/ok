package agent

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuditEntry — registro de execucao de tool para auditoria
type AuditEntry struct {
	ID        string    `json:"id"`
	Tool      string    `json:"tool"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Safety    string    `json:"safety"`
	Approved  bool      `json:"approved"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLog — registra todas as execucoes de tools
type AuditLog struct {
	db  *sql.DB
	log *zap.Logger
}

// NewAuditLog cria um audit log
func NewAuditLog(db *sql.DB, log *zap.Logger) *AuditLog {
	return &AuditLog{
		db:  db,
		log: log.Named("agent.audit"),
	}
}

// Record salva uma entrada de auditoria
func (a *AuditLog) Record(entry AuditEntry) {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	// Truncar output grande
	entry.Output = TruncateWithEllipsis(entry.Output, 500)

	_, err := a.db.Exec(
		"INSERT INTO agent_audit (id, tool, input, output, safety, approved, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		entry.ID, entry.Tool, entry.Input, entry.Output, entry.Safety, entry.Approved, entry.CreatedAt,
	)
	if err != nil {
		a.log.Debug("audit record failed", zap.Error(err))
	}
}
