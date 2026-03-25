package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/renesul/ok/domain"
)

const (
	maxSearchResults = 50
	maxSearchOutput  = 10000
)

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, ".venv": true,
	"__pycache__": true, "vendor": true, ".idea": true,
	".vscode": true, "dist": true, "build": true,
}

var regexMeta = regexp.MustCompile(`[\\.*+?^${}()|[\]]`)

type SearchTool struct {
	baseDir string
}

func NewSearchTool(baseDir string) *SearchTool { return &SearchTool{baseDir: baseDir} }

func (t *SearchTool) Name() string                       { return "search" }
func (t *SearchTool) Description() string                { return "searches content in files recursively (ripgrep-like)" }
func (t *SearchTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *SearchTool) Run(input string) (string, error) {
	var req struct {
		Directory     string `json:"directory"`
		Pattern       string `json:"pattern"`
		FileExtension string `json:"file_extension"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON: {\"directory\":\"/path\", \"pattern\":\"texto\", \"file_extension\":\".go\"}")
	}

	if req.Directory == "" {
		return "", fmt.Errorf("directory obrigatorio")
	}
	if req.Pattern == "" {
		return "", fmt.Errorf("pattern obrigatorio")
	}

	dir := req.Directory
	if !filepath.IsAbs(dir) && t.baseDir != "" {
		dir = filepath.Join(t.baseDir, dir)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolver path: %w", err)
	}
	if err := validateSearchPath(absDir); err != nil {
		return "", err
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("diretorio nao encontrado: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path nao e diretorio: %s", absDir)
	}

	isRegex := regexMeta.MatchString(req.Pattern)
	var re *regexp.Regexp
	if isRegex {
		re, err = regexp.Compile(req.Pattern)
		if err != nil {
			return "", fmt.Errorf("regex invalido: %w", err)
		}
	}

	var out strings.Builder
	count := 0

	filepath.WalkDir(absDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || count >= maxSearchResults {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if req.FileExtension != "" {
			ext := req.FileExtension
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			if filepath.Ext(d.Name()) != ext {
				return nil
			}
		}

		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() && count < maxSearchResults {
			lineNum++
			line := scanner.Text()

			matched := false
			if isRegex {
				matched = re.MatchString(line)
			} else {
				matched = strings.Contains(line, req.Pattern)
			}

			if matched {
				relPath, _ := filepath.Rel(absDir, path)
				entry := fmt.Sprintf("%s:%d: %s\n", relPath, lineNum, line)
				if out.Len()+len(entry) > maxSearchOutput {
					out.WriteString("...(output truncado)\n")
					count = maxSearchResults
					return nil
				}
				out.WriteString(entry)
				count++
			}
		}
		return nil
	})

	if count == 0 {
		return "nenhum resultado encontrado", nil
	}

	return fmt.Sprintf("%d resultados:\n%s", count, out.String()), nil
}

func validateSearchPath(absPath string) error {
	blocked := []string{"/", "/etc", "/proc", "/sys", "/dev", "/boot", "/var/run"}
	clean := filepath.Clean(absPath)
	for _, b := range blocked {
		if clean == b {
			return fmt.Errorf("path bloqueado: %s", clean)
		}
	}
	return nil
}
