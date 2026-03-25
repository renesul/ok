package adapters

import (
	"context"

	"github.com/renesul/ok/domain"
)

type mockAgentRunner struct {
	called    bool
	lastInput string
	callCount int
	response  domain.AgentResponse
	err       error
}

func newMockRunner() *mockAgentRunner {
	return &mockAgentRunner{
		response: domain.AgentResponse{
			Messages: []string{"ok"},
			Done:     true,
		},
	}
}

func (m *mockAgentRunner) Run(_ context.Context, input string) (domain.AgentResponse, error) {
	m.called = true
	m.lastInput = input
	m.callCount++
	return m.response, m.err
}

func strPtr(s string) *string {
	return &s
}
