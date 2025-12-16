// internal/handler/http/v1/user_handler.go
package v1

import (
	"log/slog"
	"net/http"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Register обрабатывает регистрацию пользователя
func (h *UserHandler) Register(c *gin.Context) {
	const op = "handler.UserHandler.Register"

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid request",
			slog.String("op", op),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Конвертируем DTO в доменный объект
	reg := domain.UserRegistration{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
		Roles:    req.Role,
	}

	user, err := h.userService.Register(c.Request.Context(), reg)
	if err != nil {
		slog.Error(op)
	}

	// Формируем ответ
	response := LoginResponse{
		Message: "User registered successfully",
		User: struct {
			ID       string   `json:"id"`
			Email    string   `json:"email"`
			FullName string   `json:"full_name"`
			Roles    []string `json:"roles"`
		}{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
			Roles:    user.GetRoles(), // Метод в домене
		},
	}

	c.JSON(http.StatusCreated, response)
}
