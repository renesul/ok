package tools

import "github.com/renesul/ok/domain"

type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string  { return "retorna o input recebido" }
func (t *EchoTool) Run(input string) (string, error) { return input, nil }
func (t *EchoTool) Safety() domain.ToolSafety          { return domain.ToolSafe }
