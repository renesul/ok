package handler

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	"go.uber.org/zap"
)

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

type WSHandler struct {
	agentService   *application.AgentService
	confirmManager *agent.ConfirmationManager
	hub            *WSHub
	log            *zap.Logger
	cancelFunc     context.CancelFunc
}

func NewWSHandler(agentService *application.AgentService, confirmManager *agent.ConfirmationManager, log *zap.Logger) *WSHandler {
	return &WSHandler{
		agentService:   agentService,
		confirmManager: confirmManager,
		hub:            NewWSHub(),
		log:            log.Named("handler.ws"),
	}
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
			if h.cancelFunc != nil {
				h.cancelFunc()
			}
		}
	}
}

func (h *WSHandler) runAgent(input string) {
	ctx, cancel := context.WithCancel(context.Background())
	h.cancelFunc = cancel
	defer func() { h.cancelFunc = nil }()

	h.hub.mu.Lock()
	h.hub.isRunning = true
	h.hub.terminalHistory = nil
	h.hub.currentPhase = ""
	h.hub.mu.Unlock()

	h.agentService.RunStream(ctx, input, func(event domain.AgentEvent) {
		// Acumular estado para hydration
		switch event.Type {
		case "phase":
			h.hub.mu.Lock()
			h.hub.currentPhase = event.Content
			h.hub.mu.Unlock()
		case "stream":
			if event.Tool == "shell" || event.Tool == "repl" {
				h.hub.mu.Lock()
				h.hub.terminalHistory = append(h.hub.terminalHistory, event.Content)
				if len(h.hub.terminalHistory) > 100 {
					h.hub.terminalHistory = h.hub.terminalHistory[len(h.hub.terminalHistory)-100:]
				}
				h.hub.mu.Unlock()
			}
		case "done":
			h.hub.mu.Lock()
			h.hub.isRunning = false
			h.hub.mu.Unlock()
		}

		// Broadcast para todos os clientes WS
		data, err := json.Marshal(event)
		if err == nil {
			h.hub.Broadcast(data)
		}
	})
}
