package handler

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/infrastructure/embedding"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

const healthCacheTTL = 60 * time.Second

type HealthHandler struct {
	llmClient   *llm.Client
	llmConfig   llm.ClientConfig
	embedClient *embedding.Client
	log         *zap.Logger

	cacheMu     sync.RWMutex
	cacheResult fiber.Map
	cacheExpire time.Time
}

func NewHealthHandler(llmClient *llm.Client, llmConfig llm.ClientConfig, embedClient *embedding.Client, log *zap.Logger) *HealthHandler {
	return &HealthHandler{
		llmClient:   llmClient,
		llmConfig:   llmConfig,
		embedClient: embedClient,
		log:         log.Named("handler.health"),
	}
}

type serviceStatus struct {
	Status    string `json:"status"`
	Model     string `json:"model,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (h *HealthHandler) CheckServices(c *fiber.Ctx) error {
	// Retornar cache se valido
	h.cacheMu.RLock()
	if h.cacheResult != nil && time.Now().Before(h.cacheExpire) {
		result := h.cacheResult
		h.cacheMu.RUnlock()
		return c.JSON(result)
	}
	h.cacheMu.RUnlock()

	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	var llmStatus, embedStatus serviceStatus
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		llmStatus = h.checkLLM(ctx)
	}()

	go func() {
		defer wg.Done()
		embedStatus = h.checkEmbedding(ctx)
	}()

	wg.Wait()

	result := fiber.Map{
		"llm":       llmStatus,
		"embedding": embedStatus,
	}

	// Salvar no cache
	h.cacheMu.Lock()
	h.cacheResult = result
	h.cacheExpire = time.Now().Add(healthCacheTTL)
	h.cacheMu.Unlock()

	return c.JSON(result)
}

func (h *HealthHandler) checkLLM(ctx context.Context) serviceStatus {
	if h.llmConfig.BaseURL == "" || h.llmConfig.Model == "" {
		return serviceStatus{Status: "disabled", Model: h.llmConfig.Model}
	}

	start := time.Now()
	err := h.llmClient.Ping(ctx, h.llmConfig)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		h.log.Debug("llm ping failed", zap.Error(err))
		return serviceStatus{Status: "error", Model: h.llmConfig.Model, LatencyMs: latency, Error: err.Error()}
	}

	return serviceStatus{Status: "ok", Model: h.llmConfig.Model, LatencyMs: latency}
}

func (h *HealthHandler) checkEmbedding(ctx context.Context) serviceStatus {
	if !h.embedClient.Enabled() {
		return serviceStatus{Status: "disabled", Model: h.embedClient.Model()}
	}

	start := time.Now()
	err := h.embedClient.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		h.log.Debug("embedding ping failed", zap.Error(err))
		return serviceStatus{Status: "error", Model: h.embedClient.Model(), LatencyMs: latency, Error: err.Error()}
	}

	return serviceStatus{Status: "ok", Model: h.embedClient.Model(), LatencyMs: latency}
}
