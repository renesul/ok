package repository

import (
	"database/sql"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/renesul/ok/infrastructure/database"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Usamos memória em modo compartilhado para permitir múltiplas conexões vislumbrirem a mesma base
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open memory sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Limpar tabelas caso a memória compartilhada vaze entre testes paralelos
	tables := []string{
		"messages_fts",
		"message_embeddings",
		"messages",
		"sessions",
		"conversations",
	}
	for _, tbl := range tables {
		db.Exec("DELETE FROM " + tbl)
	}

	return db
}
