package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func perform(r *gin.Engine, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMiddlewareAndRequireRoles(t *testing.T) {
	userID := uuid.New()
	token := func(role string, typ tokenType) string {
		tok, err := generateToken(userID, role, typ, testSecret, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return "Bearer " + tok
	}

	r := gin.New()
	r.GET("/", Middleware(testSecret), RequireRoles("admin"), func(c *gin.Context) {
		id, ok := httpx.CurrentUserID(c)
		if !ok || id != userID {
			t.Errorf("CurrentUserID = %v, %v", id, ok)
		}
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"not bearer", "Basic abc", http.StatusUnauthorized},
		{"garbage token", "Bearer abc", http.StatusUnauthorized},
		{"refresh token", token("admin", tokenTypeRefresh), http.StatusUnauthorized},
		{"wrong role", token("user", tokenTypeAccess), http.StatusForbidden},
		{"admin access token", token("admin", tokenTypeAccess), http.StatusOK},
		{"lowercase scheme", "bearer " + strings.TrimPrefix(token("admin", tokenTypeAccess), "Bearer "), http.StatusOK},
		{"empty token", "Bearer ", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := perform(r, map[string]string{"Authorization": tt.header})
			if w.Code != tt.want {
				t.Fatalf("got %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
