package v1

import (
	"net"
	"net/http"
	"volunteer-coordination-server/internal/domain"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func setAuthCookies(c *gin.Context, tokens *domain.Tokens) {
	if tokens == nil {
		// ничего не ставим — это защитит от паники
		return
	}

	// access token
	c.SetCookie(
		"access_token",
		tokens.AccessToken,
		int(tokens.ExpiresIn),
		"/",
		"",
		false, // true в prod
		true,  // можно true, чтобы JS не читал access
	)

	// refresh token
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		30*24*60*60,
		"/",
		"",
		false, // true в prod
		true,  // HttpOnly обязательно
	)
}

// Register регистрирует пользователя и выполняет автологин
// @Summary Регистрация
// @Description Создаёт пользователя и устанавливает access_token/refresh_token в cookies (токены не возвращаются в теле).
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные регистрации"
// @Success 201 {object} LoginResponse "Пользователь создан и авторизован (токены в cookies)"
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request"})
		return
	}

	reg := domain.UserRegistration{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
		Roles:    req.Roles,
	}

	res, err := h.authService.Register(
		c.Request.Context(),
		reg,
		c.Request.UserAgent(),
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	setAuthCookies(c, res.Tokens)

	c.JSON(http.StatusCreated, LoginResponse{
		Message: "User registered successfully",
		User: UserResponse{
			ID:        res.User.ID,
			Email:     res.User.Email,
			FullName:  res.User.FullName,
			Phone:     res.User.Phone,
			Roles:     res.User.GetRoles(),
			IsBlocked: res.User.IsBlocked,
			CreatedAt: res.User.CreatedAt,
			UpdatedAt: res.User.UpdatedAt,
		},
		Session: SessionDTO{
			ID:        res.Session.ID,
			ExpiresAt: res.Session.ExpiresAt,
		},
	})
}

// Login логинит пользователя и устанавливает access/refresh токены в cookies
// @Summary Логин
// @Description Аутентифицирует пользователя. Токены НЕ возвращаются в теле ответа — они устанавливаются в cookies:
// @Description - access_token (короткоживущий)
// @Description - refresh_token (HttpOnly, долгоживущий)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Данные для входа"
// @Success 200 {object} LoginResponse "Успешный логин (без токенов в теле)"
// @Failure 400 {object} ErrorResponse "Некорректный формат запроса"
// @Failure 401 {object} ErrorResponse "Неверные учетные данные"
// @Router /api/v1/auth/login [post]
// @Security CookieAuth
func (h *AuthHandler) Login(c *gin.Context) {
	const op = "handler.AuthHandler.Login"

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request"})
		return
	}

	ua := c.Request.UserAgent()
	ip := c.ClientIP()

	res, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, ua, ip)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Invalid credentials"})
		return
	}

	// 🔥 ВАЖНО: кладём токены в cookies
	setAuthCookies(c, res.Tokens)

	c.JSON(http.StatusOK, LoginResponse{
		Message: "Login successful",
		User: UserResponse{
			ID:        res.User.ID,
			Email:     res.User.Email,
			FullName:  res.User.FullName,
			Phone:     res.User.Phone,
			Roles:     res.User.GetRoles(),
			IsBlocked: res.User.IsBlocked,
			CreatedAt: res.User.CreatedAt,
			UpdatedAt: res.User.UpdatedAt,
		},
		Session: SessionDTO{
			ID:        res.Session.ID,
			ExpiresAt: res.Session.ExpiresAt,
		},
	})
}

// Refresh обновляет токены по refresh_token из cookies
// @Summary Обновление токенов
// @Description Обновляет access/refresh токены. refresh_token берётся из cookie (HttpOnly).
// @Description В случае успеха сервер обновляет cookies: access_token и refresh_token.
// @Tags auth
// @Produce json
// @Success 200 {object} SuccessOnlyResponse
// @Failure 401 {object} ErrorResponse "Нет refresh_token cookie или refresh_token невалиден"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "No refresh token"})
		return
	}

	res, err := h.authService.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Invalid refresh token"})
		return
	}

	// обновляем cookies
	setAuthCookies(c, res.Tokens)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Logout очищает cookies и завершает сессию на клиенте
// @Summary Выход
// @Description Удаляет access_token и refresh_token cookies (устанавливает пустые значения с отрицательным MaxAge).
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]bool "success=true"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, false)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			return host
		}
	}
	return ip
}
