package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const confirmationTTL = 30 * time.Second

// PendingConfirmation representa uma execucao aguardando aprovacao
type PendingConfirmation struct {
	ID        string
	Tool      string
	Input     string
	CreatedAt time.Time
	Done      chan bool
}

// ConfirmationManager gerencia confirmacoes pendentes de tools perigosas
type ConfirmationManager struct {
	mu       sync.Mutex
	pending  map[string]*PendingConfirmation
}

// NewConfirmationManager cria um gerenciador de confirmacoes
func NewConfirmationManager() *ConfirmationManager {
	return &ConfirmationManager{
		pending: make(map[string]*PendingConfirmation),
	}
}

// Request cria uma confirmacao pendente e retorna o ID
func (m *ConfirmationManager) Request(tool, input string) *PendingConfirmation {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Limpar expiradas
	m.cleanExpired()

	conf := &PendingConfirmation{
		ID:        uuid.New().String(),
		Tool:      tool,
		Input:     input,
		CreatedAt: time.Now(),
		Done:      make(chan bool, 1),
	}
	m.pending[conf.ID] = conf
	return conf
}

// Respond processa a resposta do usuario
func (m *ConfirmationManager) Respond(id string, approved bool) error {
	m.mu.Lock()
	conf, exists := m.pending[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("confirmacao nao encontrada ou expirada: %s", id)
	}
	delete(m.pending, id)
	m.mu.Unlock()

	if time.Since(conf.CreatedAt) > confirmationTTL {
		return fmt.Errorf("confirmacao expirada")
	}

	conf.Done <- approved
	return nil
}

// WaitForResponse espera a resposta do usuario com timeout
func (m *ConfirmationManager) WaitForResponse(conf *PendingConfirmation) (bool, error) {
	timer := time.NewTimer(confirmationTTL)
	defer timer.Stop()

	select {
	case approved := <-conf.Done:
		return approved, nil
	case <-timer.C:
		m.mu.Lock()
		delete(m.pending, conf.ID)
		m.mu.Unlock()
		return false, fmt.Errorf("confirmacao expirou apos %s", confirmationTTL)
	}
}

func (m *ConfirmationManager) cleanExpired() {
	now := time.Now()
	for id, conf := range m.pending {
		if now.Sub(conf.CreatedAt) > confirmationTTL {
			delete(m.pending, id)
		}
	}
}
