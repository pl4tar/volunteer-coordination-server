package v1

import (
	"log/slog"
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

func (h *AuthHandler) Login(c *gin.Context) {
	const op = "handler.AuthHandler.Login"

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid login request", slog.String("op", op), slog.Any("error", err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request format", Details: err.Error()})
		return
	}

	ua := c.Request.UserAgent()
	ip := clientIP(c)

	res, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, ua, ip)
	if err != nil {
		switch err {
		case domain.ErrUserBlocked:
			c.JSON(http.StatusLocked, ErrorResponse{Success: false, Error: "User is blocked"})
			return
		case domain.ErrInvalidPassword:
			c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Invalid credentials"})
			return
		default:
			c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Invalid credentials"})
			return
		}
	}

	resp := LoginResponse{
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
	}
	if res.Tokens != nil {
		resp.Tokens.AccessToken = res.Tokens.AccessToken
		resp.Tokens.RefreshToken = res.Tokens.RefreshToken
		resp.Tokens.TokenType = res.Tokens.TokenType
		resp.Tokens.ExpiresIn = res.Tokens.ExpiresIn
	}
	resp.Session.ID = res.Session.ID
	resp.Session.ExpiresAt = res.Session.ExpiresAt

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	const op = "handler.AuthHandler.Refresh"

	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid refresh request", slog.String("op", op), slog.Any("error", err))
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request format", Details: err.Error()})
		return
	}

	res, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case domain.ErrUserBlocked:
			c.JSON(http.StatusLocked, ErrorResponse{Success: false, Error: "User is blocked"})
			return
		default:
			c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Invalid refresh token"})
			return
		}
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		AccessToken:  res.Tokens.AccessToken,
		RefreshToken: res.Tokens.RefreshToken,
		TokenType:    res.Tokens.TokenType,
		ExpiresIn:    res.Tokens.ExpiresIn,
	})
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
