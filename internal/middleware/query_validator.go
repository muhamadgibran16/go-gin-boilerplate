package middleware

import (
	"github.com/gibran/go-gin-boilerplate/pkg/response"
	"github.com/gin-gonic/gin"
)

// ValidateQueryParams checks if the provided query parameters are within the allowed whitelist
func ValidateQueryParams(allowedParams []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryParams := c.Request.URL.Query()
		
		for param := range queryParams {
			allowed := false
			for _, allowedParam := range allowedParams {
				if param == allowedParam {
					allowed = true
					break
				}
			}
			
			if !allowed {
				response.BadRequest(c, "unexpected query parameter: "+param)
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}
