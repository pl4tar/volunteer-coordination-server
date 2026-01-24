package middleware

import (
	"net/http"

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

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		roleAny, ok := c.Get(CtxRoleKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
		role, _ := roleAny.(string)
		if _, ok := allowed[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
		c.Next()
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
