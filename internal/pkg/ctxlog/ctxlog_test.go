package ctxlog

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestFromCarriesFields(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	ctx := With(context.Background(), zap.New(core).With(zap.String("request_id", "abc")))
	ctx = AddFields(ctx, zap.String("user_id", "u1"))

	From(ctx).Info("hello")

	entry := logs.All()[0]
	fields := entry.ContextMap()
	if fields["request_id"] != "abc" || fields["user_id"] != "u1" {
		t.Fatalf("missing fields: %v", fields)
	}
}

func TestFromWithoutLoggerIsNoop(t *testing.T) {
	From(context.Background()).Info("must not panic")
}
