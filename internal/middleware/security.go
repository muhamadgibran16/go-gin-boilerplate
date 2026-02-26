package middleware

import "github.com/gin-gonic/gin"

// Security handles security-related HTTP headers
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent browsers from performing MIME-type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS filtering in browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}
