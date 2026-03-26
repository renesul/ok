package application

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type ImportService struct {
	conversationRepository domain.ConversationRepository
	messageRepository      domain.MessageRepository
	embeddingService       *EmbeddingService
	log                    *zap.Logger
}

func NewImportService(conversationRepository domain.ConversationRepository, messageRepository domain.MessageRepository, embeddingService *EmbeddingService, log *zap.Logger) *ImportService {
	return &ImportService{
		conversationRepository: conversationRepository,
		messageRepository:      messageRepository,
		embeddingService:       embeddingService,
		log:                    log.Named("service.import"),
	}
}

func (s *ImportService) ImportChatGPT(ctx context.Context, reader io.Reader) (int, error) {
	s.log.Debug("import chatgpt")

	const maxImportSize = 100 * 1024 * 1024 // 100MB
	data, err := io.ReadAll(io.LimitReader(reader, maxImportSize+1))
	if err != nil {
		return 0, fmt.Errorf("read zip data: %w", err)
	}
	if len(data) > maxImportSize {
		return 0, fmt.Errorf("file too large (max %d MB)", maxImportSize/1024/1024)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}

	var exports []chatGPTExport
	conversationFiles := findConversationFiles(zipReader)
	if len(conversationFiles) == 0 {
		return 0, fmt.Errorf("no conversations.json or conversations-NNN.json found in zip")
	}
	for _, file := range conversationFiles {
		rc, err := file.Open()
		if err != nil {
			return 0, fmt.Errorf("open %s: %w", file.Name, err)
		}
		fileData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", file.Name, err)
		}
		var batch []chatGPTExport
		if err := json.Unmarshal(fileData, &batch); err != nil {
			return 0, fmt.Errorf("parse %s: %w", file.Name, err)
		}
		exports = append(exports, batch...)
	}

	imported := 0
	for _, export := range exports {
		messages := linearizeMessages(export.Mapping, export.CreateTime)
		if len(messages) == 0 {
			continue
		}

		title := export.Title
		if title == "" {
			title = "Sem titulo"
		}

		conversation := &domain.Conversation{
			Title:     title,
			Source:    "import",
			Channel:   "web",
			CreatedAt: timeFromFloat(export.CreateTime),
			UpdatedAt: timeFromFloat(export.UpdateTime),
		}

		if err := s.conversationRepository.Create(ctx, conversation); err != nil {
			s.log.Debug("skip conversation", zap.String("title", title), zap.Error(err))
			continue
		}

		domainMessages := make([]domain.Message, len(messages))
		for i, msg := range messages {
			domainMessages[i] = domain.Message{
				ConversationID: conversation.ID,
				Role:           msg.role,
				Content:        msg.content,
				SortOrder:      i,
				CreatedAt:      msg.createdAt,
			}
		}

		if err := s.messageRepository.CreateBatch(ctx, domainMessages); err != nil {
			s.log.Debug("skip messages", zap.Uint("conversation_id", conversation.ID), zap.Error(err))
			continue
		}

		if err := s.messageRepository.IndexForSearch(ctx, domainMessages); err != nil {
			s.log.Debug("skip search index", zap.Uint("conversation_id", conversation.ID), zap.Error(err))
		}

		if s.embeddingService != nil {
			s.embeddingService.EmbedAndStore(ctx, domainMessages)
		}

		imported++
	}

	s.log.Debug("import completed", zap.Int("imported", imported), zap.Int("total", len(exports)))
	return imported, nil
}

// findConversationFiles returns conversation JSON files from the zip.
// Supports both legacy format (conversations.json) and new sharded format (conversations-NNN.json).
func findConversationFiles(reader *zip.Reader) []*zip.File {
	var files []*zip.File
	for _, file := range reader.File {
		base := filepath.Base(file.Name)
		if base == "conversations.json" || (strings.HasPrefix(base, "conversations-") && strings.HasSuffix(base, ".json")) {
			files = append(files, file)
		}
	}
	return files
}

type linearMessage struct {
	role      string
	content   string
	createdAt time.Time
}

func linearizeMessages(mapping map[string]chatGPTNode, fallbackTime float64) []linearMessage {
	if len(mapping) == 0 {
		return nil
	}

	// Find root node (parent is nil or empty)
	var rootID string
	for id, node := range mapping {
		if node.Parent == nil || *node.Parent == "" {
			rootID = id
			break
		}
	}

	// Also check for nodes whose parent doesn't exist in the mapping
	if rootID == "" {
		for id, node := range mapping {
			if node.Parent != nil {
				if _, exists := mapping[*node.Parent]; !exists {
					rootID = id
					break
				}
			}
		}
	}

	if rootID == "" {
		return nil
	}

	var messages []linearMessage
	walkTree(mapping, rootID, fallbackTime, &messages)
	return messages
}

func walkTree(mapping map[string]chatGPTNode, nodeID string, fallbackTime float64, messages *[]linearMessage) {
	node, exists := mapping[nodeID]
	if !exists {
		return
	}

	if node.Message != nil && node.Message.Content.ContentType == "text" {
		content := extractTextParts(node.Message.Content.Parts)
		if content != "" && (node.Message.Author.Role == "user" || node.Message.Author.Role == "assistant") {
			ts := fallbackTime
			if node.Message.CreateTime != nil {
				ts = *node.Message.CreateTime
			}
			*messages = append(*messages, linearMessage{
				role:      node.Message.Author.Role,
				content:   content,
				createdAt: timeFromFloat(ts),
			})
		}
	}

	if len(node.Children) > 0 {
		// Follow the last child (most recent branch in ChatGPT)
		walkTree(mapping, node.Children[len(node.Children)-1], fallbackTime, messages)
	}
}

func extractTextParts(parts []any) string {
	var texts []string
	for _, part := range parts {
		if text, ok := part.(string); ok && text != "" {
			texts = append(texts, text)
		}
	}
	if len(texts) == 0 {
		return ""
	}
	result := texts[0]
	for i := 1; i < len(texts); i++ {
		result += "\n" + texts[i]
	}
	return result
}

func timeFromFloat(ts float64) time.Time {
	if ts <= 0 {
		return time.Now()
	}
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}

// ChatGPT export JSON types

type chatGPTExport struct {
	Title      string                 `json:"title"`
	CreateTime float64                `json:"create_time"`
	UpdateTime float64                `json:"update_time"`
	Mapping    map[string]chatGPTNode `json:"mapping"`
}

type chatGPTNode struct {
	ID       string          `json:"id"`
	Message  *chatGPTMessage `json:"message"`
	Parent   *string         `json:"parent"`
	Children []string        `json:"children"`
}

type chatGPTMessage struct {
	Author     chatGPTAuthor  `json:"author"`
	Content    chatGPTContent `json:"content"`
	CreateTime *float64       `json:"create_time"`
}

type chatGPTAuthor struct {
	Role string `json:"role"`
}

type chatGPTContent struct {
	ContentType string `json:"content_type"`
	Parts       []any  `json:"parts"`
}

