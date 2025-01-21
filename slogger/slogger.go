package slogger

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

// PrettyHandlerOptions contains options specific to the PrettyHandler, mainly around slog handling.
type PrettyHandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

// PrettyHandler implements slog.Handler and provides a structured, colored logging output.
type PrettyHandler struct {
	slog.Handler
	l *log.Logger
}

// Log is a global logger instance used across the application.
var Log *slog.Logger

// Handle processes a single log record, formats it, and outputs it to the configured io.Writer.
func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// Change color based on log level
	level := r.Level.String()

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level + " ")
	case slog.LevelWarn:
		level = color.YellowString(level + " ")
	case slog.LevelError:
		level = color.RedString(level)
	}

	// Collect log attributes
	fields := make(map[string]interface{}, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "err" && a.Value.Any() != nil {
			err, ok := a.Value.Any().(error)
			if ok {
				fields[a.Key] = err.Error()
			} else {
				fields[a.Key] = a.Value.Any()
			}
		} else {
			fields[a.Key] = a.Value.Any()
		}
		return true
	})

	// Capture the source from runtime call stack
	source := make(map[string]interface{}, r.NumAttrs())

	fs := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := fs.Next()
	source["file"] = filepath.Base(frame.File)
	source["line"] = frame.Line
	source["func"] = color.CyanString(filepath.Base(frame.Function))

	// Format the timestamp
	timeStr := color.GreenString(r.Time.Format(time.DateTime))
	msg := r.Message

	// Check for a trace ID in the context and add it to the log fields if present
	traceID, ok := ctx.Value("trace-id").(uuid.UUID)
	if ok {
		fields["trace-id"] = traceID
	}
	b, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return err
	}

	// Print the formatted log entry
	h.l.Printf("%v | %v | %v | %v | %v:%v %v", timeStr, level, msg, source["func"], source["file"], source["line"], string(b))

	return nil
}

// NewPrettyHandler creates a new PrettyHandler with a given output writer and options.
func NewPrettyHandler(
	out io.Writer,
	opts PrettyHandlerOptions,
) *PrettyHandler {
	h := &PrettyHandler{
		Handler: slog.NewJSONHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}

	return h
}

// MakeLogger initializes and configures the global logger instance.
func MakeLogger() {
	opts := PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		},
	}

	handler := NewPrettyHandler(os.Stdout, opts)
	Log = slog.New(handler)
}

// truncatePath truncates the file path to show only the last 4 components.
func truncatePath(fullPath string) string {
	parts := strings.Split(fullPath, string(filepath.Separator))
	if len(parts) <= 4 {
		return fullPath
	}
	return filepath.Join(parts[len(parts)-4:]...)
}
