// internal/handler/http/v1/user_handler.go
package v1

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Register регистрирует нового пользователя
// @Summary Регистрация пользователя
// @Description Создает нового пользователя в системе
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные для регистрации"
// @Success 201 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	const op = "handler.UserHandler.Register"

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid registration request",
			slog.String("op", op),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	if len(req.Roles) == 0 {
		req.Roles = []string{"volunteer"}
	}

	reg := domain.UserRegistration{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
		Roles:    req.Roles,
	}

	user, err := h.userService.Register(c.Request.Context(), reg)
	if err != nil {
		switch {
		case err.Error() == "service.UserService.Register: email already registered":
			slog.Warn("email already registered",
				slog.String("op", op),
				slog.String("email", req.Email))
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error:   "Email already registered",
			})
			return

		case err.Error() == "service.UserService.Register: invalid role":
			slog.Warn("invalid role in registration",
				slog.String("op", op),
				slog.String("email", req.Email),
				slog.Any("roles", req.Roles))
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error:   "Invalid role specified",
			})
			return

		default:
			slog.Error("failed to register user",
				slog.String("op", op),
				slog.String("email", req.Email),
				slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error:   "Failed to register user",
			})
			return
		}
	}

	response := LoginResponse{
		Message: "User registered successfully",
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Phone:     user.Phone,
			Roles:     user.GetRoles(),
			IsBlocked: user.IsBlocked,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}

	slog.Info("user registered successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))

	c.JSON(http.StatusCreated, response)
}

// GetUserByID возвращает пользователя по ID
// @Summary Получить пользователя по ID
// @Description Возвращает информацию о пользователе по его идентификатору
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	const op = "handler.UserHandler.GetUserByID"

	userIDStr := c.Param("id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		slog.Warn("invalid UUID format",
			slog.String("op", op),
			slog.String("user_id", userIDStr))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid user ID format",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		slog.Warn("user ID not found in context",
			slog.String("op", op))
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	currentUser, err := h.userService.UserByParam(c.Request.Context(), "id", currentUserID.(string))
	if err != nil {
		slog.Error("failed to get current user",
			slog.String("op", op),
			slog.String("user_id", currentUserID.(string)),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to verify permissions",
		})
		return
	}

	if currentUser.ID != userID && !currentUser.IsAdmin() {
		slog.Warn("unauthorized access to user data",
			slog.String("op", op),
			slog.String("requested_user_id", userID.String()),
			slog.String("current_user_id", currentUser.ID.String()))
		c.JSON(http.StatusForbidden, ErrorResponse{
			Success: false,
			Error:   "Access denied",
		})
		return
	}

	user, err := h.userService.UserByParam(c.Request.Context(), "id", userID.String())
	if err != nil {
		switch {
		case err.Error() == "service.UserService.GetUserByID: user not found":
			slog.Warn("user not found",
				slog.String("op", op),
				slog.String("user_id", userID.String()))
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error:   "User not found",
			})
			return

		default:
			slog.Error("failed to get user",
				slog.String("op", op),
				slog.String("user_id", userID.String()),
				slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Success: false,
				Error:   "Failed to get user",
			})
			return
		}
	}

	response := UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Roles:     user.GetRoles(),
		IsBlocked: user.IsBlocked,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// GetCurrentUser возвращает текущего аутентифицированного пользователя
// @Summary Получить текущего пользователя
// @Description Возвращает информацию о текущем аутентифицированном пользователе
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	const op = "handler.UserHandler.GetCurrentUser"

	userID, exists := c.Get("user_id")
	if !exists {
		slog.Warn("user ID not found in context",
			slog.String("op", op))
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	user, err := h.userService.UserByParam(c.Request.Context(), "id", userID.(string))
	if err != nil {
		slog.Error("failed to get current user",
			slog.String("op", op),
			slog.String("user_id", userID.(string)),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to get user information",
		})
		return
	}

	// Формируем ответ
	response := UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Roles:     user.GetRoles(),
		IsBlocked: user.IsBlocked,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateUser обновляет данные пользователя
// @Summary Обновить данные пользователя
// @Description Обновляет информацию о текущем пользователе
// @Tags users
// @Accept json
// @Produce json
// @Param request body UpdateUserRequest true "Данные для обновления"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/users/me [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	const op = "handler.UserHandler.UpdateUser"

	userID, exists := c.Get("user_id")
	if !exists {
		slog.Warn("user ID not found in context",
			slog.String("op", op))
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid update request",
			slog.String("op", op),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	user, err := h.userService.UserByParam(c.Request.Context(), "id", userID.(string))

	if err != nil {
		slog.Error("failed to get user for update",
			slog.String("op", op),
			slog.String("user_id", userID.(string)),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to get user",
		})
		return
	}

	user.UpdateProfile(req.FullName, req.Phone)

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), user)
	if err != nil {
		slog.Error("failed to update user",
			slog.String("op", op),
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Failed to update user",
		})
		return
	}

	response := UserResponse{
		ID:        updatedUser.ID,
		Email:     updatedUser.Email,
		FullName:  updatedUser.FullName,
		Phone:     updatedUser.Phone,
		Roles:     updatedUser.GetRoles(),
		IsBlocked: updatedUser.IsBlocked,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	}

	slog.Info("user updated successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()))

	c.JSON(http.StatusOK, response)
}

// ChangePassword меняет пароль пользователя
// @Summary Сменить пароль
// @Description Изменяет пароль текущего пользователя
// @Tags users
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "Данные для смены пароля"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	const op = "handler.UserHandler.ChangePassword"

	userIDVal, exists := c.Get("user_id")
	if !exists {
		slog.Warn("user ID not found in context", slog.String("op", op))
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		slog.Warn("user ID has invalid type in context",
			slog.String("op", op),
			slog.Any("user_id", userIDVal))
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		slog.Warn("invalid user ID format in context",
			slog.String("op", op),
			slog.String("user_id", userIDStr),
			slog.Any("error", err))
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid password change request",
			slog.String("op", op),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		slog.Warn("empty password fields",
			slog.String("op", op),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request format"})
		return
	}

	// 3) service call
	if err := h.userService.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		switch {
		case err.Error() == "service.UserService.ChangePassword: invalid current password":
			slog.Warn("invalid current password",
				slog.String("op", op),
				slog.String("user_id", userID.String()))
			c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid current password"})
			return

		case err.Error() == "service.UserService.ChangePassword: user not found":
			slog.Warn("user not found while changing password",
				slog.String("op", op),
				slog.String("user_id", userID.String()))
			c.JSON(http.StatusNotFound, ErrorResponse{Success: false, Error: "User not found"})
			return

		default:
			slog.Error("failed to change password",
				slog.String("op", op),
				slog.String("user_id", userID.String()),
				slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to change password"})
			return
		}
	}

	slog.Info("password changed successfully",
		slog.String("op", op),
		slog.String("user_id", userID.String()))

	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Password changed successfully"})
}

// GetUsers возвращает список пользователей (только для админов)
// @Summary Получить список пользователей
// @Description Возвращает список всех пользователей (только для администраторов) с пагинацией и необязательным фильтром по роли.
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Номер страницы (>= 1)" default(1)
// @Param limit query int false "Количество записей на странице (1..200)" default(20)
// @Param role query string false "Фильтр по роли (например: admin, volunteer, user)"
// @Success 200 {array} UserResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	const op = "handler.UserHandler.GetUsers"

	userIDVal, exists := c.Get("user_id")
	if !exists {
		slog.Warn("user ID not found in context", slog.String("op", op))
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		slog.Warn("user ID has invalid type in context",
			slog.String("op", op),
			slog.Any("user_id", userIDVal))
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	isAdmin := false

	if rolesVal, ok := c.Get("roles"); ok {
		switch v := rolesVal.(type) {
		case []string:
			for _, r := range v {
				if r == "admin" {
					isAdmin = true
					break
				}
			}
		case []interface{}:
			for _, it := range v {
				rs, _ := it.(string)
				if rs == "admin" {
					isAdmin = true
					break
				}
			}
		default:
			slog.Warn("roles have unexpected type in context",
				slog.String("op", op),
				slog.Any("roles_type", fmt.Sprintf("%T", rolesVal)))
		}
	}

	if !isAdmin {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			slog.Warn("invalid UUID format in context",
				slog.String("op", op),
				slog.String("user_id", userIDStr),
				slog.Any("error", err))
			c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
			return
		}

		currentUser, err := h.userService.UserByParam(c.Request.Context(), "id", id.String())
		if err != nil {
			slog.Error("failed to get current user for admin check",
				slog.String("op", op),
				slog.String("user_id", userIDStr),
				slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to verify permissions"})
			return
		}

		isAdmin = currentUser.IsAdmin()
	}

	if !isAdmin {
		slog.Warn("non-admin user tried to access user list",
			slog.String("op", op),
			slog.String("user_id", userIDStr))
		c.JSON(http.StatusForbidden, ErrorResponse{Success: false, Error: "Access denied. Admin only."})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	role := c.Query("role")

	page := 1
	if p, err := strconv.Atoi(pageStr); err != nil || p < 1 {
		slog.Warn("invalid page query, using default",
			slog.String("op", op),
			slog.String("page", pageStr))
		pageStr = "1"
	} else {
		page = p
	}

	limit := 20
	if l, err := strconv.Atoi(limitStr); err != nil || l < 1 {
		slog.Warn("invalid limit query, using default",
			slog.String("op", op),
			slog.String("limit", limitStr))
		limitStr = "20"
	} else {
		if l > 200 {
			l = 200
		}
		limit = l
		limitStr = strconv.Itoa(limit)
	}

	users, total, err := h.userService.GetUsers(c.Request.Context(), strconv.Itoa(page), strconv.Itoa(limit), role)
	if err != nil {
		slog.Error("failed to get users list",
			slog.String("op", op),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to get users"})
		return
	}

	resp := make([]UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, UserResponse{
			ID:        u.ID,
			Email:     u.Email,
			FullName:  u.FullName,
			Phone:     u.Phone,
			Roles:     u.GetRoles(),
			IsBlocked: u.IsBlocked,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}

	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	c.Header("X-Page", pageStr)
	c.Header("X-Limit", limitStr)

	c.JSON(http.StatusOK, resp)
}
