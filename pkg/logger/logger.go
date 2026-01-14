// pkg/logger/logger.go
package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey struct{}

var base *slog.Logger

func Init(env string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	var h slog.Handler
	if env == "development" {
		opts.Level = slog.LevelDebug
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	base = slog.New(h)
}

func L() *slog.Logger {
	if base == nil {
		// fallback aman supaya tidak panic
		base = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	return base
}

func With(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func From(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok {
			return l
		}
	}
	return L()
}
