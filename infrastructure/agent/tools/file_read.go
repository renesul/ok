package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/renesul/ok/domain"
)

const (
	maxReadLines = 500
	maxReadBytes = 50 * 1024 // 50KB
	mimeCheckSize = 512
)

var binaryExtensions = map[string]bool{
	".sqlite": true, ".db": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".zip": true, ".tar": true, ".gz": true, ".exe": true,
	".bin": true, ".pdf": true, ".wasm": true, ".ico": true, ".bmp": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".wav": true,
	".7z": true, ".rar": true, ".so": true, ".dylib": true, ".dll": true,
}

type FileReadTool struct {
	baseDir string
}

func NewFileReadTool(baseDir string) *FileReadTool {
	return &FileReadTool{baseDir: baseDir}
}

func (t *FileReadTool) Name() string { return "file_read" }
func (t *FileReadTool) Description() string {
	return "Reads lines from a file in the sandbox. Input JSON: {\"file\":\"name\", \"start_line\":1, \"end_line\":100}. For large files, paginate with start_line/end_line. Max 500 lines per call. Rejects binaries."
}
func (t *FileReadTool) Safety() domain.ToolSafety { return domain.ToolSafe }

func (t *FileReadTool) Run(input string) (string, error) {
	var req struct {
		File      string `json:"file"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		// Fallback: input simples como nome do arquivo
		req.File = strings.TrimSpace(input)
	}

	if req.File == "" {
		return "", fmt.Errorf("file required. Input JSON: {\"file\":\"name\", \"start_line\":1, \"end_line\":100}")
	}

	safe, err := safePath(t.baseDir, req.File)
	if err != nil {
		return "", err
	}

	// Filtro de extensao binaria
	ext := strings.ToLower(filepath.Ext(safe))
	if binaryExtensions[ext] {
		return "", fmt.Errorf("binary file denied: %s", ext)
	}

	// Filtro MIME (primeiros 512 bytes)
	if err := checkTextMIME(safe); err != nil {
		return "", err
	}

	// Defaults
	if req.StartLine < 1 {
		req.StartLine = 1
	}
	if req.EndLine < req.StartLine {
		req.EndLine = req.StartLine + maxReadLines - 1
	}
	if req.EndLine-req.StartLine+1 > maxReadLines {
		req.EndLine = req.StartLine + maxReadLines - 1
	}

	return readLines(safe, req.StartLine, req.EndLine)
}

func checkTextMIME(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, mimeCheckSize)
	n, _ := f.Read(buf)
	if n == 0 {
		return nil // arquivo vazio e ok
	}

	mime := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(mime, "text/") && mime != "application/json" && mime != "application/xml" && mime != "application/javascript" {
		return fmt.Errorf("binary file denied (MIME: %s)", mime)
	}
	return nil
}

func readLines(path string, startLine, endLine int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	var out strings.Builder
	lineNum := 0
	totalBytes := 0
	truncated := false
	totalLines := 0

	for scanner.Scan() {
		lineNum++
		totalLines = lineNum

		if lineNum < startLine {
			continue
		}
		if lineNum > endLine {
			break
		}

		line := scanner.Text()
		entry := fmt.Sprintf("%d\t%s\n", lineNum, line)

		if totalBytes+len(entry) > maxReadBytes {
			truncated = true
			break
		}

		out.WriteString(entry)
		totalBytes += len(entry)
	}

	// Contar linhas restantes
	for scanner.Scan() {
		totalLines++
	}

	result := out.String()
	if truncated || lineNum < totalLines {
		nextStart := lineNum + 1
		if truncated {
			nextStart = lineNum
		}
		result += fmt.Sprintf("\n[TRUNCATED: %d total lines. Use start_line=%d to continue]", totalLines, nextStart)
	}

	return result, nil
}

func safePath(baseDir, relativePath string) (string, error) {
	if strings.Contains(relativePath, "..") {
		return "", fmt.Errorf("path traversal blocked")
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("absolute path not allowed")
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base dir: %w", err)
	}

	full := filepath.Join(absBase, filepath.Clean(relativePath))

	// Resolve symlinks se o arquivo existir para barrar escapes malignos
	evaluated, err := filepath.EvalSymlinks(full)
	if err == nil {
		full = evaluated
	}

	if !strings.HasPrefix(full, absBase) {
		return "", fmt.Errorf("path outside sandbox (symlink escape detected)")
	}

	return full, nil
}
