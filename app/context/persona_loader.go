package context

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersonaLoader loads persona files (IDENTITY.md, SOUL.md, USER.md, AGENTS.md)
// from the workspace directory with mtime-based caching.
type PersonaLoader struct {
	workspace string

	mu     sync.RWMutex
	cached *Persona
	mtimes map[string]time.Time // path → mtime at load time
}

// NewPersonaLoader creates a loader for the given workspace directory.
func NewPersonaLoader(workspace string) *PersonaLoader {
	return &PersonaLoader{
		workspace: workspace,
		mtimes:    make(map[string]time.Time),
	}
}

// personaFiles defines the mapping from Persona fields to workspace files.
var personaFiles = []struct {
	filename string
	setter   func(*Persona, string)
}{
	{"IDENTITY.md", func(p *Persona, s string) { p.Identity = s }},
	{"SOUL.md", func(p *Persona, s string) { p.Soul = s }},
	{"USER.md", func(p *Persona, s string) { p.UserProfile = s }},
	{"AGENTS.md", func(p *Persona, s string) { p.Guidelines = s }},
}

// Load returns the current persona, reloading from disk only if files changed.
func (l *PersonaLoader) Load() *Persona {
	l.mu.RLock()
	if l.cached != nil && !l.hasChangedLocked() {
		p := l.cached
		l.mu.RUnlock()
		return p
	}
	l.mu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock.
	if l.cached != nil && !l.hasChangedLocked() {
		return l.cached
	}

	p := &Persona{}
	newMtimes := make(map[string]time.Time, len(personaFiles))

	for _, pf := range personaFiles {
		path := filepath.Join(l.workspace, pf.filename)
		if data, err := os.ReadFile(path); err == nil {
			pf.setter(p, string(data))
			if info, statErr := os.Stat(path); statErr == nil {
				newMtimes[path] = info.ModTime()
			}
		}
	}

	l.cached = p
	l.mtimes = newMtimes
	return p
}

// HasChanged reports whether any persona file has been modified since last load.
func (l *PersonaLoader) HasChanged() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.hasChangedLocked()
}

// hasChangedLocked checks file changes. Caller must hold at least RLock.
func (l *PersonaLoader) hasChangedLocked() bool {
	for _, pf := range personaFiles {
		path := filepath.Join(l.workspace, pf.filename)
		info, err := os.Stat(path)

		cachedMtime, wasCached := l.mtimes[path]
		existsNow := err == nil

		if wasCached != existsNow {
			return true // file created or deleted
		}
		if existsNow && !info.ModTime().Equal(cachedMtime) {
			return true
		}
	}
	return false
}
