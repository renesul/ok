package adapters

import (
	"context"

	"github.com/renesul/ok/domain"
)

type mockAgentRunner struct {
	called     bool
	lastInput  string
	callCount  int
	response   domain.AgentResponse
	err        error
	panicOnRun bool
	panicValue interface{}
}

func newMockRunner() *mockAgentRunner {
	return &mockAgentRunner{
		response: domain.AgentResponse{
			Messages: []string{"ok"},
			Done:     true,
		},
	}
}

func newPanickingRunner(val interface{}) *mockAgentRunner {
	return &mockAgentRunner{
		panicOnRun: true,
		panicValue: val,
	}
}

func (m *mockAgentRunner) Run(_ context.Context, input string) (domain.AgentResponse, error) {
	m.called = true
	m.lastInput = input
	m.callCount++
	if m.panicOnRun {
		panic(m.panicValue)
	}
	return m.response, m.err
}

func strPtr(s string) *string {
	return &s
}
