package cli

import (
	"context"
	"fmt"
	"io"
)

type verboseContextKey struct{}

type verboseLogger struct {
	enabled bool
	stderr  io.Writer
}

func withVerbose(ctx context.Context, stderr io.Writer, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, verboseContextKey{}, verboseLogger{
		enabled: enabled,
		stderr:  stderr,
	})
}

func tracef(ctx context.Context, format string, args ...any) {
	if ctx == nil {
		return
	}

	logger, ok := ctx.Value(verboseContextKey{}).(verboseLogger)
	if !ok || !logger.enabled || logger.stderr == nil {
		return
	}

	_, _ = fmt.Fprintf(logger.stderr, "[trace] "+format+"\n", args...)
}
