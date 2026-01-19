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

// Register регистрирует пользователя и сразу возвращает токены и сессию (автологин)
// @Summary Регистрация
// @Description Создаёт пользователя и возвращает access/refresh токены + сессию
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные регистрации"
// @Success 201 {object} LoginResponse
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
