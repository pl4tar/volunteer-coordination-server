package http

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(ctx context.Context, db *pgxpool.Pool) *gin.Engine {
	r := gin.Default()
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
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	//r.Use(cors.New(corsConfig), middleware.MetricsMiddleware())

	//r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "pong"})
	})

	//teamHandler := handler.NewTeamHandler(ctx, db)
	//userHandler := handler.NewUserHandler(ctx, db)
	//PRHandler := handler.NewPRHandler(ctx, db)

	//userHandler := v1.NewUserHandler(ctx, db)

	//api := r.Group("/api/v1")
	//{
	//users := api.Group("/users")
	//{
	// Получение пользователя по ID
	//users.GET("/:id", userHandler.UserByUUIDHandler)

	// Создание пользователя (универсальный метод)
	//users.POST("/", userHandler.CreateUserHandler)

	// Для обратной совместимости можно оставить старые маршруты
	//register := users.Group("/register")
	//{
	// Перенаправляем на универсальный метод
	//register.POST("/admins", userHandler.CreateUserHandler)
	//register.POST("/volunteers", userHandler.CreateUserHandler)
	//register.POST("/organizers", userHandler.CreateUserHandler)
	//}
	//}
	//}

	return r
}
