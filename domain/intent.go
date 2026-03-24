package domain

// ToolSafety — nivel de seguranca de uma tool
type ToolSafety string

const (
	ToolSafe       ToolSafety = "safe"       // Execucao direta
	ToolRestricted ToolSafety = "restricted" // Validar input
	ToolDangerous  ToolSafety = "dangerous"  // Exigir confirmacao
)

// SafeTool — tool com classificacao de seguranca
type SafeTool interface {
	Tool
	Safety() ToolSafety
}
