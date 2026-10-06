package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", ValidateQueryParams([]string{"page", "sort"}), func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		query string
		want  int
	}{
		{"", http.StatusOK},
		{"?page=1&sort=name", http.StatusOK},
		{"?foo=1", http.StatusBadRequest},
		// Go silently drops parameters containing ";", so it must be rejected, not ignored
		{"?sort=name;DROP%20TABLE%20users", http.StatusBadRequest},
		{"?page=%zz", http.StatusBadRequest},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/"+tt.query, nil))
		if w.Code != tt.want {
			t.Errorf("%q: got %d, want %d", tt.query, w.Code, tt.want)
		}
	}
}
