package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestFrom(t *testing.T) {
	notFound := NotFound("user not found")

	if got := From(fmt.Errorf("context: %w", notFound)); got.Kind != KindNotFound || got.Message != "user not found" {
		t.Fatalf("wrapped app error must be unwrapped, got %+v", got)
	}

	cause := errors.New("pq: connection refused")
	got := From(cause)
	if got.Kind != KindInternal || !errors.Is(got, cause) {
		t.Fatalf("unknown error must become internal and keep its cause, got %+v", got)
	}
}

func TestIsMatchesSentinelAfterWrap(t *testing.T) {
	sentinel := Conflict("email already registered")
	wrapped := sentinel.Wrap(errors.New("duplicate key"))

	if !errors.Is(wrapped, sentinel) {
		t.Fatal("wrapped error must match its sentinel")
	}
	if errors.Is(wrapped, Conflict("something else")) {
		t.Fatal("different message must not match")
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := map[Kind]int{
		KindBadRequest:      http.StatusBadRequest,
		KindValidation:      http.StatusBadRequest,
		KindUnauthorized:    http.StatusUnauthorized,
		KindForbidden:       http.StatusForbidden,
		KindNotFound:        http.StatusNotFound,
		KindConflict:        http.StatusConflict,
		KindPayloadTooLarge: http.StatusRequestEntityTooLarge,
		KindInternal:        http.StatusInternalServerError,
	}
	for kind, want := range tests {
		if got := kind.HTTPStatus(); got != want {
			t.Errorf("Kind(%d).HTTPStatus() = %d, want %d", kind, got, want)
		}
	}
}
