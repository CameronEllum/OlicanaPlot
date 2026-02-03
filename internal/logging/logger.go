// Package logging provides structured logging for OlicanaPlot.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	globalWriter io.Writer = os.Stderr
	writerMu     sync.RWMutex
	logLevel     = slog.LevelDebug
)

// SafeMultiWriter is a writer that writes to multiple writers but doesn't
// stop if one of them fails (useful for Windows GUI apps where Stdout might be closed).
type SafeMultiWriter struct {
	Writers []io.Writer
}

func (s *SafeMultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range s.Writers {
		if w == nil {
			continue
		}
		_, _ = w.Write(p) // Ignore errors from individual writers (like closed stdout in GUI)
	}
	return len(p), nil
}

// SetOutput updates the global output for all loggers.
func SetOutput(w ...io.Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()

	if len(w) == 1 {
		globalWriter = w[0]
	} else {
		globalWriter = &SafeMultiWriter{Writers: w}
	}
}

// Logger is the interface for structured logging.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// slogLogger wraps slog.Logger to implement our Logger interface.
type slogLogger struct {
	name string
}

// NewLogger creates a new structured logger with the given name.
func NewLogger(name string) Logger {
	return &slogLogger{name: name}
}

func (l *slogLogger) getLogger() *slog.Logger {
	writerMu.RLock()
	defer writerMu.RUnlock()
	handler := slog.NewTextHandler(globalWriter, &slog.HandlerOptions{
		Level: logLevel,
	})
	return slog.New(handler).With("component", l.name)
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.getLogger().Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.getLogger().Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.getLogger().Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.getLogger().Error(msg, args...)
}

// Redirector implements io.Writer to redirect standard log calls to slog.
type Redirector struct {
	logger Logger
}

func (r *Redirector) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		r.logger.Info(msg)
	}
	return len(p), nil
}

// NewRedirector creates a new writer that redirects output to the given logger.
func NewRedirector(l Logger) io.Writer {
	return &Redirector{logger: l}
}
