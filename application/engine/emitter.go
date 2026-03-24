package engine

// Emitter abstrai como eventos do loop sao entregues (buffer sincrono vs SSE stream)
type Emitter interface {
	EmitPhase(phase string)
	EmitStep(name, tool, status string, elapsedMs int64)
	EmitMessage(content string)
	EmitDone()
	EmitMemories(memories []string)
	EmitTerminal(tool, output string)
	EmitDiff(file, before, after string)
	EmitConfirm(id, tool, summary string)
	EmitStream(source, chunk string)
}
