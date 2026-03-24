package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"go.uber.org/zap"
)

const maxImportSize = 100 * 1024 * 1024 // 100MB

type ImportHandler struct {
	importService *application.ImportService
	log           *zap.Logger
}

func NewImportHandler(importService *application.ImportService, log *zap.Logger) *ImportHandler {
	return &ImportHandler{
		importService: importService,
		log:           log.Named("handler.import"),
	}
}

func (h *ImportHandler) ImportChatGPT(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "arquivo nao enviado",
		})
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "apenas arquivos .zip sao aceitos",
		})
	}

	if file.Size > maxImportSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "arquivo muito grande (max 100MB)",
		})
	}

	reader, err := file.Open()
	if err != nil {
		h.log.Debug("open uploaded file failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to read file",
		})
	}
	defer reader.Close()

	count, err := h.importService.ImportChatGPT(c.Context(), reader)
	if err != nil {
		h.log.Debug("import chatgpt failed", zap.Error(err))
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": fmt.Sprintf("falha ao importar: %s", err.Error()),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": fmt.Sprintf("%d conversas importadas", count),
		"count":   count,
	})
}
