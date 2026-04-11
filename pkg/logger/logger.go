// backend/pkg/logger/logger.go

package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// Interface -.
type Interface interface {
	Debug(message interface{}, args ...interface{})
	Info(message string, args ...interface{})
	InfoFields(message string, fields map[string]interface{})
	Warn(message string, args ...interface{})
	Error(message interface{}, args ...interface{})
	Fatal(message interface{}, args ...interface{})
}

// Logger -.
type Logger struct {
	logger *zerolog.Logger
}

var _ Interface = (*Logger)(nil)

// New -.
func New(level string, pretty bool) *Logger {
	var l zerolog.Level

	switch strings.ToLower(level) {
	case "error":
		l = zerolog.ErrorLevel
	case "warn":
		l = zerolog.WarnLevel
	case "info":
		l = zerolog.InfoLevel
	case "debug":
		l = zerolog.DebugLevel
	default:
		l = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(l)

	initSlogJSON(level)

	skipFrameCount := 3
	logger := zerolog.New(buildWriter(pretty)).With().Timestamp().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + skipFrameCount).Logger()

	return &Logger{
		logger: &logger,
	}
}

// initSlogJSON aligns stdlib slog with app log level and JSON lines on stdout (same sink as zerolog when not pretty).
func initSlogJSON(level string) {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel(level),
	})
	slog.SetDefault(slog.New(h))
}

func slogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return slog.LevelError
	case "warn", "warning":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func buildWriter(pretty bool) io.Writer {
	if !pretty {
		return os.Stdout
	}

	return &prettyJSONWriter{
		out: os.Stdout,
	}
}

type prettyJSONWriter struct {
	out io.Writer
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *prettyJSONWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.buf.Write(p); err != nil {
		return 0, err
	}

	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			// Incomplete line stays buffered for the next write.
			w.buf.Write(line)
			break
		}

		if err = writePrettyLine(w.out, bytes.TrimSuffix(line, []byte("\n"))); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

func writePrettyLine(dst io.Writer, line []byte) error {
	if len(bytes.TrimSpace(line)) == 0 {
		_, err := dst.Write([]byte("\n"))
		return err
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, line, "", "  "); err != nil {
		if _, writeErr := dst.Write(append(line, '\n')); writeErr != nil {
			return writeErr
		}
		return nil
	}

	pretty.WriteByte('\n')
	pretty.WriteByte('\n')
	_, err := dst.Write(pretty.Bytes())
	return err
}

// Debug -.
func (l *Logger) Debug(message interface{}, args ...interface{}) {
	l.msg(zerolog.DebugLevel, message, args...)
}

// Info -.
func (l *Logger) Info(message string, args ...interface{}) {
	l.log(zerolog.InfoLevel, message, args...)
}

// InfoFields logs a message with structured fields.
func (l *Logger) InfoFields(message string, fields map[string]interface{}) {
	event := l.logger.Info()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg(message)
}

// Warn -.
func (l *Logger) Warn(message string, args ...interface{}) {
	l.log(zerolog.WarnLevel, message, args...)
}

// Error -.
func (l *Logger) Error(message interface{}, args ...interface{}) {
	if l.logger.GetLevel() == zerolog.DebugLevel {
		l.Debug(message, args...)
	}

	l.msg(zerolog.ErrorLevel, message, args...)
}

// Fatal -.
func (l *Logger) Fatal(message interface{}, args ...interface{}) {
	l.msg(zerolog.FatalLevel, message, args...)

	os.Exit(1)
}

func (l *Logger) log(level zerolog.Level, message string, args ...interface{}) {
	if len(args) == 0 {
		l.logger.WithLevel(level).Msg(message)
	} else {
		l.logger.WithLevel(level).Msgf(message, args...)
	}
}

func (l *Logger) msg(level zerolog.Level, message interface{}, args ...interface{}) {
	switch msg := message.(type) {
	case error:
		// err.Error() must not be passed to Msgf: it can contain "%" and extra args are handler context, not printf operands.
		e := l.logger.WithLevel(level).Err(msg)
		if len(args) > 0 {
			e = e.Str("context", fmt.Sprint(args...))
		}
		e.Msg("error")
	case string:
		l.log(level, msg, args...)
	default:
		l.log(level, fmt.Sprintf("%s message %v has unknown type %v", level, message, msg), args...)
	}
}
