// Package ctxlog carries a request-scoped zap logger in a context.Context.
//
// The HTTP layer stores a logger that already has request_id (and user_id once
// authenticated), so any code that receives the request context can log with them:
//
//	ctxlog.From(ctx).Info("user registered", zap.String("email", email))
package ctxlog

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

// With returns a copy of ctx carrying logger
func With(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// From returns the logger stored in ctx, or a no-op logger if there is none
func From(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

// AddFields returns a copy of ctx whose logger has the given fields added
func AddFields(ctx context.Context, fields ...zap.Field) context.Context {
	return With(ctx, From(ctx).With(fields...))
}
