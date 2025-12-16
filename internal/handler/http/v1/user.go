package v1

import (
	"context"
	"log/slog"
	"net/http"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/repository/UserRepository"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(ctx context.Context, db *pgxpool.Pool) *UserHandler {
	userRepo := repository.NewUserRepository(db)

	userService := service.NewUserService(userRepo)

	return &UserHandler{
		userService: userService,
	}
}

// CreateUserHandler Объединенный хендлер для создания пользователя
func (h *UserHandler) CreateUserHandler(c *gin.Context) {
	const op = "handler.UserHandler.CreateUserHandler"

	var req domain.UserRegistration

	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid request",
			slog.String("op", op),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	userID, err := h.userService.Register(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "service.UserService.AddNewUser: email already registered" {
			slog.Warn("email already registered",
				slog.String("op", op),
				slog.String("email", req.Email))
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already registered",
			})
			return
		}

		slog.Error("failed to create user",
			slog.String("op", op),
			slog.String("email", req.Email),
			slog.Any("role", req.Role),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	slog.Info("user created successfully",
		slog.String("op", op),
		slog.String("user_id", userID),
		slog.String("email", req.Email),
		slog.Any("role", req.Role))

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user_id": userID,
		"email":   req.Email,
		"role":    req.Role,
	})
}

// UserByUUIDHandler - получает пользователя по ID
func (h *UserHandler) UserByUUIDHandler(c *gin.Context) {
	const op = "handler.UserHandler.UserByUUIDHandler"

	userIDStr := c.Param("id")

	_, err := uuid.Parse(userIDStr)
	if err != nil {
		slog.Warn("invalid UUID format",
			slog.String("op", op),
			slog.String("user_id", userIDStr))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	user, err := h.userService.UserByParam(c.Request.Context(), "id", userIDStr)
	if err != nil {
		if err.Error() == "service.UserService.UserByParam: user not found" {
			slog.Warn("user not found",
				slog.String("op", op),
				slog.String("user_id", userIDStr))
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		slog.Error("failed to get user",
			slog.String("op", op),
			slog.String("user_id", userIDStr),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user",
		})
		return
	}

	user.PasswordHash = ""

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
