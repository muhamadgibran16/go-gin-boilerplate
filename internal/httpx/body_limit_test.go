package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler(), BodyLimit(64))
	r.POST("/", func(c *gin.Context) {
		var body map[string]any
		if !BindJSON(c, &body) {
			return
		}
		c.Status(http.StatusOK)
	})

	small := `{"name":"ok"}`
	large := `{"name":"` + strings.Repeat("a", 100) + `"}`

	tests := []struct {
		name    string
		body    string
		chunked bool // no Content-Length, so the limit is hit while reading
		want    int
	}{
		{"small body", small, false, http.StatusOK},
		{"large body with content length", large, false, http.StatusRequestEntityTooLarge},
		{"large chunked body", large, true, http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.chunked {
				req.ContentLength = -1
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("got %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
