package log

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// GlogHandler is a compatibility type that wraps a regular slog handler
// and adds glog-specific methods like Verbosity and Vmodule
type GlogHandler struct {
	handler  slog.Handler
	level    slog.Level
	vmodules map[string]slog.Level
	mu       sync.RWMutex
}

// NewGlogHandler creates a new glog-compatible handler
func NewGlogHandler(h slog.Handler) *GlogHandler {
	return &GlogHandler{
		handler:  h,
		level:    slog.LevelInfo,
		vmodules: make(map[string]slog.Level),
	}
}

// Handle implements slog.Handler
func (g *GlogHandler) Handle(ctx context.Context, r slog.Record) error {
	return g.handler.Handle(ctx, r)
}

// Enabled implements slog.Handler
func (g *GlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return g.handler.Enabled(ctx, level)
}

// WithAttrs implements slog.Handler
func (g *GlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &GlogHandler{
		handler:  g.handler.WithAttrs(attrs),
		level:    g.level,
		vmodules: g.vmodules,
	}
}

// WithGroup implements slog.Handler
func (g *GlogHandler) WithGroup(name string) slog.Handler {
	return &GlogHandler{
		handler:  g.handler.WithGroup(name),
		level:    g.level,
		vmodules: g.vmodules,
	}
}

// Verbosity sets the global log verbosity level
func (g *GlogHandler) Verbosity(level slog.Level) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.level = level
	// Update the underlying handler if it supports level setting
	if setter, ok := g.handler.(interface{ SetLevel(slog.Level) }); ok {
		setter.SetLevel(level)
	}
}

// Vmodule sets the log verbosity pattern. The pattern syntax is module=level[,module=level]...
func (g *GlogHandler) Vmodule(pattern string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Clear existing vmodules
	g.vmodules = make(map[string]slog.Level)
	
	// Parse the pattern
	if pattern != "" {
		pairs := strings.Split(pattern, ",")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				module := strings.TrimSpace(parts[0])
				level := strings.TrimSpace(parts[1])
				// Convert level string to slog.Level
				var lvl slog.Level
				switch level {
				case "0":
					lvl = slog.LevelError
				case "1":
					lvl = slog.LevelWarn
				case "2":
					lvl = slog.LevelInfo
				case "3":
					lvl = slog.LevelDebug
				default:
					// Try to parse as integer
					continue
				}
				g.vmodules[module] = lvl
			}
		}
	}
	return nil
}

// SetDefault sets the default global logger
func SetDefault(l Logger) {
	if logger, ok := l.(*logger); ok {
		slog.SetDefault(logger.inner)
		defaultLogger = l
	}
}