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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(_ context.Context, cfg *config.Config, db *pgxpool.Pool) *gin.Engine {
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

	// CORS для cookie-auth
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "pong"}) })

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	userService := service.NewUserService(userRepo)
	tokenService := service.NewTokenService(cfg)
	authService := service.NewAuthService(userRepo, sessionRepo, tokenService, cfg)

	authHandler := v1.NewAuthHandler(authService)
	userHandler := v1.NewUserHandler(userService)

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
		}

		users := api.Group("/users")
		users.Use(authMiddleware(tokenService))
		{
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateUser)

			users.PUT("/me/password", userHandler.ChangePassword)

			// Алиас, чтобы /users/password тоже работал (и не попадал в /:id)
			users.PUT("/password", userHandler.ChangePassword)

			users.GET("", userHandler.GetUsers)

			users.GET("/:id", userHandler.GetUserByID)
		}
	}

	return r
}

func authMiddleware(tokenSvc *service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		if v, err := c.Cookie("access_token"); err == nil && v != "" {
			tokenStr = v
		}

		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenStr = strings.TrimSpace(parts[1])
				}
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, v1.ErrorResponse{Success: false, Error: "Unauthorized"})
			return
		}

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
