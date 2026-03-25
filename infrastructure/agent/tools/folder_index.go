package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/renesul/ok/domain"
)

const (
	maxFileReadSize   = 50 * 1024  // 50KB por arquivo
	maxOutputSize     = 100 * 1024 // 100KB output total
	defaultMaxDepth   = 5
	binaryCheckSize   = 512
)

var blockedPaths = []string{"/", "/etc", "/proc", "/sys", "/dev", "/boot", "/var/run", "/tmp"}

type IndexFolderTool struct {
	baseDir string
}

func NewIndexFolderTool(baseDir string) *IndexFolderTool {
	return &IndexFolderTool{baseDir: baseDir}
}

func (t *IndexFolderTool) Name() string                       { return "folder_index" }
func (t *IndexFolderTool) Description() string                { return "recursively scans directory and returns structure + file contents" }
func (t *IndexFolderTool) Safety() domain.ToolSafety          { return domain.ToolRestricted }

func (t *IndexFolderTool) Run(input string) (string, error) {
	var req struct {
		Path     string `json:"path"`
		Pattern  string `json:"pattern"`
		MaxDepth int    `json:"max_depth"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON: {\"path\":\"/caminho\", \"pattern\":\"*.go\", \"max_depth\":3}")
	}

	if req.Path == "" {
		return "", fmt.Errorf("path obrigatorio")
	}
	if req.Pattern == "" {
		req.Pattern = "*"
	}
	if req.MaxDepth <= 0 {
		req.MaxDepth = defaultMaxDepth
	}

	dir := req.Path
	if !filepath.IsAbs(dir) && t.baseDir != "" {
		dir = filepath.Join(t.baseDir, dir)
	}
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolver path: %w", err)
	}

	if err := validatePath(absPath); err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("path nao encontrado: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path nao e um diretorio: %s", absPath)
	}

	return indexDirectory(absPath, req.Pattern, req.MaxDepth)
}

func validatePath(absPath string) error {
	clean := filepath.Clean(absPath)
	for _, blocked := range blockedPaths {
		if clean == blocked {
			return fmt.Errorf("path bloqueado: %s", clean)
		}
	}
	return nil
}

func indexDirectory(root, pattern string, maxDepth int) (string, error) {
	var out strings.Builder
	var fileCount int
	var totalSize int64

	type fileEntry struct {
		path    string
		relPath string
		size    int64
		modTime string
		content string
		binary  bool
	}

	var entries []fileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}

		relPath, _ := filepath.Rel(root, path)
		depth := 0
		if relPath != "." {
			depth = strings.Count(relPath, string(filepath.Separator)) + 1
			if !d.IsDir() {
				// Para arquivos, a profundidade e o numero de diretorios pai
				depth = strings.Count(relPath, string(filepath.Separator))
			}
		}

		if depth > maxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		name := d.Name()
		matched, matchErr := filepath.Match(pattern, name)
		if matchErr != nil || !matched {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		entry := fileEntry{
			path:    path,
			relPath: relPath,
			size:    info.Size(),
			modTime: info.ModTime().Format("2006-01-02 15:04"),
		}

		if info.Size() <= maxFileReadSize {
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				if isTextContent(data) {
					entry.content = string(data)
				} else {
					entry.binary = true
				}
			}
		} else {
			entry.binary = true
		}

		entries = append(entries, entry)
		fileCount++
		totalSize += info.Size()
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk directory: %w", err)
	}

	fmt.Fprintf(&out, "=== %s ===\n", root)
	fmt.Fprintf(&out, "Files: %d | Total: %s\n\n", fileCount, formatSize(totalSize))

	for _, entry := range entries {
		if out.Len() > maxOutputSize {
			fmt.Fprintf(&out, "\n... output truncado (%s limite)\n", formatSize(maxOutputSize))
			break
		}

		if entry.content != "" {
			fmt.Fprintf(&out, "--- %s (%s) ---\n", entry.relPath, formatSize(entry.size))
			out.WriteString(entry.content)
			if !strings.HasSuffix(entry.content, "\n") {
				out.WriteByte('\n')
			}
			out.WriteByte('\n')
		} else {
			reason := "binary"
			if entry.size > maxFileReadSize {
				reason = "too large"
			}
			fmt.Fprintf(&out, "--- %s (%s, %s) ---\n", entry.relPath, formatSize(entry.size), reason)
			fmt.Fprintf(&out, "[metadata only: %s, modified %s]\n\n", formatSize(entry.size), entry.modTime)
		}
	}

	return out.String(), nil
}

func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	check := data
	if len(check) > binaryCheckSize {
		check = check[:binaryCheckSize]
	}
	return utf8.Valid(check)
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.1fMB", mb)
}
