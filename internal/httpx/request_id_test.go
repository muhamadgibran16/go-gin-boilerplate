package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	perform := func(id string) string {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", id)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Header().Get("X-Request-ID")
	}

	if got := perform("abc-123"); got != "abc-123" {
		t.Fatalf("valid client ID must be kept, got %q", got)
	}

	for _, bad := range []string{"", "has space", "line\r\nbreak", strings.Repeat("a", 65)} {
		got := perform(bad)
		if _, err := uuid.Parse(got); err != nil {
			t.Fatalf("invalid client ID %q must be replaced with a UUID, got %q", bad, got)
		}
	}
}
