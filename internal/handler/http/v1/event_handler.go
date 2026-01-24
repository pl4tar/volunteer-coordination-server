package v1

import (
	"net/http"
	"strconv"
	"time"

	"volunteer-coordination-server/internal/repository"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventHandler struct {
	vol *service.VolunteerService
}

func NewEventHandler(vol *service.VolunteerService) *EventHandler {
	return &EventHandler{vol: vol}
}

// ListEvents godoc
// @Summary      Список мероприятий
// @Description  Возвращает список доступных мероприятий с фильтрами
// @Tags         events
// @Accept       json
// @Produce      json
// @Param        from         query     string  false  "Дата/время начала (RFC3339), starts_at >= from"
// @Param        to           query     string  false  "Дата/время конца (RFC3339), starts_at <= to"
// @Param        city         query     string  false  "Город/место (поиск по address ILIKE)"
// @Param        category_id  query     int     false  "ID категории"
// @Param        status       query     string  false  "Статус мероприятия" Enums(active,cancelled,draft)
// @Param        q            query     string  false  "Поиск по title/description"
// @Param        page         query     int     false  "Страница" default(1)
// @Param        limit        query     int     false  "Лимит" default(20)
// @Success      200  {object}  map[string]interface{}  "items,total,page,limit"
// @Failure      500  {object}  ErrorResponse
// @Router       /events [get]
func (h *EventHandler) ListEvents(c *gin.Context) {
	var q repository.EventListQuery

	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.To = &t
		}
	}
	if v := c.Query("city"); v != "" {
		q.City = v
	}
	if v := c.Query("q"); v != "" {
		q.Q = v
	}
	if v := c.Query("status"); v != "" {
		q.Status = v
	}
	if v := c.Query("category_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Category = &n
		}
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}

	items, total, err := h.vol.ListEvents(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "Failed to list events"})
		return
	}

	// чтобы не плодить отдельный DTO на каждое поле — отдаём как есть
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, it)
	}

	c.JSON(http.StatusOK, gin.H{
		"items": out,
		"total": total,
		"page":  q.Page,
		"limit": q.Limit,
	})
}

// GetEvent godoc
// @Summary      Детальная информация о мероприятии
// @Tags         events
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200  {object}  repository.Event
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /events/{id} [get]
func (h *EventHandler) GetEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid event id"})
		return
	}

	e, err := h.vol.GetEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Success: false, Error: "Event not found"})
		return
	}

	c.JSON(http.StatusOK, e)
}
