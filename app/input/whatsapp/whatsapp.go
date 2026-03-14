// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package whatsapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	msgbus "ok/app/input/bus"
	channels "ok/app/input"
	"ok/internal/config"
	"ok/internal/identity"
	"ok/internal/logger"
	"ok/internal/media"
	"ok/internal/utils"
)

const (
	sqliteDriver   = "sqlite"
	whatsappDBName = "store.db"

	reconnectInitial    = 5 * time.Second
	reconnectMax        = 5 * time.Minute
	reconnectMultiplier = 2.0
)

// QREvent is sent to SSE subscribers when the QR state changes.
type QREvent struct {
	Event  string `json:"event"`            // "code", "timeout", "success", "paired"
	Code   string `json:"code,omitempty"`   // QR code string (only for "code" events)
	Paired bool   `json:"paired,omitempty"` // true when already paired
}

// WhatsAppChannel implements the WhatsApp channel using whatsmeow (in-process, no external bridge).
type WhatsAppChannel struct {
	*channels.BaseChannel
	config       config.WhatsAppConfig
	storePath    string
	client       *whatsmeow.Client
	container    *sqlstore.Container
	mu           sync.Mutex
	runCtx       context.Context
	runCancel    context.CancelFunc
	reconnectMu  sync.Mutex
	reconnecting bool
	stopping     atomic.Bool    // set once Stop begins; prevents new wg.Add calls
	wg           sync.WaitGroup // tracks background goroutines (QR handler, reconnect)

	// QR state for web UI SSE streaming
	qrMu          sync.RWMutex
	qrCurrent     *QREvent                 // latest QR state (nil = no QR needed)
	qrSubscribers map[chan QREvent]struct{} // SSE subscribers

	// Track messages sent by the bot to prevent self-reply loops
	sentMu  sync.Mutex
	sentIDs map[string]struct{}
}

// NewWhatsAppChannel creates a WhatsApp channel that uses whatsmeow for connection.
// storePath is the directory for the SQLite session store (e.g. workspace/whatsapp).
func NewWhatsAppChannel(
	cfg config.WhatsAppConfig,
	bus *msgbus.MessageBus,
	storePath string,
) (channels.Channel, error) {
	base := channels.NewBaseChannel("whatsapp", cfg, bus, cfg.AllowFrom, channels.WithMaxMessageLength(65536))
	if storePath == "" {
		storePath = "whatsapp"
	}
	c := &WhatsAppChannel{
		BaseChannel:   base,
		config:        cfg,
		storePath:     storePath,
		qrSubscribers: make(map[chan QREvent]struct{}),
	}
	return c, nil
}

// WebhookPath implements channels.WebhookHandler — registers on the shared HTTP mux.
func (c *WhatsAppChannel) WebhookPath() string { return "/whatsapp/" }

// ServeHTTP handles QR-related HTTP endpoints.
// Access is restricted to loopback origins only.
func (c *WhatsAppChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow requests from loopback addresses.
	origin := r.Header.Get("Origin")
	if origin != "" && !isLoopbackOrigin(origin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	path := strings.TrimPrefix(r.URL.Path, "/whatsapp")
	switch {
	case path == "/qr/stream" || path == "/qr/stream/":
		c.handleQRStream(w, r)
	case path == "/qr/current" || path == "/qr/current/":
		c.handleQRCurrent(w, r)
	default:
		http.NotFound(w, r)
	}
}

// isLoopbackOrigin checks if an Origin header points to a loopback address.
func isLoopbackOrigin(origin string) bool {
	// Typical origins: http://127.0.0.1:18800, http://localhost:18800
	origin = strings.ToLower(origin)
	for _, prefix := range []string{
		"http://127.0.0.1", "https://127.0.0.1",
		"http://localhost", "https://localhost",
		"http://[::1]", "https://[::1]",
	} {
		if strings.HasPrefix(origin, prefix) {
			return true
		}
	}
	return false
}

func (c *WhatsAppChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting WhatsApp channel (whatsmeow)", map[string]any{"store": c.storePath})

	// Reset lifecycle state from any previous Stop() so a restarted channel
	// behaves correctly.  Use reconnectMu to be consistent with eventHandler
	// and Stop() which coordinate under the same lock.
	c.reconnectMu.Lock()
	c.stopping.Store(false)
	c.reconnecting = false
	c.reconnectMu.Unlock()

	if err := os.MkdirAll(c.storePath, 0o700); err != nil {
		return fmt.Errorf("create session store dir: %w", err)
	}

	dbPath := filepath.Join(c.storePath, whatsappDBName)
	connStr := "file:" + dbPath + "?_foreign_keys=on"

	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return fmt.Errorf("open whatsapp store: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	waLogger := waLog.Stdout("WhatsApp", "WARN", true)
	container := sqlstore.NewWithDB(db, sqliteDriver, waLogger)
	if err = container.Upgrade(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("open whatsapp store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return fmt.Errorf("get device store: %w", err)
	}

	store.DeviceProps.Os = proto.String("OK")
	client := whatsmeow.NewClient(deviceStore, waLogger)

	// Create runCtx/runCancel BEFORE registering event handler and starting
	// goroutines so that Stop() can cancel them at any time, including during
	// the QR-login flow.
	c.runCtx, c.runCancel = context.WithCancel(ctx)

	client.AddEventHandler(c.eventHandler)

	c.mu.Lock()
	c.container = container
	c.client = client
	c.mu.Unlock()

	// cleanupOnError clears struct references and releases resources when
	// Start() fails after fields are already assigned.  This prevents
	// Stop() from operating on stale references (double-close, disconnect
	// of a partially-initialized client, or stray event handler callbacks).
	startOK := false
	defer func() {
		if startOK {
			return
		}
		c.runCancel()
		client.Disconnect()
		c.mu.Lock()
		c.client = nil
		c.container = nil
		c.mu.Unlock()
		_ = container.Close()
	}()

	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(c.runCtx)
		if err != nil {
			return fmt.Errorf("get QR channel: %w", err)
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		// Handle QR events in a background goroutine so Start() returns
		// promptly.  The goroutine is tracked via c.wg and respects
		// c.runCtx for cancellation.
		// Guard wg.Add with reconnectMu + stopping check (same protocol
		// as eventHandler) so a concurrent Stop() cannot enter wg.Wait()
		// while we call wg.Add(1).
		c.reconnectMu.Lock()
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return fmt.Errorf("channel stopped during QR setup")
		}
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			for {
				select {
				case <-c.runCtx.Done():
					return
				case evt, ok := <-qrChan:
					if !ok {
						return
					}
					if evt.Event == "code" {
						logger.InfoCF("whatsapp", "QR code available — scan via web UI or Linked Devices", nil)
						c.publishQR(QREvent{Event: "code", Code: evt.Code})
					} else if evt.Event == "success" {
						logger.InfoCF("whatsapp", "WhatsApp paired successfully", nil)
						c.publishQR(QREvent{Event: "success", Paired: true})
					} else {
						logger.InfoCF("whatsapp", "WhatsApp login event", map[string]any{"event": evt.Event})
						c.publishQR(QREvent{Event: evt.Event})
					}
				}
			}
		}()
	} else {
		// Already paired — publish state for any SSE listeners.
		c.publishQR(QREvent{Event: "paired", Paired: true})
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
	}

	startOK = true
	c.SetRunning(true)
	logger.InfoC("whatsapp", "WhatsApp channel connected")
	return nil
}

func (c *WhatsAppChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsapp", "Stopping WhatsApp channel")

	// Mark as stopping under reconnectMu so the flag is visible to
	// eventHandler atomically with respect to its wg.Add(1) call.
	// This closes the TOCTOU window where eventHandler could check
	// stopping (false), then Stop sets it true + enters wg.Wait,
	// then eventHandler calls wg.Add(1) — causing a panic.
	c.reconnectMu.Lock()
	c.stopping.Store(true)
	c.reconnectMu.Unlock()

	if c.runCancel != nil {
		c.runCancel()
	}

	// Close all QR SSE subscribers.
	c.qrMu.Lock()
	for ch := range c.qrSubscribers {
		close(ch)
		delete(c.qrSubscribers, ch)
	}
	c.qrCurrent = nil
	c.qrMu.Unlock()

	// Disconnect the client first so any blocking Connect()/reconnect loops
	// can be interrupted before we wait on the goroutines.
	c.mu.Lock()
	client := c.client
	container := c.container
	c.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}

	// Wait for background goroutines (QR handler, reconnect) to finish in a
	// context-aware way so Stop can be bounded by ctx.
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines have finished.
	case <-ctx.Done():
		// Context canceled or timed out; log and proceed with best-effort cleanup.
		logger.WarnC("whatsapp", fmt.Sprintf("Stop context canceled before all goroutines finished: %v", ctx.Err()))
	}

	// Now it is safe to clear and close resources.
	c.mu.Lock()
	c.client = nil
	c.container = nil
	c.mu.Unlock()

	if container != nil {
		_ = container.Close()
	}
	c.SetRunning(false)
	return nil
}

// publishQR updates the current QR state and fans out to all SSE subscribers.
func (c *WhatsAppChannel) publishQR(evt QREvent) {
	c.qrMu.Lock()
	defer c.qrMu.Unlock()

	c.qrCurrent = &evt
	for ch := range c.qrSubscribers {
		select {
		case ch <- evt:
		default:
			// Slow subscriber — drop event to avoid blocking.
		}
	}
}

// handleQRStream serves an SSE stream of QR code events.
func (c *WhatsAppChannel) handleQRStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe
	ch := make(chan QREvent, 4)
	c.qrMu.Lock()
	c.qrSubscribers[ch] = struct{}{}
	current := c.qrCurrent
	c.qrMu.Unlock()

	defer func() {
		c.qrMu.Lock()
		delete(c.qrSubscribers, ch)
		c.qrMu.Unlock()
	}()

	// Send current state immediately so the client doesn't wait.
	if current != nil {
		writeSSE(w, *current)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, open := <-ch:
			if !open {
				return
			}
			writeSSE(w, evt)
			flusher.Flush()
		}
	}
}

// handleQRCurrent returns the current QR state as JSON (polling fallback).
func (c *WhatsAppChannel) handleQRCurrent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	c.qrMu.RLock()
	current := c.qrCurrent
	c.qrMu.RUnlock()

	if current == nil {
		_, _ = w.Write([]byte(`{"event":"unknown","paired":false}`))
		return
	}
	data, _ := json.Marshal(current)
	_, _ = w.Write(data)
}

func writeSSE(w http.ResponseWriter, evt QREvent) {
	data, _ := json.Marshal(evt)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func (c *WhatsAppChannel) eventHandler(evt any) {
	switch evt.(type) {
	case *events.Message:
		c.handleIncoming(evt.(*events.Message))
	case *events.Disconnected:
		logger.InfoCF("whatsapp", "WhatsApp disconnected, will attempt reconnection", nil)
		c.reconnectMu.Lock()
		if c.reconnecting {
			c.reconnectMu.Unlock()
			return
		}
		// Check stopping while holding the lock so the check and wg.Add
		// are atomic with respect to Stop() setting the flag + calling
		// wg.Wait(). This prevents the TOCTOU race.
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return
		}
		c.reconnecting = true
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			c.reconnectWithBackoff()
		}()
	}
}

func (c *WhatsAppChannel) reconnectWithBackoff() {
	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	backoff := reconnectInitial
	for {
		select {
		case <-c.runCtx.Done():
			return
		default:
		}

		c.mu.Lock()
		client := c.client
		c.mu.Unlock()
		if client == nil {
			return
		}

		logger.InfoCF("whatsapp", "WhatsApp reconnecting", map[string]any{"backoff": backoff.String()})
		err := client.Connect()
		if err == nil {
			logger.InfoC("whatsapp", "WhatsApp reconnected")
			return
		}

		logger.WarnCF("whatsapp", "WhatsApp reconnect failed", map[string]any{"error": err.Error()})

		select {
		case <-c.runCtx.Done():
			return
		case <-time.After(backoff):
			if backoff < reconnectMax {
				next := time.Duration(float64(backoff) * reconnectMultiplier)
				if next > reconnectMax {
					next = reconnectMax
				}
				backoff = next
			}
		}
	}
}

func (c *WhatsAppChannel) handleIncoming(evt *events.Message) {
	if evt.Message == nil {
		return
	}

	// Skip messages sent by the bot itself to prevent self-reply loops
	if evt.Info.IsFromMe && c.isSentByBot(evt.Info.ID) {
		return
	}

	senderID := evt.Info.Sender.String()
	chatID := evt.Info.Chat.String()
	content := evt.Message.GetConversation()
	if content == "" && evt.Message.ExtendedTextMessage != nil {
		content = evt.Message.ExtendedTextMessage.GetText()
	}
	content = utils.SanitizeMessageContent(content)

	// Audio/voice message download
	var mediaPaths []string
	scope := channels.BuildMediaScope("whatsapp", chatID, evt.Info.ID)
	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "whatsapp",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath
	}

	if audio := evt.Message.GetAudioMessage(); audio != nil {
		data, err := c.client.Download(c.runCtx, audio)
		if err == nil {
			ext := ".ogg"
			if audio.GetMimetype() == "audio/mp4" {
				ext = ".m4a"
			}
			tmpPath := filepath.Join(os.TempDir(), "ok_media", fmt.Sprintf("wa_%s%s", evt.Info.ID, ext))
			os.MkdirAll(filepath.Dir(tmpPath), 0o755)
			if err := os.WriteFile(tmpPath, data, 0o644); err == nil {
				mediaPaths = append(mediaPaths, storeMedia(tmpPath, "voice"+ext))
				if content != "" {
					content += "\n"
				}
				content += "[voice]"
			}
		} else {
			logger.WarnCF("whatsapp", "Failed to download audio", map[string]any{"error": err.Error()})
		}
	}

	if content == "" && len(mediaPaths) == 0 {
		return
	}

	metadata := make(map[string]string)
	metadata["message_id"] = evt.Info.ID
	if evt.Info.PushName != "" {
		metadata["user_name"] = evt.Info.PushName
	}
	if evt.Info.Chat.Server == types.GroupServer {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = chatID
	} else {
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID
	}

	peerKind := "direct"
	if evt.Info.Chat.Server == types.GroupServer {
		peerKind = "group"
	}
	peer := msgbus.Peer{Kind: peerKind, ID: chatID}
	messageID := evt.Info.ID
	sender := msgbus.SenderInfo{
		Platform:    "whatsapp",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("whatsapp", senderID),
		DisplayName: evt.Info.PushName,
	}

	if !c.isAllowedMessage(evt.Info.Chat, senderID, chatID, sender, evt.Info.IsFromMe) {
		return
	}

	logger.DebugCF(
		"whatsapp",
		"WhatsApp message received",
		map[string]any{"sender_id": senderID, "content_preview": utils.Truncate(content, 50)},
	)
	c.HandleMessage(c.runCtx, peer, messageID, senderID, chatID, content, mediaPaths, metadata, sender)
}

// isAllowedMessage checks whether an incoming message should be processed.
// Group messages require allow_groups=true AND the group JID to be in allowed_groups.
// Direct messages require allow_direct=true AND the sender to be in allowed_contacts.
// If allow_self is true, messages from the connected account are always allowed.
// Empty list = nobody allowed (reject all).
func (c *WhatsAppChannel) isAllowedMessage(chat types.JID, senderID, chatID string, sender msgbus.SenderInfo, isFromMe bool) bool {
	// Allow messages from self (uses IsFromMe flag which handles LID format correctly)
	if c.config.AllowSelf && isFromMe {
		return true
	}

	if chat.Server == types.GroupServer {
		if !c.config.AllowGroups {
			return false
		}
		return matchJID(chatID, c.config.AllowedGroups)
	}
	if !c.config.AllowDirect {
		return false
	}
	return matchJID(senderID, c.config.AllowedContacts)
}

// matchJID checks if a JID (e.g. "5511999999999@s.whatsapp.net") matches any entry in the list.
// Entries can be full JIDs or just the user part (phone number / group ID).
func matchJID(jid string, allowed []string) bool {
	userPart := jid
	if idx := strings.Index(jid, "@"); idx > 0 {
		userPart = jid[:idx]
	}
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if a == jid || a == userPart {
			return true
		}
		// Strip server from allowed entry too for comparison
		allowedUser := a
		if idx := strings.Index(a, "@"); idx > 0 {
			allowedUser = a[:idx]
		}
		if allowedUser == userPart {
			return true
		}
	}
	return false
}

// StartTyping implements channels.TypingCapable.
func (c *WhatsAppChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return func() {}, nil
	}

	to, err := parseJID(chatID)
	if err != nil {
		return func() {}, nil
	}

	_ = client.SendChatPresence(ctx, to, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	return func() {
		_ = client.SendChatPresence(context.Background(), to, types.ChatPresencePaused, types.ChatPresenceMediaText)
	}, nil
}

// typingDelay returns a delay between 3-7 seconds proportional to content length.
func typingDelay(contentLen int) time.Duration {
	const minDelay = 3 * time.Second
	const maxDelay = 7 * time.Second
	const maxLen = 500 // messages >= 500 chars get max delay

	if contentLen <= 0 {
		return minDelay
	}
	if contentLen >= maxLen {
		return maxDelay
	}
	ratio := float64(contentLen) / float64(maxLen)
	return minDelay + time.Duration(ratio*float64(maxDelay-minDelay))
}

func (c *WhatsAppChannel) Send(ctx context.Context, msg msgbus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}

	// Detect unpaired state: the client is connected (to WhatsApp servers)
	// but has not completed QR-login yet, so sending would fail.
	if client.Store.ID == nil {
		return fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := parseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, err)
	}

	// Show typing indicator with delay proportional to response length
	_ = client.SendChatPresence(ctx, to, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	delay := typingDelay(len(msg.Content))
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	_ = client.SendChatPresence(ctx, to, types.ChatPresencePaused, types.ChatPresenceMediaText)

	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	resp, err := client.SendMessage(ctx, to, waMsg)
	if err != nil {
		return fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}
	// Track sent message ID to prevent self-reply loops
	c.trackSentID(resp.ID)
	return nil
}

// trackSentID records a message ID sent by the bot to prevent self-reply loops.
func (c *WhatsAppChannel) trackSentID(id string) {
	c.sentMu.Lock()
	defer c.sentMu.Unlock()
	if c.sentIDs == nil {
		c.sentIDs = make(map[string]struct{})
	}
	c.sentIDs[id] = struct{}{}
	// Cap at 1000 entries to prevent unbounded growth
	if len(c.sentIDs) > 1000 {
		c.sentIDs = make(map[string]struct{})
	}
}

// isSentByBot checks if a message ID was sent by the bot itself.
func (c *WhatsAppChannel) isSentByBot(id string) bool {
	c.sentMu.Lock()
	defer c.sentMu.Unlock()
	if c.sentIDs == nil {
		return false
	}
	_, ok := c.sentIDs[id]
	if ok {
		delete(c.sentIDs, id)
	}
	return ok
}

// parseJID converts a chat ID (phone number or JID string) to types.JID.
func parseJID(s string) (types.JID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return types.JID{}, fmt.Errorf("empty chat id")
	}
	if strings.Contains(s, "@") {
		return types.ParseJID(s)
	}
	return types.NewJID(s, types.DefaultUserServer), nil
}
