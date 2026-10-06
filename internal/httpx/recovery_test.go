package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRecoveryReturnsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery(zap.NewNop()))
	r.GET("/", func(c *gin.Context) { panic("secret internal detail") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	var res ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("response is not JSON: %q", w.Body.String())
	}
	if w.Code != http.StatusInternalServerError || res.Status != "error" {
		t.Fatalf("got %d %+v, want 500 error", w.Code, res)
	}
	if strings.Contains(w.Body.String(), "secret") {
		t.Fatalf("panic value leaked: %s", w.Body.String())
	}
}
