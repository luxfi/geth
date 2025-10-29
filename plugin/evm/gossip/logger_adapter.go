// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gossip

import (
	"github.com/luxfi/log"
	"github.com/luxfi/node/utils/logging"
	"go.uber.org/zap"
)

// loggerAdapter adapts luxfi/log.Logger to node/utils/logging.Logger
type loggerAdapter struct {
	logger log.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger log.Logger) logging.Logger {
	return &loggerAdapter{logger: logger}
}

func (l *loggerAdapter) Write(p []byte) (n int, err error) {
	l.logger.Info(string(p))
	return len(p), nil
}

func (l *loggerAdapter) Stop() {
	// No-op for this adapter
}

func (l *loggerAdapter) StopOnPanic() {
	// No-op for this adapter
}

func (l *loggerAdapter) RecoverAndPanic(f func()) {
	defer func() {
		if r := recover(); r != nil {
			l.logger.Error("panic recovered", "error", r)
			panic(r)
		}
	}()
	f()
}

func (l *loggerAdapter) RecoverAndExit(f func(), exit func()) {
	defer func() {
		if r := recover(); r != nil {
			l.logger.Error("panic recovered, exiting", "error", r)
			exit()
		}
	}()
	f()
}

func (l *loggerAdapter) GetLevel() logging.Level {
	return logging.Info
}

func (l *loggerAdapter) SetLevel(level logging.Level) {
	// No-op for this adapter
}

func (l *loggerAdapter) GetDisplayLevel() logging.Level {
	return logging.Info
}

func (l *loggerAdapter) SetDisplayLevel(level logging.Level) {
	// No-op for this adapter
}

func (l *loggerAdapter) GetLogLevel() logging.Level {
	return logging.Info
}

func (l *loggerAdapter) SetLogLevel(level logging.Level) {
	// No-op for this adapter
}

func (l *loggerAdapter) Log(level logging.Level, msg string, fields ...zap.Field) {
	// Extract key-value pairs from zap.Field
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Interface)
	}

	switch level {
	case logging.Fatal:
		l.logger.Error(msg, args...)
	case logging.Error:
		l.logger.Error(msg, args...)
	case logging.Warn:
		l.logger.Warn(msg, args...)
	case logging.Info:
		l.logger.Info(msg, args...)
	case logging.Debug:
		l.logger.Debug(msg, args...)
	case logging.Trace:
		l.logger.Debug(msg, args...)
	default:
		l.logger.Info(msg, args...)
	}
}

func (l *loggerAdapter) Fatal(msg string, fields ...zap.Field) {
	l.Log(logging.Fatal, msg, fields...)
}

func (l *loggerAdapter) Error(msg string, fields ...zap.Field) {
	l.Log(logging.Error, msg, fields...)
}

func (l *loggerAdapter) Warn(msg string, fields ...zap.Field) {
	l.Log(logging.Warn, msg, fields...)
}

func (l *loggerAdapter) Info(msg string, fields ...zap.Field) {
	l.Log(logging.Info, msg, fields...)
}

func (l *loggerAdapter) Debug(msg string, fields ...zap.Field) {
	l.Log(logging.Debug, msg, fields...)
}

func (l *loggerAdapter) Trace(msg string, fields ...zap.Field) {
	l.Log(logging.Trace, msg, fields...)
}

func (l *loggerAdapter) Verbo(msg string, fields ...zap.Field) {
	l.Log(logging.Verbo, msg, fields...)
}

func (l *loggerAdapter) Enabled(level logging.Level) bool {
	// For now, always return true - all log levels are enabled
	return true
}

func (l *loggerAdapter) With(fields ...zap.Field) logging.Logger {
	// For now, return the same logger - field accumulation not supported
	return l
}

func (l *loggerAdapter) WithOptions(opts ...zap.Option) logging.Logger {
	// For now, return the same logger - options not supported
	return l
}
