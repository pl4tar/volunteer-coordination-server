package v1

import (
	"errors"
	"net/http"
	"strconv"
	"volunteer-coordination-server/internal/domain"

	"volunteer-coordination-server/internal/repository"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrganizerHandler struct {
	svc   *service.OrganizerService
	admin *service.AdminService
}

func NewOrganizerHandler(svc *service.OrganizerService, admin *service.AdminService) *OrganizerHandler {
	return &OrganizerHandler{svc: svc, admin: admin}
}

// getUserID достаёт user_id из контекста gin.
// В вашем middleware user_id кладётся строкой из jwt claims.
func getUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// ============================
//  Organizer: Events CRUD
// ============================

// CreateEvent godoc
// @Summary      Создать мероприятие (организатор)
// @Tags         organizer
// @Accept       json
// @Produce      json
// @Success      201 {object}  map[string]interface{} "success,id"
// @Failure      400 {object}  ErrorResponse
// @Failure      401 {object}  ErrorResponse
// @Security BearerAuth
// @Router       /organizer/events [post]
func (h *OrganizerHandler) CreateEvent(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	var req EventUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}

	id, err := h.svc.CreateEvent(c.Request.Context(), organizerID, domain.EventUpsert(req))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

// UpdateEvent godoc
// @Summary      Редактировать мероприятие (организатор)
// @Tags         organizer
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200 {object}  SuccessResponse
// @Failure      400 {object}  ErrorResponse
// @Failure      401 {object}  ErrorResponse
// @Failure      404 {object}  ErrorResponse
// @Security BearerAuth
// @Router       /organizer/events/{id} [put]
func (h *OrganizerHandler) UpdateEvent(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid event id"})
		return
	}

	var req EventUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}

	if err := h.svc.UpdateEvent(c.Request.Context(), organizerID, eventID, domain.EventUpsert(req)); err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) {
			code = http.StatusNotFound
		}
		c.JSON(code, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "updated"})
}

// DeleteEvent godoc
// @Summary      Удалить (отменить) мероприятие (организатор)
// @Tags         organizer
// @Produce      json
// @Param        id   path      string  true  "Event ID (UUID)"
// @Success      200 {object}  SuccessResponse
// @Failure      401 {object}  ErrorResponse
// @Failure      404 {object}  ErrorResponse
// @Security BearerAuth
// @Router       /organizer/events/{id} [delete]
func (h *OrganizerHandler) DeleteEvent(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid event id"})
		return
	}

	if err := h.svc.DeleteEvent(c.Request.Context(), organizerID, eventID); err != nil {
		code := http.StatusBadRequest
		if err == repository.ErrNotFound {
			code = http.StatusNotFound
		}
		c.JSON(code, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "deleted"})
}

// ListMyEvents godoc
// @Summary      Список моих мероприятий (организатор)
// @Tags         organizer
// @Produce      json
// @Param        status  query string false "Статус" Enums(active,cancelled,draft,pending_moderation,rejected,finished)
// @Param        page    query int    false "Страница" default(1)
// @Param        limit   query int    false "Лимит" default(20)
// @Success      200 {object} map[string]interface{} "success,items,total,page,limit"
// @Failure      401 {object} ErrorResponse
// @Security BearerAuth
// @Router       /organizer/events [get]
func (h *OrganizerHandler) ListMyEvents(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	q := repository.EventListQuery{
		Page:   page,
		Limit:  limit,
		Status: status,
	}

	items, total, err := h.svc.ListMyEvents(c.Request.Context(), organizerID, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "failed to list", Details: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   items,
		"total":   total,
		"page":    q.Page,
		"limit":   q.Limit,
	})
}

// ============================
//  Me: organizer request
// ============================

// RequestOrganizer godoc
// @Summary      Подать заявку на статус организатора
// @Tags         organizer
// @Accept       json
// @Produce      json
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Security BearerAuth
// @Router       /me/organizer/request [post]
func (h *OrganizerHandler) RequestOrganizer(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}

	var req organizerRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.admin.RequestOrganizer(c.Request.Context(), userID, req.CompanyName, req.Documents); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "request submitted"})
}
