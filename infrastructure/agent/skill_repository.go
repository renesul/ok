package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/renesul/ok/domain"
)

// FileSkillRepository le skills de arquivos .md com frontmatter em disco
type FileSkillRepository struct {
	skillsDir string
}

func NewFileSkillRepository(sandboxDir string) *FileSkillRepository {
	dir := filepath.Join(sandboxDir, "skills")
	os.MkdirAll(dir, 0755)
	return &FileSkillRepository{skillsDir: dir}
}

func (r *FileSkillRepository) List() ([]domain.Skill, error) {
	entries, err := os.ReadDir(r.skillsDir)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	var skills []domain.Skill
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(r.skillsDir, entry.Name())
		skill, parseErr := parseSkillFile(path)
		if parseErr != nil {
			continue
		}
		skills = append(skills, *skill)
	}
	return skills, nil
}

func (r *FileSkillRepository) Get(name string) (*domain.Skill, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, fmt.Errorf("empty skill name")
	}

	path := filepath.Join(r.skillsDir, name+".md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}

	return parseSkillFile(path)
}

func parseSkillFile(path string) (*domain.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill: %w", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("missing frontmatter in %s", path)
	}

	// Split: ["", frontmatter, content...]
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("incomplete frontmatter in %s", path)
	}

	frontmatter := parts[1]
	body := strings.TrimSpace(parts[2])

	skill := &domain.Skill{Content: body}

	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if key, val, ok := strings.Cut(line, ":"); ok {
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			switch key {
			case "name":
				skill.Name = val
			case "description":
				skill.Description = val
			}
		}
	}

	if skill.Name == "" {
		// Fallback: usar nome do arquivo
		base := filepath.Base(path)
		skill.Name = strings.TrimSuffix(base, ".md")
	}

	return skill, nil
}
