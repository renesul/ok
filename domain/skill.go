package domain

// Skill representa uma habilidade carregavel pelo agente a partir de arquivo .md
type Skill struct {
	Name        string
	Description string
	Content     string
}

// SkillRepository permite listar e buscar skills do filesystem
type SkillRepository interface {
	List() ([]Skill, error)
	Get(name string) (*Skill, error)
}
