package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// stubService returns err from every method
type stubService struct{ err error }

func (s stubService) List(context.Context, ListQuery) ([]User, int64, error) {
	return nil, 0, s.err
}

func (s stubService) Get(context.Context, uuid.UUID) (*User, error) {
	return nil, s.err
}

func (s stubService) Update(context.Context, uuid.UUID, uuid.UUID, UpdateInput) (*User, error) {
	return &User{}, s.err
}

func (s stubService) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return s.err
}

// newRouter mounts the handler like the app does: behind ErrorHandler, as an authenticated admin
func newRouter(svc service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(svc)
	r := gin.New()
	r.Use(httpx.ErrorHandler(), func(c *gin.Context) {
		httpx.SetCurrentUser(c, uuid.New(), RoleAdmin)
	})
	r.GET("/users", h.GetMany)
	r.GET("/users/:id", h.GetOne)
	r.PUT("/users/:id", h.Update)
	r.DELETE("/users/:id", h.Delete)
	return r
}

func do(r *gin.Engine, method, path, body string) (int, httpx.ErrorResponse) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var res httpx.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	return w.Code, res
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", ErrNotFound, http.StatusNotFound},
		{"self deletion", ErrSelfModification, http.StatusForbidden},
		{"internal error is hidden", errors.New("pq: connection refused"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, res := do(newRouter(stubService{err: tt.err}), http.MethodDelete, "/users/"+uuid.NewString(), "")
			if code != tt.want {
				t.Fatalf("got %d, want %d", code, tt.want)
			}
			if strings.Contains(res.Message, "pq:") {
				t.Fatalf("internal error leaked: %q", res.Message)
			}
		})
	}
}

func TestValidationMessages(t *testing.T) {
	r := newRouter(stubService{})

	tests := []struct {
		name, method, path, body string
		wantField, wantMessage   string
	}{
		{"invalid path id", http.MethodGet, "/users/not-a-uuid", "", "id", "id must be a valid UUID"},
		{"number limit uses query name", http.MethodGet, "/users?perPage=101", "", "perPage", "perPage must be at most 100"},
		{"string length uses json name", http.MethodPut, "/users/" + uuid.NewString(), `{"name":"` + strings.Repeat("a", 256) + `"}`, "name", "name must be at most 255 characters long"},
		{"oneof", http.MethodPut, "/users/" + uuid.NewString(), `{"role":"root"}`, "role", "role must be one of: admin, user"},
		{"wrong json type", http.MethodPut, "/users/" + uuid.NewString(), `{"name":123}`, "name", "name must be of type string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, res := do(r, tt.method, tt.path, tt.body)
			if code != http.StatusBadRequest {
				t.Fatalf("got %d, want 400", code)
			}
			fields, _ := res.Errors.(map[string]any)
			if fields[tt.wantField] != tt.wantMessage {
				t.Fatalf("errors = %v, want %s: %q", res.Errors, tt.wantField, tt.wantMessage)
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	code, res := do(newRouter(stubService{}), http.MethodPut, "/users/"+uuid.NewString(), `{"name":`)
	if code != http.StatusBadRequest || res.Message != "Invalid JSON body" {
		t.Fatalf("got %d %q, want 400 \"Invalid JSON body\"", code, res.Message)
	}
}

func TestPaginationDefaults(t *testing.T) {
	r := newRouter(stubService{})
	for _, q := range []string{"", "?page=", "?perPage=", "?page=0&perPage=0"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users"+q, nil))

		var res struct{ Meta map[string]int }
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if w.Code != http.StatusOK || res.Meta["currentPage"] != 1 || res.Meta["perPage"] != 10 {
			t.Errorf("/users%s -> %d %v, want 200 with page 1 and perPage 10", q, w.Code, res.Meta)
		}
	}
}
