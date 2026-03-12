package context

import (
	"fmt"
	"strings"
)

// Persona holds the loaded identity content from workspace markdown files.
// It provides the static "who am I" context for the agent's system prompt.
type Persona struct {
	Name        string // agent name (from config or default)
	Identity    string // IDENTITY.md content
	Soul        string // SOUL.md content
	UserProfile string // USER.md content
	Guidelines  string // AGENTS.md content
}

// BuildPromptSection returns the concatenated bootstrap content for the system prompt.
// Files that are empty or missing are silently skipped.
func (p *Persona) BuildPromptSection() string {
	files := []struct {
		label   string
		content string
	}{
		{"AGENTS.md", p.Guidelines},
		{"SOUL.md", p.Soul},
		{"USER.md", p.UserProfile},
		{"IDENTITY.md", p.Identity},
	}

	var sb strings.Builder
	for _, f := range files {
		if f.content != "" {
			fmt.Fprintf(&sb, "## %s\n\n%s\n\n", f.label, f.content)
		}
	}
	return sb.String()
}
