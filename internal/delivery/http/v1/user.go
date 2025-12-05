package v1

import (
	"context"
	"log/slog"
	"net/http"
	"volunteer-coordination-server/internal/entity"
	"volunteer-coordination-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	admin     = "admin"
	volunteer = "volunteer"
	organizer = "organizer"
)

type UserHandler struct {
	usrService *service.UserService
}

func NewUserHandler(ctx context.Context, db *pgxpool.Pool) *UserHandler {
	return &UserHandler{
		usrService: service.NewUserService(db),
	}
}

func (h *UserHandler) NewAdminUserHandler(c *gin.Context) {
	var user entity.UserRegistration

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Role != admin {
		slog.Error("User role is not admin")
		c.JSON(http.StatusBadRequest, gin.H{"msg": "User role is not admin"})
		return
	}

	id, err := h.usrService.AddNewUser(c.Request.Context(), user)
	if err != nil {
		slog.Error("User failed to add new user", err.Error())
		c.JSON(http.StatusConflict, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"new user id": id})
}

func (h *UserHandler) NewVolunteerUserHandler(c *gin.Context) {
	var user entity.UserRegistration

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Role != volunteer {
		slog.Error("User role is not volunteer")
		c.JSON(http.StatusBadRequest, gin.H{"msg": "User role is not volunteer"})
		return
	}

	id, err := h.usrService.AddNewUser(c.Request.Context(), user)
	if err != nil {
		slog.Error("User failed to add new user", err.Error())
		c.JSON(http.StatusConflict, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"new user id": id})
}

func (h *UserHandler) NewOrganizerUserHandler(c *gin.Context) {
	var user entity.UserRegistration

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Role != organizer {
		slog.Error("User role is not organizer")
		c.JSON(http.StatusBadRequest, gin.H{"msg": "User role is not organizer"})
		return
	}

	id, err := h.usrService.AddNewUser(c.Request.Context(), user)
	if err != nil {
		slog.Error("User failed to add new user", err.Error())
		c.JSON(http.StatusConflict, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"new user id": id})
}

func (h *UserHandler) UserByUUIDHandler(c *gin.Context) {
	var user entity.User
	params := c.Params.ByName("id")
	user, err := h.usrService.UserByParam(c.Request.Context(), "id", params)
	if err != nil {
		slog.Error("User failed to find user by uuid", err.Error())
		c.JSON(http.StatusNotFound, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}
