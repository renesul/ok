package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type ChatHandler struct {
	conversationService *application.ConversationService
	chatService         *application.ChatService
	log                 *zap.Logger
}

func NewChatHandler(conversationService *application.ConversationService, chatService *application.ChatService, log *zap.Logger) *ChatHandler {
	return &ChatHandler{
		conversationService: conversationService,
		chatService:         chatService,
		log:                 log.Named("handler.chat"),
	}
}

func (h *ChatHandler) ChatPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(readTemplate("chat.html"))
}

func (h *ChatHandler) ListConversations(c *fiber.Ctx) error {
	conversations, err := h.conversationService.ListConversations(c.Context())
	if err != nil {
		h.log.Debug("list conversations failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list conversations",
		})
	}
	return c.JSON(conversations)
}

func (h *ChatHandler) GetMessages(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid conversation id",
		})
	}

	messages, err := h.conversationService.GetMessages(c.Context(), uint(conversationID))
	if err != nil {
		if errors.Is(err, application.ErrConversationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "conversation not found",
			})
		}
		h.log.Debug("get messages failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get messages",
		})
	}
	return c.JSON(messages)
}

func (h *ChatHandler) DeleteConversation(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid conversation id",
		})
	}

	if err := h.conversationService.DeleteConversation(c.Context(), uint(conversationID)); err != nil {
		if errors.Is(err, application.ErrConversationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "conversation not found",
			})
		}
		h.log.Debug("delete conversation failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete conversation",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ChatHandler) SearchConversations(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON([]any{})
	}

	conversations, err := h.conversationService.SearchConversations(c.Context(), query)
	if err != nil {
		h.log.Debug("search conversations failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to search conversations",
		})
	}
	return c.JSON(conversations)
}

func (h *ChatHandler) CreateConversation(c *fiber.Ctx) error {
	var req struct {
		Title string `json:"title"`
	}
	c.BodyParser(&req)

	conversation, err := h.chatService.CreateConversation(c.Context(), req.Title)
	if err != nil {
		h.log.Debug("create conversation failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create conversation",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(conversation)
}

func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid conversation id",
		})
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "message content required",
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		onEvent := func(event domain.AgentEvent) {
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}

		assistantMessage, err := h.chatService.SendMessage(
			context.Background(),
			uint(conversationID),
			req.Content,
			onEvent,
		)

		if err != nil {
			h.log.Debug("send message failed", zap.Error(err))
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", errData)
			w.Flush()
			return
		}

		doneData, _ := json.Marshal(map[string]interface{}{
			"done":       true,
			"message_id": assistantMessage.ID,
		})
		fmt.Fprintf(w, "data: %s\n\n", doneData)
		w.Flush()
	})

	return nil
}
