package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
	Fatal(msg string)
	Fatalf(format string, args ...interface{})
	With(key string, value interface{}) Logger
}

// zerologger - zerolog implementation
type zerologger struct {
	logger zerolog.Logger
}

// New creates a new logger
func New(level string) Logger {
	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	// Pretty console output
	zerolog.TimeFieldFormat = time.RFC3339
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	logger := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()
	return &zerologger{logger: logger}
}

// Debug logs debug message
func (l *zerologger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs formatted debug message
func (l *zerologger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs info message
func (l *zerologger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs formatted info message
func (l *zerologger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs warning message
func (l *zerologger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs formatted warning message
func (l *zerologger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs error message
func (l *zerologger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs formatted error message
func (l *zerologger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// Fatal logs fatal message and exits
func (l *zerologger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Fatalf logs formatted fatal message and exits
func (l *zerologger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// With adds context to logger
func (l *zerologger) With(key string, value interface{}) Logger {
	newLogger := l.logger.With().Interface(key, value).Logger()
	return &zerologger{logger: newLogger}
}
