package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTimeoutAnswers503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler(), Timeout(20*time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		// Simulates a query that waits on the request context, like GORM with WithContext
		<-c.Request.Context().Done()
		Abort(c, c.Request.Context().Err())
	})

	start := time.Now()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503 (%s)", w.Code, w.Body.String())
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("took %v, the timeout did not cancel the request", elapsed)
	}
}
