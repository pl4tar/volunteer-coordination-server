package v1

import (
	"net/http"
	"strconv"

	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VolunteerHandler struct {
	vol *service.VolunteerService
}

func NewVolunteerHandler(vol *service.VolunteerService) *VolunteerHandler {
	return &VolunteerHandler{vol: vol}
}

// Signup godoc
// @Summary      Записаться на мероприятие
// @Tags         events
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200  {object}  map[string]bool  "success"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /events/{id}/signup [post]
func (h *VolunteerHandler) Signup(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid event id"})
		return
	}

	if err := h.vol.Signup(c.Request.Context(), eventID, userID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Cancel godoc
// @Summary      Отменить заявку на участие
// @Tags         events
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200  {object}  map[string]bool  "success"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /events/{id}/cancel [post]
func (h *VolunteerHandler) Cancel(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid event id"})
		return
	}

	if err := h.vol.Cancel(c.Request.Context(), eventID, userID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Notifications godoc
// @Summary      Получить уведомления
// @Tags         me
// @Security     BearerAuth
// @Produce      json
// @Param        unread  query     bool  false  "Только непрочитанные" default(false)
// @Param        limit   query     int   false  "Лимит" default(50)
// @Success      200  {object}  map[string]interface{}  "items"
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /me/notifications [get]
func (h *VolunteerHandler) Notifications(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	onlyUnread := c.Query("unread") == "true"
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	items, err := h.vol.Notifications(c.Request.Context(), userID, onlyUnread, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to load notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// MarkNotificationRead godoc
// @Summary      Отметить уведомление прочитанным
// @Tags         me
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Notification ID (UUID)"
// @Success      200  {object}  map[string]bool  "success"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /me/notifications/{id}/read [post]
func (h *VolunteerHandler) MarkNotificationRead(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	nid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid notification id"})
		return
	}

	if err := h.vol.MarkNotificationRead(c.Request.Context(), userID, nid); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to update notification"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// History godoc
// @Summary      История участия
// @Tags         me
// @Security     BearerAuth
// @Produce      json
// @Param        page   query     int  false  "Страница" default(1)
// @Param        limit  query     int  false  "Лимит" default(20)
// @Success      200  {object}  map[string]interface{}  "items,total,page,limit"
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /me/history [get]
func (h *VolunteerHandler) History(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	page := 1
	limit := 20
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	items, total, err := h.vol.History(c.Request.Context(), userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to load history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

// Stats godoc
// @Summary      Статистика активности
// @Tags         me
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /me/stats [get]
func (h *VolunteerHandler) Stats(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	data, err := h.vol.Stats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to load stats"})
		return
	}
	c.JSON(http.StatusOK, data)
}

// Rating godoc
// @Summary      Рейтинг волонтёра
// @Tags         me
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /me/rating [get]
func (h *VolunteerHandler) Rating(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	data, err := h.vol.Rating(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to load rating"})
		return
	}
	c.JSON(http.StatusOK, data)
}

// LeaveReview godoc
// @Summary      Оставить отзыв о мероприятии
// @Tags         events
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string        true  "Event ID (UUID)"
// @Param        body  body      ReviewRequest true  "Оценка и комментарий"
// @Success      201  {object}   map[string]bool  "success"
// @Failure      400  {object}   ErrorResponse
// @Failure      401  {object}   ErrorResponse
// @Router       /events/{id}/reviews [post]
func (h *VolunteerHandler) LeaveReview(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid event id"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request"})
		return
	}

	if err := h.vol.LeaveReview(c.Request.Context(), eventID, userID, req.Rating, req.Comment); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true})
}
