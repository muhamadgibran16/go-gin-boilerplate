package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRateLimiterPerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	alice, bob := uuid.New(), uuid.New()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Simulates the auth middleware: the user comes from the X-Test-User header here
		if id, err := uuid.Parse(c.GetHeader("X-Test-User")); err == nil {
			SetCurrentUser(c, id, "user")
		}
	}, RateLimiterPerUser(2, time.Minute))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(user uuid.UUID) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234" // every request comes from the same IP
		if user != uuid.Nil {
			req.Header.Set("X-Test-User", user.String())
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	for i := 0; i < 2; i++ {
		if code := do(alice); code != http.StatusOK {
			t.Fatalf("alice request %d: got %d", i+1, code)
		}
	}
	if code := do(alice); code != http.StatusTooManyRequests {
		t.Fatalf("alice over the limit: got %d, want 429", code)
	}

	// Same IP, different user: own quota
	if code := do(bob); code != http.StatusOK {
		t.Fatalf("bob must not share alice's quota: got %d", code)
	}

	// Unauthenticated requests are limited by IP, separately from the users
	for i := 0; i < 2; i++ {
		if code := do(uuid.Nil); code != http.StatusOK {
			t.Fatalf("anonymous request %d: got %d", i+1, code)
		}
	}
	if code := do(uuid.Nil); code != http.StatusTooManyRequests {
		t.Fatalf("anonymous over the limit: got %d, want 429", code)
	}
}
