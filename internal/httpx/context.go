package httpx

import (
	"github.com/gibran/go-gin-boilerplate/internal/pkg/ctxlog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Context keys for the authenticated user. They live here, not in the auth module,
// so any module can read the current user without importing auth.
const (
	ContextKeyUserID = "userID"
	ContextKeyRole   = "role"
)

// SetCurrentUser stores the authenticated user in the request context and adds
// user_id to the request logger
func SetCurrentUser(c *gin.Context, userID uuid.UUID, role string) {
	c.Set(ContextKeyUserID, userID)
	c.Set(ContextKeyRole, role)
	c.Request = c.Request.WithContext(ctxlog.AddFields(c.Request.Context(), zap.String("user_id", userID.String())))
}

// CurrentUserID returns the authenticated user's ID
func CurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := c.Get(ContextKeyUserID)
	if !ok {
		return uuid.Nil, false
	}
	userID, ok := id.(uuid.UUID)
	return userID, ok
}

// CurrentRole returns the authenticated user's role
func CurrentRole(c *gin.Context) (string, bool) {
	role, ok := c.Get(ContextKeyRole)
	if !ok {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
