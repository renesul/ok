package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDatabase_NewAndMigrations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ok_test_db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := New(dbPath, false)
	if err != nil {
		t.Fatalf("failed to create new database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations on fresh database: %v", err)
	}

	// Double-check if tables were created
	var count int
	err = db.QueryRow("SELECT count() FROM sqlite_master WHERE type='table' AND name IN ('conversations', 'messages', 'messages_fts')").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check tables: %v", err)
	}
	if count < 3 {
		t.Fatalf("expected at least 3 tables to be created by migrations, got %d", count)
	}
}
