package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	"go.uber.org/zap"
)

type ChannelStatus struct {
	WhatsAppEnabled bool
	TelegramEnabled bool
	DiscordEnabled  bool
}

type AgentHandler struct {
	agentService   *application.AgentService
	confirmManager *agent.ConfirmationManager
	channelStatus  ChannelStatus
	log            *zap.Logger
	cancelFunc     context.CancelFunc
}

func NewAgentHandler(agentService *application.AgentService, confirmManager *agent.ConfirmationManager, channelStatus ChannelStatus, log *zap.Logger) *AgentHandler {
	return &AgentHandler{
		agentService:   agentService,
		confirmManager: confirmManager,
		channelStatus:  channelStatus,
		log:            log.Named("handler.agent"),
	}
}

func (h *AgentHandler) Run(c *fiber.Ctx) error {
	var req struct {
		Input string `json:"input"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Input == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "input required",
		})
	}

	h.log.Debug("agent run", zap.String("input", req.Input))

	resp, err := h.agentService.Run(c.Context(), req.Input)
	if err != nil {
		h.log.Debug("agent run failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resp)
}

func (h *AgentHandler) Stream(c *fiber.Ctx) error {
	var req struct {
		Input string `json:"input"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Input == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "input required",
		})
	}

	h.log.Debug("agent stream", zap.String("input", req.Input))

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		h.cancelFunc = cancel

		onEvent := func(event domain.AgentEvent) {
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}

		h.agentService.RunStream(ctx, req.Input, onEvent)
		h.cancelFunc = nil
	})

	return nil
}

func (h *AgentHandler) Status(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"state":             "idle",
		"whatsapp_enabled":  h.channelStatus.WhatsAppEnabled,
		"telegram_enabled":  h.channelStatus.TelegramEnabled,
		"discord_enabled":   h.channelStatus.DiscordEnabled,
	})
}

func (h *AgentHandler) GetExecution(c *fiber.Ctx) error {
	id := c.Params("id")
	record, err := h.agentService.GetExecution(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if record == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "execution not found"})
	}
	return c.JSON(record)
}

func (h *AgentHandler) ListExecutions(c *fiber.Ctx) error {
	records, err := h.agentService.GetRecentExecutions(20)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if records == nil {
		records = []domain.ExecutionRecord{}
	}
	return c.JSON(records)
}

func (h *AgentHandler) GetLimits(c *fiber.Ctx) error {
	return c.JSON(h.agentService.GetLimits())
}

func (h *AgentHandler) SetLimits(c *fiber.Ctx) error {
	var limits domain.AgentLimits
	if err := c.BodyParser(&limits); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid request body"})
	}

	if limits.MaxSteps <= 0 || limits.MaxAttempts <= 0 || limits.TimeoutMs <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "max_steps, max_attempts and timeout_ms must be > 0"})
	}

	if err := h.agentService.SetLimits(c.Context(), limits); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(limits)
}

func (h *AgentHandler) Metrics(c *fiber.Ctx) error {
	metrics, err := h.agentService.GetMetrics()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if metrics == nil {
		return c.JSON(domain.ExecutionMetrics{
			ToolUsageCount: map[string]int{},
		})
	}
	return c.JSON(metrics)
}

func (h *AgentHandler) ListTools(c *fiber.Ctx) error {
	return c.JSON(h.agentService.ListTools())
}

func (h *AgentHandler) ListSkills(c *fiber.Ctx) error {
	skills := h.agentService.ListSkills()
	if skills == nil {
		skills = []map[string]string{}
	}
	return c.JSON(skills)
}

func (h *AgentHandler) GetConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	repo := h.agentService.GetConfigRepo()
	if repo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "config not available"})
	}
	value, err := repo.Get(c.Context(), key)
	if err != nil || value == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "key not found"})
	}
	return c.JSON(fiber.Map{"key": key, "value": value})
}

func (h *AgentHandler) SetConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	var req struct {
		Value string `json:"value"`
	}
	if err := c.BodyParser(&req); err != nil || req.Value == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "value required"})
	}
	repo := h.agentService.GetConfigRepo()
	if repo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "config not available"})
	}
	if err := repo.Set(c.Context(), key, req.Value); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	// Recarregar templates em memoria para qualquer key de template
	h.agentService.ReloadSoul()
	return c.JSON(fiber.Map{"key": key, "value": req.Value})
}

func (h *AgentHandler) Confirm(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Approved bool `json:"approved"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid request body"})
	}
	if h.confirmManager == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "confirmation not available"})
	}
	if err := h.confirmManager.Respond(id, req.Approved); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *AgentHandler) Cancel(c *fiber.Ctx) error {
	if h.cancelFunc != nil {
		h.cancelFunc()
		h.cancelFunc = nil
		return c.JSON(fiber.Map{"status": "cancelled"})
	}
	return c.JSON(fiber.Map{"status": "not_running"})
}
