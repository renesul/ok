package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/identity"
	"github.com/renesul/ok/pkg/logger"
	"github.com/renesul/ok/pkg/providers"
)

// chatConn represents a single WebSocket connection to the chat UI.
type chatConn struct {
	id        string
	conn      *websocket.Conn
	sessionID string
	writeMu   sync.Mutex
	closed    atomic.Bool
}

func (c *chatConn) writeJSON(v any) error {
	if c.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}

func (c *chatConn) close() {
	if c.closed.CompareAndSwap(false, true) {
		c.conn.Close()
	}
}

// HistoryProvider returns past session messages for a given channel and chatID.
type HistoryProvider func(channel, chatID string) []providers.Message

// ChatChannel is the built-in web chat channel. Always active, no config needed.
type ChatChannel struct {
	*BaseChannel
	upgrader        websocket.Upgrader
	connections     sync.Map // connID → *chatConn
	ctx             context.Context
	cancel          context.CancelFunc
	historyProvider HistoryProvider
}

// SetHistoryProvider sets the callback used to load session history on connect.
func (c *ChatChannel) SetHistoryProvider(hp HistoryProvider) {
	c.historyProvider = hp
}

// NewChatChannel creates the built-in web chat channel.
func NewChatChannel(messageBus *bus.MessageBus) *ChatChannel {
	base := NewBaseChannel("chat", nil, messageBus, nil)
	return &ChatChannel{
		BaseChannel: base,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (c *ChatChannel) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	logger.InfoC("chat", "Web chat channel started")
	return nil
}

func (c *ChatChannel) Stop(ctx context.Context) error {
	c.SetRunning(false)
	c.connections.Range(func(key, value any) bool {
		if cc, ok := value.(*chatConn); ok {
			cc.close()
		}
		c.connections.Delete(key)
		return true
	})
	if c.cancel != nil {
		c.cancel()
	}
	logger.InfoC("chat", "Web chat channel stopped")
	return nil
}

// WebhookPath implements WebhookHandler — registers on the shared mux.
func (c *ChatChannel) WebhookPath() string { return "/chat/" }

// ServeHTTP handles WebSocket upgrade for the chat channel.
func (c *ChatChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/chat")
	switch {
	case path == "/ws" || path == "/ws/":
		c.handleWebSocket(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Send delivers an outbound message to the matching WebSocket session.
func (c *ChatChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return ErrNotRunning
	}
	out := chatMsg{
		Type:      "message.create",
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"content": msg.Content},
	}
	return c.broadcastToSession(msg.ChatID, out)
}

// EditMessage implements MessageEditor.
func (c *ChatChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	out := chatMsg{
		Type:      "message.update",
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"message_id": messageID, "content": content},
	}
	return c.broadcastToSession(chatID, out)
}

// StartTyping implements TypingCapable.
func (c *ChatChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	start := chatMsg{Type: "typing.start", Timestamp: time.Now().UnixMilli()}
	if err := c.broadcastToSession(chatID, start); err != nil {
		return func() {}, err
	}
	return func() {
		stop := chatMsg{Type: "typing.stop", Timestamp: time.Now().UnixMilli()}
		c.broadcastToSession(chatID, stop)
	}, nil
}

// SendPlaceholder implements PlaceholderCapable.
func (c *ChatChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	msgID := uuid.New().String()
	out := chatMsg{
		Type:      "message.create",
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"content": "Thinking...", "message_id": msgID},
	}
	if err := c.broadcastToSession(chatID, out); err != nil {
		return "", err
	}
	return msgID, nil
}

func (c *ChatChannel) broadcastToSession(chatID string, msg chatMsg) error {
	sessionID := strings.TrimPrefix(chatID, "chat:")
	msg.SessionID = sessionID

	var sent bool
	c.connections.Range(func(key, value any) bool {
		cc, ok := value.(*chatConn)
		if !ok {
			return true
		}
		if cc.sessionID == sessionID {
			if err := cc.writeJSON(msg); err == nil {
				sent = true
			}
		}
		return true
	})
	if !sent {
		return fmt.Errorf("no active connections for session %s: %w", sessionID, ErrSendFailed)
	}
	return nil
}

func (c *ChatChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !c.IsRunning() {
		http.Error(w, "channel not running", http.StatusServiceUnavailable)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("chat", "WebSocket upgrade failed", map[string]any{"error": err.Error()})
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = "default"
	}

	cc := &chatConn{
		id:        uuid.New().String(),
		conn:      conn,
		sessionID: sessionID,
	}
	c.connections.Store(cc.id, cc)

	logger.InfoCF("chat", "Client connected", map[string]any{"session_id": sessionID})

	// Send previous session history to the client.
	c.sendHistory(cc)

	go c.readLoop(cc)
}

func (c *ChatChannel) sendHistory(cc *chatConn) {
	if c.historyProvider == nil {
		return
	}
	chatID := "chat:" + cc.sessionID
	msgs := c.historyProvider("chat", chatID)
	if len(msgs) == 0 {
		return
	}
	for _, msg := range msgs {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		if msg.Content == "" {
			continue
		}
		role := "bot"
		if msg.Role == "user" {
			role = "user"
		}
		out := chatMsg{
			Type:      "message.create",
			SessionID: cc.sessionID,
			Timestamp: time.Now().UnixMilli(),
			Payload:   map[string]any{"content": msg.Content, "role": role},
		}
		if err := cc.writeJSON(out); err != nil {
			logger.WarnCF("chat", "Failed to send history message", map[string]any{"error": err.Error()})
			return
		}
	}
	logger.InfoCF("chat", "Session history sent", map[string]any{
		"session_id": cc.sessionID,
		"messages":   len(msgs),
	})
}

func (c *ChatChannel) readLoop(cc *chatConn) {
	defer func() {
		cc.close()
		c.connections.Delete(cc.id)
		logger.InfoCF("chat", "Client disconnected", map[string]any{"session_id": cc.sessionID})
	}()

	readTimeout := 60 * time.Second
	_ = cc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	cc.conn.SetPongHandler(func(string) error {
		_ = cc.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	go c.pingLoop(cc)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, raw, err := cc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.DebugCF("chat", "WebSocket read error", map[string]any{"error": err.Error()})
			}
			return
		}
		_ = cc.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg chatMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			cc.writeJSON(chatMsg{Type: "error", Payload: map[string]any{"message": "invalid message"}})
			continue
		}

		switch msg.Type {
		case "ping":
			cc.writeJSON(chatMsg{Type: "pong", ID: msg.ID})
		case "message.send":
			c.handleInbound(cc, msg)
		}
	}
}

func (c *ChatChannel) pingLoop(cc *chatConn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if cc.closed.Load() {
				return
			}
			cc.writeMu.Lock()
			err := cc.conn.WriteMessage(websocket.PingMessage, nil)
			cc.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (c *ChatChannel) handleInbound(cc *chatConn, msg chatMsg) {
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		cc.writeJSON(chatMsg{Type: "error", Payload: map[string]any{"message": "empty content"}})
		return
	}

	sessionID := cc.sessionID
	chatID := "chat:" + sessionID
	senderID := "chat-user"

	peer := bus.Peer{Kind: "direct", ID: chatID}
	sender := bus.SenderInfo{
		Platform:    "chat",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("chat", senderID),
	}

	c.HandleMessage(c.ctx, peer, msg.ID, senderID, chatID, content, nil, map[string]string{
		"platform":   "chat",
		"session_id": sessionID,
	}, sender)
}

// chatMsg is the wire format for chat WebSocket messages.
type chatMsg struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

