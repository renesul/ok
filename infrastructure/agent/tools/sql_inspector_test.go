package tools

import (
	"strings"
	"testing"
)

func TestSqlInspector_EmptyQuery(t *testing.T) {
	tool := NewSqlInspectorTool(nil)
	_, err := tool.Run("")
	if err == nil || !strings.Contains(err.Error(), "vazia") {
		t.Fatalf("expected 'vazia' error, got %v", err)
	}
}

func TestSqlInspector_BlocksDestructive(t *testing.T) {
	tool := NewSqlInspectorTool(nil)
	blocked := []string{
		"UPDATE users SET name='x'",
		"DELETE FROM users",
		"DROP TABLE users",
		"INSERT INTO users VALUES(1)",
		"ALTER TABLE users ADD col INT",
		"TRUNCATE TABLE users",
		"GRANT ALL ON users TO admin",
		"REVOKE ALL ON users FROM admin",
		"REPLACE INTO users VALUES(1)",
	}
	for _, q := range blocked {
		_, err := tool.Run(q)
		if err == nil || !strings.Contains(err.Error(), "bloqueada") {
			t.Errorf("expected 'bloqueada' for %q, got %v", q, err)
		}
	}
}

func TestSqlInspector_BlocksCaseInsensitive(t *testing.T) {
	tool := NewSqlInspectorTool(nil)
	_, err := tool.Run("update USERS set name='x'")
	if err == nil || !strings.Contains(err.Error(), "bloqueada") {
		t.Fatalf("expected case-insensitive block, got %v", err)
	}
}

func TestSqlInspector_MissingConfig(t *testing.T) {
	// With nil configRepo, should panic or error on SELECT
	// We test only validation since configRepo is concrete struct
	tool := NewSqlInspectorTool(nil)
	if tool.Name() != "sql_inspector" {
		t.Fatalf("expected name 'sql_inspector', got %q", tool.Name())
	}
	if tool.Safety() != "dangerous" {
		t.Fatalf("expected safety 'dangerous', got %q", tool.Safety())
	}
}
