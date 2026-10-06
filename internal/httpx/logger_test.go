package httpx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestPrettyAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, color := range []bool{false, true} {
		var out bytes.Buffer
		r := gin.New()
		r.Use(RequestID(), Logger(zap.NewNop(), LoggerOptions{Pretty: true, Color: color, Out: &out}), ErrorHandler())
		r.GET("/users", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/users?page=2", nil)
		req.Header.Set("X-Request-ID", "trace-1")
		r.ServeHTTP(httptest.NewRecorder(), req)

		line := out.String()
		for _, want := range []string{" 200 ", "GET", "/users?page=2", "req=trace-1"} {
			if !strings.Contains(line, want) {
				t.Errorf("color=%v: line %q does not contain %q", color, line, want)
			}
		}
		if hasEscape := strings.Contains(line, "\033["); hasEscape != color {
			t.Errorf("color=%v: escape codes present = %v in %q", color, hasEscape, line)
		}
	}
}

func TestPrettyAccessLogShowsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var out bytes.Buffer
	r := gin.New()
	r.Use(Logger(zap.NewNop(), LoggerOptions{Pretty: true, Out: &out}), ErrorHandler())
	r.GET("/", func(c *gin.Context) { ParamUUID(c, "id") })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if line := out.String(); !strings.Contains(line, " 400 ") || !strings.Contains(line, "error=Validation failed") {
		t.Fatalf("line %q must show the status and error", line)
	}
}
