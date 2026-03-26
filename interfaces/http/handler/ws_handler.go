package handler

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	"go.uber.org/zap"
)

const maxWSHistory = 20

type WSHub struct {
	mu              sync.RWMutex
	conns           map[*websocket.Conn]bool
	terminalHistory []string
	isRunning       bool
	currentPhase    string
}

func NewWSHub() *WSHub {
	return &WSHub{conns: make(map[*websocket.Conn]bool)}
}

func (h *WSHub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.conns {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (h *WSHub) SetRunning(running bool) {
	h.mu.Lock()
	h.isRunning = running
	if running {
		h.terminalHistory = nil
		h.currentPhase = ""
	}
	h.mu.Unlock()
}

func (h *WSHub) SetPhase(phase string) {
	h.mu.Lock()
	h.currentPhase = phase
	h.mu.Unlock()
}

func (h *WSHub) AppendTerminal(content string) {
	h.mu.Lock()
	h.terminalHistory = append(h.terminalHistory, content)
	if len(h.terminalHistory) > 100 {
		h.terminalHistory = h.terminalHistory[len(h.terminalHistory)-100:]
	}
	h.mu.Unlock()
}

func (h *WSHub) HydrationState() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]interface{}{
		"type":             "hydration",
		"running":          h.isRunning,
		"phase":            h.currentPhase,
		"terminal_history": h.terminalHistory,
	}
}

type wsTurn struct {
	Role    string
	Content string
}

type WSHandler struct {
	agentService   *application.AgentService
	confirmManager *agent.ConfirmationManager
	hub            *WSHub
	log            *zap.Logger
	cancelMu       sync.Mutex
	cancelFunc     context.CancelFunc
	historyMu      sync.Mutex
	history        []wsTurn
}

func NewWSHandler(agentService *application.AgentService, confirmManager *agent.ConfirmationManager, log *zap.Logger) *WSHandler {
	return &WSHandler{
		agentService:   agentService,
		confirmManager: confirmManager,
		hub:            NewWSHub(),
		log:            log.Named("handler.ws"),
	}
}

// Hub returns the WebSocket hub for external broadcast
func (h *WSHandler) Hub() *WSHub {
	return h.hub
}

func (h *WSHandler) Handle(c *websocket.Conn) {
	h.hub.mu.Lock()
	h.hub.conns[c] = true
	h.hub.mu.Unlock()

	defer func() {
		h.hub.mu.Lock()
		delete(h.hub.conns, c)
		h.hub.mu.Unlock()
		c.Close()
	}()

	// Hydration on connect
	state := h.hub.HydrationState()
	if data, err := json.Marshal(state); err == nil {
		c.WriteMessage(websocket.TextMessage, data)
	}

	h.log.Debug("ws client connected")

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			h.log.Debug("ws client disconnected")
			break
		}

		var cmd struct {
			Type     string `json:"type"`
			Content  string `json:"content"`
			ID       string `json:"id"`
			Approved bool   `json:"approved"`
		}
		if err := json.Unmarshal(msg, &cmd); err != nil {
			continue
		}

		switch cmd.Type {
		case "input":
			if cmd.Content != "" {
				go h.runAgent(cmd.Content)
			}
		case "confirm":
			if h.confirmManager != nil && cmd.ID != "" {
				h.confirmManager.Respond(cmd.ID, cmd.Approved)
			}
		case "cancel":
			h.cancelMu.Lock()
			fn := h.cancelFunc
			h.cancelMu.Unlock()
			if fn != nil {
				fn()
			}
		}
	}
}

func (h *WSHandler) buildAgentInput(input string) string {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	if len(h.history) == 0 {
		return input
	}

	var parts []string
	for _, turn := range h.history {
		parts = append(parts, turn.Role+": "+turn.Content)
	}
	return "Conversation history:\n" + strings.Join(parts, "\n") + "\n\nCurrent message: " + input
}

func (h *WSHandler) appendHistory(role, content string) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	h.history = append(h.history, wsTurn{Role: role, Content: content})
	if len(h.history) > maxWSHistory*2 {
		h.history = h.history[len(h.history)-maxWSHistory*2:]
	}
}

func (h *WSHandler) runAgent(input string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	h.cancelMu.Lock()
	h.cancelFunc = cancel
	h.cancelMu.Unlock()
	defer func() {
		h.cancelMu.Lock()
		h.cancelFunc = nil
		h.cancelMu.Unlock()
	}()

	agentInput := h.buildAgentInput(input)
	h.appendHistory("user", input)

	h.hub.SetRunning(true)

	var response strings.Builder
	h.agentService.RunStream(ctx, agentInput, func(e domain.AgentEvent) {
		if e.Type == "message" && e.Content != "" {
			response.WriteString(e.Content)
		}
	})

	if response.Len() > 0 {
		h.appendHistory("assistant", response.String())
	}
}
