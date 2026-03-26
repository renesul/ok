package domain

import (
	"context"
	"strings"
	"time"
)

// Decision — vem do LLM (nao confiavel)
type Decision struct {
	Tool  string `json:"tool"`
	Input string `json:"input"`
	Done  bool   `json:"done"`
}

// Plan — validado pelo Planner (confiavel)
type Plan struct {
	Tool  Tool
	Input string
}

// Tool — contrato para qualquer ferramenta
type Tool interface {
	Name() string
	Description() string
	Run(input string) (string, error)
}

// ContextualTool — tool que aceita context para cancelamento e timeout
type ContextualTool interface {
	Tool
	RunWithContext(ctx context.Context, input string) (string, error)
}

// StreamCallback recebe chunks de output em tempo real
type StreamCallback func(chunk string)

// StreamingTool — tool que pode emitir output incrementalmente via PTY
type StreamingTool interface {
	ContextualTool
	SetStreamCallback(cb StreamCallback)
}

// ToolSchema describes a tool for function calling APIs
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Planner — valida decisoes do LLM antes da execucao
type Planner interface {
	Plan(decision Decision, ctx *AgentContext) (Plan, error)
	RegisterTool(tool Tool)
	ToolDescriptions() string
	ToolSchemas() []ToolSchema
	Tools() map[string]Tool
}

// Executor — executa plans validados
type Executor interface {
	Execute(plan Plan) (string, error)
}

// MemoryEntry — unidade de memoria persistente
type MemoryEntry struct {
	ID        string
	Content   string
	Category  string // "preference", "context", "fact"
	CreatedAt time.Time
}

// StepResult — resultado de um step para o frontend
type StepResult struct {
	Name   string `json:"name"`
	Tool   string `json:"tool"`
	Status string `json:"status"`
}

// AgentResponse — resposta estruturada do agent
type AgentResponse struct {
	Messages []string     `json:"messages"`
	Steps    []StepResult `json:"steps"`
	Memory   []string     `json:"memory,omitempty"`
	Done     bool         `json:"done"`
}

// AgentEvent — evento de streaming para o frontend
type AgentEvent struct {
	Type      string `json:"type"`                 // "step", "message", "done", "intent", "token", "confirm", "phase"
	Name      string `json:"name,omitempty"`
	Tool      string `json:"tool,omitempty"`
	Status    string `json:"status,omitempty"`
	Content   string `json:"content,omitempty"`
	ElapsedMs int64  `json:"elapsed_ms,omitempty"`
	Mode      string `json:"mode,omitempty"`       // "direct", "task", "agent"
	Intent    string `json:"intent,omitempty"`      // "chat", "task", "action"
	Safety    string `json:"safety,omitempty"`      // "safe", "restricted", "dangerous"
}

// EventCallback — funcao chamada para cada evento durante streaming
type EventCallback func(event AgentEvent)

// AgentContext — historico + controle de steps + memorias
type AgentContext struct {
	History  []string
	Steps    int
	MaxSteps int
}

func NewAgentContext(maxSteps int) *AgentContext {
	return &AgentContext{MaxSteps: maxSteps}
}

func (c *AgentContext) Add(entry string) {
	c.History = append(c.History, entry)
}

func (c *AgentContext) Increment() {
	c.Steps++
}

func (c *AgentContext) LimitReached() bool {
	return c.Steps >= c.MaxSteps
}

func (c *AgentContext) String() string {
	return strings.Join(c.History, "\n")
}

// --- Autonomous Agent Loop Types ---

// ExecutionPhase — fase atual do loop autonomo
type ExecutionPhase string

const (
	PhaseObserve ExecutionPhase = "observe"
	PhasePlan    ExecutionPhase = "plan"
	PhaseAct     ExecutionPhase = "act"
	PhaseReflect ExecutionPhase = "reflect"
	PhaseDone    ExecutionPhase = "done"
	PhaseError   ExecutionPhase = "error"
)

// ExecutionBudget — limites de execucao
type ExecutionBudget struct {
	MaxSteps    int
	MaxDuration time.Duration
	StartTime   time.Time
}

// ExecutionState — estado completo de uma execucao autonoma
type ExecutionState struct {
	Goal        string
	Plan        *ExecutionPlan
	CurrentStep int
	Phase       ExecutionPhase
	History     []ExecutionEntry
	Memories    []string
	StepResults []StepResult
	Attempts    int
	Budget      ExecutionBudget
}

// ExecutionPlan — plano estruturado gerado pelo LLM
type ExecutionPlan struct {
	Steps     []PlannedStep `json:"steps"`
	Reasoning string        `json:"reasoning"`
}

// PlannedStep — um passo planejado
type PlannedStep struct {
	Name    string `json:"name"`
	Tool    string `json:"tool"`
	Input   string `json:"input"`
	Purpose string `json:"purpose"`
	Output  string `json:"-"`
	Status  string `json:"-"`
}

// ExecutionEntry — entrada no scratchpad de execucao
type ExecutionEntry struct {
	Phase      ExecutionPhase `json:"phase"`
	Content    string         `json:"content"`
	Tool       string         `json:"tool,omitempty"`
	DurationMs int64          `json:"duration_ms,omitempty"`
}

// ExecutionRecord — registro persistido de uma execucao completa
type ExecutionRecord struct {
	ID          string           `json:"id"`
	Goal        string           `json:"goal"`
	Status      string           `json:"status"` // done, error
	Steps       []StepResult     `json:"steps"`
	Timeline    []ExecutionEntry `json:"timeline"`
	TotalMs     int64            `json:"total_ms"`
	StepCount     int              `json:"step_count"`
	ToolsUsed     []string         `json:"tools_used"`
	FailureReason string           `json:"failure_reason,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
}

// ExecutionMetrics — metricas agregadas de execucoes
type ExecutionMetrics struct {
	TotalExecutions int            `json:"total_executions"`
	SuccessRate     float64        `json:"success_rate"`
	AvgDurationMs   int64          `json:"avg_duration_ms"`
	AvgStepCount    float64        `json:"avg_step_count"`
	ToolUsageCount  map[string]int `json:"tool_usage_count"`
}

// --- Agent Limits ---

// AgentLimits — limites de execucao do agente
type AgentLimits struct {
	MaxSteps    int `json:"max_steps"`
	MaxAttempts int `json:"max_attempts"`
	TimeoutMs   int `json:"timeout_ms"`
}

// DefaultAgentLimits retorna limites padrao para o agente local
func DefaultAgentLimits() AgentLimits {
	return AgentLimits{MaxSteps: 6, MaxAttempts: 4, TimeoutMs: 120000}
}

// ReflectionResult — avaliacao do LLM apos executar um step
type ReflectionResult struct {
	Action      string        `json:"action"`
	Reason      string        `json:"reason"`
	RevisedPlan []PlannedStep `json:"revised_plan,omitempty"`
	FinalAnswer string        `json:"final_answer,omitempty"`
}
