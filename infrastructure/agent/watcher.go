package agent

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fsnotify/fsnotify"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

const (
	watchDebounce   = 2 * time.Second
	watchMaxFileSize = 50 * 1024 // 50KB
)

var watchSkipDirs = map[string]bool{
	".git": true, "node_modules": true, ".venv": true,
	"__pycache__": true, "vendor": true, ".idea": true,
	".vscode": true, "dist": true, "build": true, "data": true,
}

type FileWatcher struct {
	memory  *SQLiteMemory
	rootDir string
	log     *zap.Logger
	watcher *fsnotify.Watcher
	stopCh  chan struct{}

	mu       sync.Mutex
	lastSeen map[string]time.Time
}

func NewFileWatcher(memory *SQLiteMemory, rootDir string, log *zap.Logger) *FileWatcher {
	return &FileWatcher{
		memory:   memory,
		rootDir:  rootDir,
		log:      log.Named("watcher"),
		stopCh:   make(chan struct{}),
		lastSeen: make(map[string]time.Time),
	}
}

func (w *FileWatcher) Start() {
	if w.rootDir == "" {
		w.log.Debug("watcher disabled: no root dir configured")
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.log.Debug("watcher init failed", zap.Error(err))
		return
	}
	w.watcher = watcher

	w.addDirsRecursive(w.rootDir)
	w.log.Debug("watcher started", zap.String("root", w.rootDir))

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write != 0 {
				w.handleWrite(event.Name)
			}
			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.addDirsRecursive(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			w.log.Debug("watcher error", zap.Error(err))

		case <-w.stopCh:
			watcher.Close()
			return
		}
	}
}

func (w *FileWatcher) Stop() {
	close(w.stopCh)
}

func (w *FileWatcher) handleWrite(path string) {
	w.mu.Lock()
	last, exists := w.lastSeen[path]
	now := time.Now()
	if exists && now.Sub(last) < watchDebounce {
		w.mu.Unlock()
		return
	}
	w.lastSeen[path] = now
	w.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > watchMaxFileSize {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	if !utf8.Valid(data[:min(len(data), 512)]) {
		return
	}

	relPath, _ := filepath.Rel(w.rootDir, path)
	content := "file:" + relPath + "\n" + string(data)

	// Deletar entries antigas deste arquivo
	w.memory.DB().Exec("DELETE FROM agent_memory WHERE category = 'file_index' AND content LIKE ?", "file:"+relPath+"\n%")

	chunks := splitChunks(content, maxContentLength)
	for i, chunk := range chunks {
		id := "file_" + strings.ReplaceAll(relPath, "/", "_")
		if i > 0 {
			id += fmt.Sprintf("_chunk_%d", i)
		}
		w.memory.Save(domain.MemoryEntry{
			ID:       id,
			Content:  chunk,
			Category: "file_index",
		})
	}

	w.log.Debug("file re-indexed", zap.String("path", relPath), zap.Int("chunks", len(chunks)))
}

func (w *FileWatcher) addDirsRecursive(root string) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if watchSkipDirs[d.Name()] {
				return fs.SkipDir
			}
			w.watcher.Add(path)
		}
		return nil
	})
}

