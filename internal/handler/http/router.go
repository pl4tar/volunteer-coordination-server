package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"volunteer-coordination-server/internal/middleware"

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

	eventRepo := repository.NewEventRepository(db)
	regRepo := repository.NewRegistrationRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	volunteerService := service.NewVolunteerService(eventRepo, regRepo, notifRepo, reviewRepo)

	eventHandler := v1.NewEventHandler(volunteerService)
	volunteerHandler := v1.NewVolunteerHandler(volunteerService)

	modRepo := repository.NewModerationRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	verifyRepo := repository.NewOrganizerVerificationRepository(db)

	adminService := service.NewAdminService(userRepo, modRepo, catRepo, tagRepo, statsRepo, setRepo, verifyRepo)
	adminHandler := v1.NewAdminHandler(adminService)

	orgEventRepo := repository.NewOrganizerEventRepository(db)
	organizerService := service.NewOrganizerService(orgEventRepo)

	organizerHandler := v1.NewOrganizerHandler(organizerService, adminService)

	api := r.Group("/api/v1")
	{
		organizer := api.Group("/organizer")
		organizer.Use(authMiddleware(tokenService))
		{
			organizer.GET("/events", organizerHandler.ListMyEvents)
			organizer.POST("/events", organizerHandler.CreateEvent)
			organizer.PUT("/events/:id", organizerHandler.UpdateEvent)
			organizer.DELETE("/events/:id", organizerHandler.DeleteEvent)

			//organizer.GET("/events/:id/registrations", organizerHandler.ListRegistrations)
			//organizer.POST("/events/:id/registrations/:userId", organizerHandler.DecideRegistration)
			//
			//organizer.POST("/events/:id/notify", organizerHandler.Notify)
			//organizer.POST("/events/:id/participants/:userId/score", organizerHandler.UpsertScore)
			//
			//organizer.GET("/stats", organizerHandler.Stats)
		}

		admin := api.Group("/admin")
		admin.Use(authMiddleware(tokenService), middleware.RequireRole("admin"))
		{
			admin.POST("/users/:id/block", adminHandler.BlockUser)
			admin.POST("/users/:id/unblock", adminHandler.UnblockUser)
			//admin.DELETE("/users/:id", adminHandler.DeleteUser)

			admin.POST("/events/:id/moderate", adminHandler.ModerateEvent)

			admin.GET("/categories", adminHandler.ListCategories)
			admin.POST("/categories", adminHandler.CreateCategory)
			admin.DELETE("/categories/:id", adminHandler.DeleteCategory)

			admin.GET("/tags", adminHandler.ListTags)
			admin.POST("/tags", adminHandler.CreateTag)
			admin.DELETE("/tags/:id", adminHandler.DeleteTag)

			admin.GET("/stats", adminHandler.Stats)

			admin.GET("/settings/:key", adminHandler.GetSettings)
			admin.PUT("/settings/:key", adminHandler.SetSettings)

			admin.GET("/organizer/verifications", adminHandler.ListOrganizerVerifications)
			admin.POST("/organizer/verifications/:userId/decide", adminHandler.DecideOrganizerVerification)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		users := api.Group("/users")
		users.Use(authMiddleware(tokenService))
		{
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateUser)

			users.PUT("/me/password", userHandler.ChangePassword)

			users.PUT("/password", userHandler.ChangePassword)

			users.GET("", userHandler.GetUsers)

			users.GET("/:id", userHandler.GetUserByID)
		}
		events := api.Group("/events")
		{
			events.GET("", eventHandler.ListEvents)
			events.GET("/:id", eventHandler.GetEvent)

			events.Use(authMiddleware(tokenService))
			{
				events.POST("/:id/signup", volunteerHandler.Signup)
				events.POST("/:id/cancel", volunteerHandler.Cancel)
				events.POST("/:id/reviews", volunteerHandler.LeaveReview)
			}
		}

		me := api.Group("/me")
		me.Use(authMiddleware(tokenService))
		{
			me.GET("/notifications", volunteerHandler.Notifications)
			me.POST("/notifications/:id/read", volunteerHandler.MarkNotificationRead)

			me.GET("/history", volunteerHandler.History)
			me.GET("/stats", volunteerHandler.Stats)
			me.GET("/rating", volunteerHandler.Rating)
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
