package middleware

import (
	"net/http"
	v1 "volunteer-coordination-server/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserMeta struct {
	UserID          uuid.UUID
	Role            string
	OrganizerStatus string
	IsBlocked       bool
}

const (
	CtxUserIDKey          = "user_id"
	CtxRoleKey            = "role"
	CtxOrganizerStatusKey = "organizer_status"
	CtxIsBlockedKey       = "is_blocked"
)

func RequireActiveUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		blockedAny, ok := c.Get(CtxIsBlockedKey)
		if ok {
			if blocked, _ := blockedAny.(bool); blocked {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "user is blocked"})
				return
			}
		}
		c.Next()
	}
}

func RequireRole(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, ok := c.Get("roles")
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, v1.ErrorResponse{Success: false, Error: "forbidden"})
			return
		}

		// ВАЖНО: у тебя roles кладутся как []string
		switch roles := rolesVal.(type) {

		case []string:
			for _, r := range roles {
				if r == required {
					c.Next()
					return
				}
			}

		case []interface{}: // на всякий случай, если где-то роли приходят иначе
			for _, it := range roles {
				if s, ok := it.(string); ok && s == required {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, v1.ErrorResponse{Success: false, Error: "forbidden"})
	}
}

func RequireApprovedOrganizer() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleAny, _ := c.Get(CtxRoleKey)
		role, _ := roleAny.(string)
		if role != "organizer" && role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "organizer required"})
			return
		}
		if role == "admin" {
			c.Next()
			return
		}
		stAny, _ := c.Get(CtxOrganizerStatusKey)
		st, _ := stAny.(string)
		if st != "approved" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "organizer not approved"})
			return
		}
		c.Next()
	}
}
