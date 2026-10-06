package auth

import (
	"slices"
	"strings"

	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gin-gonic/gin"
)

// Middleware validates the Bearer access token and stores the user in the request context.
// Read it with httpx.CurrentUserID / httpx.CurrentRole.
func Middleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpx.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		// The scheme is case-insensitive (RFC 9110), so "bearer" is accepted too
		scheme, token, ok := strings.Cut(authHeader, " ")
		token = strings.TrimSpace(token)
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			httpx.Unauthorized(c, "Authorization header must be Bearer token")
			c.Abort()
			return
		}

		claims, err := validateToken(token, secret, tokenTypeAccess)
		if err != nil {
			httpx.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		httpx.SetCurrentUser(c, claims.UserID, claims.Role)

		c.Next()
	}
}

// RequireRoles allows the request only if the authenticated user has one of the given roles.
// It must run after Middleware.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := httpx.CurrentRole(c)
		if !ok {
			httpx.Unauthorized(c, "Unauthenticated")
			c.Abort()
			return
		}

		if !slices.Contains(allowedRoles, role) {
			httpx.Forbidden(c, "You don't have permission to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}
