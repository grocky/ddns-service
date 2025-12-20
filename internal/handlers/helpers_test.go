package handlers

import (
	"io"
	"log/slog"
)

// newTestLogger creates a logger that discards output for testing.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
