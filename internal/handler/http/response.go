package router

import (
	"github.com/gin-gonic/gin"
)

// Response структура для стандартизированных ответов
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SuccessResponse возвращает успешный ответ
func SuccessResponse(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse возвращает ответ с ошибкой
func ErrorResponse(c *gin.Context, status int, message string, err error) {
	response := Response{
		Success: false,
		Error:   message,
	}

	if err != nil {
		// В development режиме можно показывать детали ошибки
		if gin.Mode() == gin.DebugMode {
			response.Error = err.Error()
		}
	}

	c.JSON(status, response)
}

// ValidationError возвращает ошибку валидации
func ValidationError(c *gin.Context, err error) {
	ErrorResponse(c, 400, "Validation error", err)
}

// NotFoundError возвращает ошибку "не найдено"
func NotFoundError(c *gin.Context, resource string) {
	ErrorResponse(c, 404, resource+" not found", nil)
}

// UnauthorizedError возвращает ошибку авторизации
func UnauthorizedError(c *gin.Context) {
	ErrorResponse(c, 401, "Unauthorized", nil)
}

// ForbiddenError возвращает ошибку доступа
func ForbiddenError(c *gin.Context) {
	ErrorResponse(c, 403, "Forbidden", nil)
}

// InternalServerError возвращает внутреннюю ошибку сервера
func InternalServerError(c *gin.Context, err error) {
	ErrorResponse(c, 500, "Internal server error", err)
}
