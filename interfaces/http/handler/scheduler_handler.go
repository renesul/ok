package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"go.uber.org/zap"
)

type SchedulerHandler struct {
	schedulerService *application.SchedulerService
	log              *zap.Logger
}

func NewSchedulerHandler(schedulerService *application.SchedulerService, log *zap.Logger) *SchedulerHandler {
	return &SchedulerHandler{
		schedulerService: schedulerService,
		log:              log.Named("handler.scheduler"),
	}
}

func (h *SchedulerHandler) List(c *fiber.Ctx) error {
	jobs, err := h.schedulerService.ListJobs(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(jobs)
}

func (h *SchedulerHandler) Create(c *fiber.Ctx) error {
	var req struct {
		Name            string `json:"name"`
		TaskType        string `json:"task_type"`
		Input           string `json:"input"`
		IntervalSeconds int    `json:"interval_seconds"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid request body"})
	}

	job, err := h.schedulerService.CreateJob(c.Context(), req.Name, req.TaskType, req.Input, req.IntervalSeconds)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(job)
}

func (h *SchedulerHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Enabled         *bool `json:"enabled"`
		IntervalSeconds *int  `json:"interval_seconds"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid request body"})
	}

	job, err := h.schedulerService.UpdateJob(c.Context(), id, req.Enabled, req.IntervalSeconds)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(job)
}

func (h *SchedulerHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.schedulerService.DeleteJob(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
