package tools

import (
	"encoding/base64"
	"encoding/json"

	"github.com/renesul/ok/domain"
	"fmt"
)

type Base64Tool struct{}

func (t *Base64Tool) Name() string        { return "base64" }
func (t *Base64Tool) Description() string { return "encodes/decodes base64 (action: encode|decode)" }
func (t *Base64Tool) Safety() domain.ToolSafety          { return domain.ToolSafe }

type base64Input struct {
	Action string `json:"action"` // encode, decode
	Data   string `json:"data"`
}

func (t *Base64Tool) Run(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty input")
	}

	var req base64Input
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		// Default: encode
		return base64.StdEncoding.EncodeToString([]byte(input)), nil
	}

	switch req.Action {
	case "encode":
		return base64.StdEncoding.EncodeToString([]byte(req.Data)), nil
	case "decode":
		decoded, err := base64.StdEncoding.DecodeString(req.Data)
		if err != nil {
			return "", fmt.Errorf("decode base64: %w", err)
		}
		return string(decoded), nil
	default:
		return "", fmt.Errorf("action must be 'encode' or 'decode'")
	}
}
