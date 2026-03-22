package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type UserHandler struct {
	service *application.UserService
	log     *zap.Logger
}

func NewUserHandler(service *application.UserService, log *zap.Logger) *UserHandler {
	return &UserHandler{service: service, log: log.Named("handler.user")}
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type updateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.Debug("invalid request body", zap.Error(err))
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	h.log.Debug("create user request", zap.String("name", req.Name), zap.String("email", req.Email))

	user := &domain.User{
		Name:  req.Name,
		Email: req.Email,
	}

	if err := h.service.CreateUser(c.Context(), user); err != nil {
		if errors.Is(err, application.ErrValidation) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		h.log.Debug("create user failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *UserHandler) Get(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	h.log.Debug("get user request", zap.Uint("id", id))

	user, err := h.service.GetUser(c.Context(), id)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		h.log.Debug("get user failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	return c.JSON(user)
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	h.log.Debug("list users request")

	users, err := h.service.ListUsers(c.Context())
	if err != nil {
		h.log.Debug("list users failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list users",
		})
	}

	return c.JSON(users)
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.Debug("invalid request body", zap.Error(err))
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	h.log.Debug("update user request", zap.Uint("id", id), zap.String("name", req.Name), zap.String("email", req.Email))

	user, err := h.service.UpdateUser(c.Context(), id, req.Name, req.Email)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		if errors.Is(err, application.ErrValidation) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		h.log.Debug("update user failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update user",
		})
	}

	return c.JSON(user)
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	h.log.Debug("delete user request", zap.Uint("id", id))

	if err := h.service.DeleteUser(c.Context(), id); err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		h.log.Debug("delete user failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete user",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func parseID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
