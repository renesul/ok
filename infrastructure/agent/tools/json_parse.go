package tools

import (
	"encoding/json"
	"fmt"

	"github.com/renesul/ok/domain"
	"strings"
)

type JSONParseTool struct{}

func (t *JSONParseTool) Name() string        { return "json_parse" }
func (t *JSONParseTool) Description() string { return "parseia JSON e extrai campos por path (ex: field.sub)" }
func (t *JSONParseTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

type jsonParseInput struct {
	JSON string `json:"json"`
	Path string `json:"path"`
}

func (t *JSONParseTool) Run(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("input vazio")
	}

	var req jsonParseInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		// Trata input direto como JSON para formatar
		var raw interface{}
		if err2 := json.Unmarshal([]byte(input), &raw); err2 != nil {
			return "", fmt.Errorf("JSON invalido: %w", err2)
		}
		formatted, _ := json.MarshalIndent(raw, "", "  ")
		return string(formatted), nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(req.JSON), &data); err != nil {
		return "", fmt.Errorf("JSON invalido: %w", err)
	}

	if req.Path == "" {
		formatted, _ := json.MarshalIndent(data, "", "  ")
		return string(formatted), nil
	}

	result := extractPath(data, strings.Split(req.Path, "."))
	if result == nil {
		return "", fmt.Errorf("path '%s' nao encontrado", req.Path)
	}

	switch v := result.(type) {
	case string:
		return v, nil
	default:
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b), nil
	}
}

func extractPath(data interface{}, parts []string) interface{} {
	if len(parts) == 0 {
		return data
	}

	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	val, exists := m[parts[0]]
	if !exists {
		return nil
	}

	return extractPath(val, parts[1:])
}
