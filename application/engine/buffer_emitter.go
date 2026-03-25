package engine

import "github.com/renesul/ok/domain"

// BufferEmitter acumula eventos em um AgentResponse (caminho sincrono)
type BufferEmitter struct {
	response *domain.AgentResponse
}

func NewBufferEmitter() *BufferEmitter {
	return &BufferEmitter{
		response: &domain.AgentResponse{Done: true},
	}
}

func (e *BufferEmitter) EmitPhase(phase string) {}

func (e *BufferEmitter) EmitStep(name, tool, status string, elapsedMs int64) {
	e.response.Steps = append(e.response.Steps, domain.StepResult{
		Name:   name,
		Tool:   tool,
		Status: status,
	})
}

func (e *BufferEmitter) EmitMessage(content string) {
	e.response.Messages = append(e.response.Messages, content)
}

func (e *BufferEmitter) EmitDone() {}

func (e *BufferEmitter) EmitMemories(memories []string) {
	e.response.Memory = memories
}

func (e *BufferEmitter) EmitTerminal(tool, output string) {}
func (e *BufferEmitter) EmitDiff(file, before, after string) {}
func (e *BufferEmitter) EmitConfirm(id, tool, summary string) {}
func (e *BufferEmitter) EmitStream(source, chunk string)      {}

// Forward routes an AgentEvent to the appropriate buffer method
func (e *BufferEmitter) Forward(event domain.AgentEvent) {
	switch event.Type {
	case "step":
		e.EmitStep(event.Name, event.Tool, event.Status, event.ElapsedMs)
	case "message":
		e.EmitMessage(event.Content)
	case "memory":
		// memories come as single event, not array
	}
}

func (e *BufferEmitter) Response() domain.AgentResponse {
	return *e.response
}
