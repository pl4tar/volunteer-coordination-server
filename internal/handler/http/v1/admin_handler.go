package v1

import (
	"net/http"
	"strconv"

	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler { return &AdminHandler{svc: svc} }

type moderateEventRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note,omitempty"`
}

type createCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description,omitempty"`
}

type createTagRequest struct {
	Name string `json:"name" binding:"required"`
}

type setSettingsRequest struct {
	Value map[string]any `json:"value" binding:"required"`
}

type decideOrganizerVerificationRequest struct {
	Approve bool   `json:"approve"`
	Note    string `json:"note,omitempty"`
}

// BlockUser godoc
// @Summary      Заблокировать пользователя
// @Description  Блокирует учетную запись пользователя (is_blocked=true). Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Param        id   path      string  true  "User ID (UUID)"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/users/{id}/block [post]
func (h *AdminHandler) BlockUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid user id"})
		return
	}
	if err := h.svc.BlockUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "blocked"})
}

// UnblockUser godoc
// @Summary      Разблокировать пользователя
// @Description  Снимает блокировку учетной записи пользователя (is_blocked=false). Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Param        id   path      string  true  "User ID (UUID)"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/users/{id}/unblock [post]
func (h *AdminHandler) UnblockUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid user id"})
		return
	}
	if err := h.svc.UnblockUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "unblocked"})
}

//	func (h *AdminHandler) DeleteUser(c *gin.Context) {
//		userID, err := uuid.Parse(c.Param("id"))
//		if err != nil {
//			c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid user id"})
//			return
//		}
//		if err := h.svc.DeleteUser(c.Request.Context(), userID); err != nil {
//			c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
//			return
//		}
//		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "deleted"})
//	}
//
// ModerateEvent godoc
// @Summary      Модерировать мероприятие (изменить статус)
// @Description  Меняет статус мероприятия (например: active/rejected/pending_moderation) и пишет лог модерации. Доступно только администратору.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "Event ID (UUID)"
// @Param        body  body      moderateEventRequest true  "Moderation payload"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/events/{id}/moderate [post]
func (h *AdminHandler) ModerateEvent(c *gin.Context) {
	adminID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid event id"})
		return
	}
	var req moderateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}
	if err := h.svc.ModerateEvent(c.Request.Context(), adminID, eventID, req.Status, req.Note); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "moderated"})
}

// ListCategories godoc
// @Summary      Список категорий
// @Description  Возвращает список категорий мероприятий. Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  map[string]interface{} "success,items"
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/categories [get]
func (h *AdminHandler) ListCategories(c *gin.Context) {
	items, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "failed", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": items})
}

// CreateCategory godoc
// @Summary      Создать категорию
// @Description  Создает новую категорию (name, slug, description). Доступно только администратору.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      createCategoryRequest true  "Category payload"
// @Success      201  {object}  map[string]interface{} "success,id"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/categories [post]
func (h *AdminHandler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}
	id, err := h.svc.CreateCategory(c.Request.Context(), req.Name, req.Slug, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

// DeleteCategory godoc
// @Summary      Удалить категорию
// @Description  Удаляет категорию по ID. Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Param        id   path      int  true  "Category ID"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/categories/{id} [delete]
func (h *AdminHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid id"})
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "deleted"})
}

// ListTags godoc
// @Summary      Список тегов
// @Description  Возвращает список тегов мероприятий. Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  map[string]interface{} "success,items"
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/tags [get]
func (h *AdminHandler) ListTags(c *gin.Context) {
	items, err := h.svc.ListTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "failed", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": items})
}

// CreateTag godoc
// @Summary      Создать тег
// @Description  Создает новый тег. Доступно только администратору.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      createTagRequest true  "Tag payload"
// @Success      201  {object}  map[string]interface{} "success,id"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/tags [post]
func (h *AdminHandler) CreateTag(c *gin.Context) {
	var req createTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}
	id, err := h.svc.CreateTag(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

// DeleteTag godoc
// @Summary      Удалить тег
// @Description  Удаляет тег по ID. Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Param        id   path      int  true  "Tag ID"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/tags/{id} [delete]
func (h *AdminHandler) DeleteTag(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid id"})
		return
	}
	if err := h.svc.DeleteTag(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "deleted"})
}

// Stats godoc
// @Summary      Общая статистика системы
// @Description  Возвращает агрегированную статистику: users/events/active volunteers и т.д. Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  AdminStatsResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/stats [get]
func (h *AdminHandler) Stats(c *gin.Context) {
	data, err := h.svc.SystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "failed", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GetSettings godoc
// @Summary      Получить системную настройку
// @Description  Возвращает значение настройки по ключу (jsonb). Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Param        key  path      string  true  "Setting key"
// @Success      200  {object}  AdminSettingsResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/settings/{key} [get]
func (h *AdminHandler) GetSettings(c *gin.Context) {
	key := c.Param("key")
	data, err := h.svc.GetSettings(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "failed", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "key": key, "value": data})
}

// SetSettings godoc
// @Summary      Установить системную настройку
// @Description  Создает/обновляет настройку по ключу (jsonb). Доступно только администратору.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        key   path      string             true  "Setting key"
// @Param        body  body      setSettingsRequest true  "Settings payload"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/settings/{key} [put]
func (h *AdminHandler) SetSettings(c *gin.Context) {
	key := c.Param("key")
	var req setSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}
	if err := h.svc.SetSettings(c.Request.Context(), key, req.Value); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "saved"})
}

// ListOrganizerVerifications godoc
// @Summary      Список заявок на статус организатора
// @Description  Возвращает список заявок на организатора с фильтрацией по статусу и пагинацией. Доступно только администратору.
// @Tags         admin
// @Produce      json
// @Param        status  query  string  false  "Status filter" Enums(pending,approved,rejected)
// @Param        page    query  int     false  "Page" default(1)
// @Param        limit   query  int     false  "Limit" default(20)
// @Success      200  {object}  map[string]interface{} "success,items,total,page,limit"
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/organizer/verifications [get]
func (h *AdminHandler) ListOrganizerVerifications(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	items, total, err := h.svc.ListOrganizerVerifications(c.Request.Context(), status, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Success: false, Error: "failed", Details: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": items, "total": total, "page": page, "limit": limit})
}

// DecideOrganizerVerification godoc
// @Summary      Принять решение по заявке организатора
// @Description  Approve/reject заявки на статус организатора. При approve — выдаёт роль organizer. Доступно только администратору.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        userId  path      string                           true  "User ID (UUID)"
// @Param        body    body      decideOrganizerVerificationRequest true  "Decision payload"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security BearerAuth
// @Router       /admin/organizer/verifications/{userId}/decide [post]
func (h *AdminHandler) DecideOrganizerVerification(c *gin.Context) {
	adminID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Success: false, Error: "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "invalid user id"})
		return
	}

	var req decideOrganizerVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: "Invalid request", Details: err.Error()})
		return
	}

	if err := h.svc.DecideOrganizerVerification(c.Request.Context(), adminID, userID, req.Approve, req.Note); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "decided"})
}
