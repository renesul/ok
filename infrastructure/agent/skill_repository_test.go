package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupSkillsDir(t *testing.T) string {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)

	os.WriteFile(filepath.Join(skillsDir, "copywriter.md"), []byte(`---
name: copywriter
description: Sabe vender produtos com persuasao
---

# Regras
1. Use gatilhos mentais
2. Seja direto`), 0644)

	os.WriteFile(filepath.Join(skillsDir, "analyst.md"), []byte(`---
name: analyst
description: Analisa dados e gera insights
---

# Processo
1. Colete dados
2. Identifique padroes`), 0644)

	return dir
}

func TestFileSkillRepository_List(t *testing.T) {
	dir := setupSkillsDir(t)
	repo := NewFileSkillRepository(dir)
	skills, err := repo.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
}

func TestFileSkillRepository_Get(t *testing.T) {
	dir := setupSkillsDir(t)
	repo := NewFileSkillRepository(dir)
	skill, err := repo.Get("copywriter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "copywriter" {
		t.Fatalf("expected name 'copywriter', got %q", skill.Name)
	}
	if skill.Description != "Sabe vender produtos com persuasao" {
		t.Fatalf("expected description, got %q", skill.Description)
	}
	if !strings.Contains(skill.Content, "gatilhos mentais") {
		t.Fatalf("expected content to contain 'gatilhos mentais', got %q", skill.Content)
	}
}

func TestFileSkillRepository_GetNotFound(t *testing.T) {
	dir := setupSkillsDir(t)
	repo := NewFileSkillRepository(dir)
	_, err := repo.Get("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestFileSkillRepository_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileSkillRepository(dir)
	skills, err := repo.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(skills))
	}
}

func TestFileSkillRepository_MalformedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)

	// No frontmatter
	os.WriteFile(filepath.Join(skillsDir, "broken.md"), []byte("just text without frontmatter"), 0644)

	// Valid one
	os.WriteFile(filepath.Join(skillsDir, "valid.md"), []byte("---\nname: valid\ndescription: works\n---\ncontent"), 0644)

	repo := NewFileSkillRepository(dir)
	skills, err := repo.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Malformed should be skipped, only valid returned
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill (malformed skipped), got %d", len(skills))
	}
	if skills[0].Name != "valid" {
		t.Fatalf("expected 'valid', got %q", skills[0].Name)
	}
}
