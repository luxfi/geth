// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package log re-exports github.com/luxfi/log for backward compatibility.
// All geth-style logging functions (Crit, Fatal, Error, Warn, Info, Debug, Trace)
// are provided by luxfi/log which writes to stderr by default.
//
// Unlike the upstream go-ethereum log package which uses slog with a DiscardHandler
// (silently swallowing log.Crit messages before calling os.Exit), this package
// ensures all log output is visible through the unified luxfi/log infrastructure.
package log

import (
	"context"
	"io"
	"log/slog"
	"os"

	luxlog "github.com/luxfi/log"
)

// Logger is the geth-compatible logger interface.
type Logger = luxlog.Logger

// Level re-exports from luxfi/log.
type Level = luxlog.Level

// slog-compatible level constants for geth code that references these.
const (
	LevelTrace slog.Level = -8
	LevelDebug            = slog.LevelDebug
	LevelInfo             = slog.LevelInfo
	LevelWarn             = slog.LevelWarn
	LevelError            = slog.LevelError
	LevelCrit  slog.Level = 12
)

// Root returns the root logger.
func Root() Logger { return luxlog.Root() }

// SetDefault sets the default global logger.
func SetDefault(l Logger) { luxlog.SetDefault(l) }

// New creates a new logger with the given context.
func New(ctx ...interface{}) Logger { return luxlog.New(ctx...) }

// Package-level logging functions. Crit/Fatal log the message to stderr
// then exit — unlike the upstream slog DiscardHandler which silently exits.
func Trace(msg string, ctx ...interface{}) { luxlog.Trace(msg, ctx...) }
func Debug(msg string, ctx ...interface{}) { luxlog.Debug(msg, ctx...) }
func Info(msg string, ctx ...interface{})  { luxlog.Info(msg, ctx...) }
func Warn(msg string, ctx ...interface{})  { luxlog.Warn(msg, ctx...) }
func Error(msg string, ctx ...interface{}) { luxlog.Error(msg, ctx...) }
func Crit(msg string, ctx ...interface{})  { luxlog.Crit(msg, ctx...) }
func Fatal(msg string, ctx ...interface{}) { luxlog.Fatal(msg, ctx...) }
func Log(level Level, msg string, ctx ...interface{}) {
	luxlog.Log(level, msg, ctx...)
}

// NewLogger creates a new logger from an slog.Handler (compatibility shim).
func NewLogger(_ slog.Handler) Logger {
	return luxlog.New()
}

// NewTestLogger returns a logger suitable for testing.
func NewTestLogger(level ...Level) Logger {
	return luxlog.NewTestLogger(level...)
}

// NewNoOpLogger returns a disabled logger.
func NewNoOpLogger() Logger { return luxlog.Noop() }

// FormatLogfmtUint64 formats a uint64 as a compact string.
func FormatLogfmtUint64(n uint64) string { return luxlog.FormatLogfmtUint64(n) }

// SetSlogDefault sets the slog default logger (compatibility).
func SetSlogDefault(l Logger) { SetDefault(l) }

// Handler shims for slog.Handler backward compatibility.

func DiscardHandler() slog.Handler { return &discardHandler{} }

func NewTerminalHandler(wr io.Writer, useColor bool) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func NewTerminalHandlerWithLevel(wr io.Writer, lvl slog.Level, useColor bool) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{Level: lvl})
}

func JSONHandler(wr io.Writer) slog.Handler {
	return slog.NewJSONHandler(wr, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func JSONHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewJSONHandler(wr, &slog.HandlerOptions{Level: level})
}

func LogfmtHandler(wr io.Writer) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func LogfmtHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{Level: level})
}

type discardHandler struct{}

func (h *discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (h *discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h *discardHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *discardHandler) WithGroup(_ string) slog.Handler               { return h }

// Ensure package compiles with stderr reference.
func init() { _ = os.Stderr }
