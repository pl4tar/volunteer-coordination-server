package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"volunteer-coordination-server/config"
	v1 "volunteer-coordination-server/internal/handler/http/v1"
	"volunteer-coordination-server/internal/repository"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupRouter собирает зависимости (repos/services/handlers) и регистрирует HTTP роуты.
func SetupRouter(ctx context.Context, cfg *config.Config, db *pgxpool.Pool) *gin.Engine {
	r := gin.New()

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	//corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "pong"})
	})

	// --- repositories ---
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// --- services ---
	userService := service.NewUserService(userRepo)
	tokenService := service.NewTokenService(cfg)
	authService := service.NewAuthService(userRepo, sessionRepo, tokenService, cfg)

	// --- handlers ---
	userHandler := v1.NewUserHandler(userService)
	authHandler := v1.NewAuthHandler(authService)

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
		}

		users := api.Group("/users")
		users.Use(authMiddleware(tokenService))
		{
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateUser)
			users.PUT("/me/password", userHandler.ChangePassword)

			users.GET("", userHandler.GetUsers)        // admin only
			users.GET("/:id", userHandler.GetUserByID) // self/admin
		}
	}

	_ = ctx
	return r
}

// authMiddleware валидирует Bearer access token и кладёт user_id/roles/email в gin.Context.
func authMiddleware(tokenSvc *service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, v1.ErrorResponse{Success: false, Error: "Unauthorized"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, v1.ErrorResponse{Success: false, Error: "Unauthorized"})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims, err := tokenSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, v1.ErrorResponse{Success: false, Error: "Unauthorized"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}
