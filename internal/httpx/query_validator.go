package httpx

import (
	"net/url"
	"slices"

	"github.com/gin-gonic/gin"
)

// ValidateQueryParams rejects requests whose query string is malformed or contains
// parameters outside allowedParams.
//
// A malformed query string (e.g. containing ";") must be rejected explicitly: Go drops
// the affected parameters silently, so the request would otherwise run without them.
func ValidateQueryParams(allowedParams []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryParams, err := url.ParseQuery(c.Request.URL.RawQuery)
		if err != nil {
			BadRequest(c, "Invalid query string")
			c.Abort()
			return
		}

		for param := range queryParams {
			if !slices.Contains(allowedParams, param) {
				BadRequest(c, "unexpected query parameter: "+param)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
