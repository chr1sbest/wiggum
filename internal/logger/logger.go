package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents log severity levels.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger is the interface for all logger implementations.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
}

// baseLogger provides common formatting logic.
type baseLogger struct {
	writer io.Writer
	level  Level
	fields []Field
	mu     sync.Mutex
}

func (b *baseLogger) log(level Level, msg string, fields ...Field) {
	if level < b.level {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	allFields := append(b.fields, fields...)

	fieldStr := ""
	for _, f := range allFields {
		fieldStr += fmt.Sprintf(" %s=%v", f.Key, f.Value)
	}

	fmt.Fprintf(b.writer, "[%s] %s: %s%s\n", timestamp, level.String(), msg, fieldStr)
}

// StdoutLogger logs to stdout.
type StdoutLogger struct {
	baseLogger
}

// NewStdoutLogger creates a logger that writes to stdout.
func NewStdoutLogger(level Level) *StdoutLogger {
	return &StdoutLogger{
		baseLogger: baseLogger{
			writer: os.Stdout,
			level:  level,
		},
	}
}

func (l *StdoutLogger) Debug(msg string, fields ...Field) { l.log(LevelDebug, msg, fields...) }
func (l *StdoutLogger) Info(msg string, fields ...Field)  { l.log(LevelInfo, msg, fields...) }
func (l *StdoutLogger) Warn(msg string, fields ...Field)  { l.log(LevelWarn, msg, fields...) }
func (l *StdoutLogger) Error(msg string, fields ...Field) { l.log(LevelError, msg, fields...) }

func (l *StdoutLogger) WithFields(fields ...Field) Logger {
	return &StdoutLogger{
		baseLogger: baseLogger{
			writer: l.writer,
			level:  l.level,
			fields: append(l.fields, fields...),
		},
	}
}

// FileLogger logs to a file.
type FileLogger struct {
	baseLogger
	file *os.File
}

// NewFileLogger creates a logger that writes to a file.
func NewFileLogger(path string, level Level) (*FileLogger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		baseLogger: baseLogger{
			writer: file,
			level:  level,
		},
		file: file,
	}, nil
}

func (l *FileLogger) Debug(msg string, fields ...Field) { l.log(LevelDebug, msg, fields...) }
func (l *FileLogger) Info(msg string, fields ...Field)  { l.log(LevelInfo, msg, fields...) }
func (l *FileLogger) Warn(msg string, fields ...Field)  { l.log(LevelWarn, msg, fields...) }
func (l *FileLogger) Error(msg string, fields ...Field) { l.log(LevelError, msg, fields...) }

func (l *FileLogger) WithFields(fields ...Field) Logger {
	return &FileLogger{
		baseLogger: baseLogger{
			writer: l.writer,
			level:  l.level,
			fields: append(l.fields, fields...),
		},
		file: l.file,
	}
}

// Close closes the log file.
func (l *FileLogger) Close() error {
	return l.file.Close()
}

// MultiLogger composes multiple loggers together.
type MultiLogger struct {
	loggers []Logger
	fields  []Field
}

// NewMultiLogger creates a logger that writes to multiple destinations.
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

func (m *MultiLogger) Debug(msg string, fields ...Field) {
	allFields := append(m.fields, fields...)
	for _, l := range m.loggers {
		l.Debug(msg, allFields...)
	}
}

func (m *MultiLogger) Info(msg string, fields ...Field) {
	allFields := append(m.fields, fields...)
	for _, l := range m.loggers {
		l.Info(msg, allFields...)
	}
}

func (m *MultiLogger) Warn(msg string, fields ...Field) {
	allFields := append(m.fields, fields...)
	for _, l := range m.loggers {
		l.Warn(msg, allFields...)
	}
}

func (m *MultiLogger) Error(msg string, fields ...Field) {
	allFields := append(m.fields, fields...)
	for _, l := range m.loggers {
		l.Error(msg, allFields...)
	}
}

func (m *MultiLogger) WithFields(fields ...Field) Logger {
	newLoggers := make([]Logger, len(m.loggers))
	copy(newLoggers, m.loggers)
	return &MultiLogger{
		loggers: newLoggers,
		fields:  append(m.fields, fields...),
	}
}
