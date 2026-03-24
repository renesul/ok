package engine

import "github.com/renesul/ok/domain"

// CallbackEmitter delega eventos para uma funcao callback (caminho streaming/SSE)
type CallbackEmitter struct {
	emit func(domain.AgentEvent)
}

func NewCallbackEmitter(emit func(domain.AgentEvent)) *CallbackEmitter {
	return &CallbackEmitter{emit: emit}
}

func (e *CallbackEmitter) EmitPhase(phase string) {
	e.emit(domain.AgentEvent{Type: "phase", Content: phase})
}

func (e *CallbackEmitter) EmitStep(name, tool, status string, elapsedMs int64) {
	e.emit(domain.AgentEvent{Type: "step", Name: name, Tool: tool, Status: status, ElapsedMs: elapsedMs})
}

func (e *CallbackEmitter) EmitMessage(content string) {
	e.emit(domain.AgentEvent{Type: "message", Content: content})
}

func (e *CallbackEmitter) EmitDone() {
	e.emit(domain.AgentEvent{Type: "done"})
}

func (e *CallbackEmitter) EmitMemories(memories []string) {}

func (e *CallbackEmitter) EmitTerminal(tool, output string) {
	e.emit(domain.AgentEvent{Type: "terminal", Tool: tool, Content: output})
}

func (e *CallbackEmitter) EmitDiff(file, before, after string) {
	e.emit(domain.AgentEvent{Type: "diff", Name: file, Content: before + "\n---SEPARATOR---\n" + after})
}

func (e *CallbackEmitter) EmitConfirm(id, tool, summary string) {
	e.emit(domain.AgentEvent{Type: "confirm", Name: id, Tool: tool, Content: summary})
}

func (e *CallbackEmitter) EmitStream(source, chunk string) {
	e.emit(domain.AgentEvent{Type: "stream", Tool: source, Content: chunk})
}
