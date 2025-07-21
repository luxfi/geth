// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"
	"io"
	stdslog "log/slog"
	"runtime"
	"strings"

	"github.com/luxfi/geth/log"
	gethlog "github.com/luxfi/geth/log"
	"golang.org/x/exp/slog"
)

type GethLogger struct {
	gethlog.Logger

	logLevel *slog.LevelVar
}

// stdLevelVar wraps exp/slog.LevelVar to implement log/slog.Leveler
type stdLevelVar slog.LevelVar

func (s *stdLevelVar) Level() stdslog.Level {
	return stdslog.Level((*slog.LevelVar)(s).Level())
}

// InitLogger initializes logger with alias and sets the log level and format with the original [os.StdErr] interface
// along with the context logger.
func InitLogger(alias string, level string, jsonFormat bool, writer io.Writer) (GethLogger, error) {
	logLevel := &slog.LevelVar{}

	var handler slog.Handler
	if jsonFormat {
		chainStr := fmt.Sprintf("%s Chain", alias)
		// Convert exp/slog level to standard library level for geth log functions
		stdLevel := stdslog.Leveler((*stdLevelVar)(logLevel))
		stdHandler := log.JSONHandlerWithLevel(writer, stdLevel)
		// Wrap the standard handler as exp/slog.Handler
		handler = &stdToExpHandler{Handler: stdHandler}
		handler = &addContext{Handler: handler, logger: chainStr}
	} else {
		useColor := false
		chainStr := fmt.Sprintf("<%s Chain> ", alias)
		stdLevel := stdslog.Leveler((*stdLevelVar)(logLevel))
		termHandler := log.NewTerminalHandlerWithLevel(writer, stdLevel, useColor)
		termHandler.Prefix = func(r stdslog.Record) string {
			file, line := getSourceStd(r)
			if file != "" {
				return fmt.Sprintf("%s%s:%d ", chainStr, file, line)
			}
			return chainStr
		}
		// Need to wrap the termHandler as exp/slog.Handler
		handler = &stdToExpHandler{Handler: termHandler}
	}

	// Create handler by wrapping the exp/slog handler with our adapter
	adaptedHandler := &slogAdapter{handler: handler}
	c := GethLogger{
		Logger:   gethlog.NewLogger(adaptedHandler),
		logLevel: logLevel,
	}

	if err := c.SetLogLevel(level); err != nil {
		return GethLogger{}, err
	}
	gethlog.SetDefault(c.Logger)
	return c, nil
}

// SetLogLevel sets the log level of initialized log handler.
func (c *GethLogger) SetLogLevel(level string) error {
	// Set log level
	logLevel, err := log.LvlFromString(level)
	if err != nil {
		return err
	}
	// Convert exp/slog.Level to int value for standard library slog.Level
	c.logLevel.Set(slog.Level(int(logLevel)))
	return nil
}

// locationTrims are trimmed for display to avoid unwieldy log lines.
var locationTrims = []string{
	"geth",
}

func trimPrefixes(s string) string {
	for _, prefix := range locationTrims {
		idx := strings.LastIndex(s, prefix)
		if idx < 0 {
			continue
		}
		slashIdx := strings.Index(s[idx:], "/")
		if slashIdx < 0 || slashIdx+idx >= len(s)-1 {
			continue
		}
		s = s[idx+slashIdx+1:]
	}
	return s
}

func getSource(r slog.Record) (string, int) {
	frames := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := frames.Next()
	return trimPrefixes(frame.File), frame.Line
}

func getSourceStd(r stdslog.Record) (string, int) {
	frames := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := frames.Next()
	return trimPrefixes(frame.File), frame.Line
}

type addContext struct {
	slog.Handler

	logger string
}

func (a *addContext) Handle(ctx context.Context, r slog.Record) error {
	r.Add(slog.String("logger", a.logger))
	file, line := getSource(r)
	if file != "" {
		r.Add(slog.String("caller", fmt.Sprintf("%s:%d", file, line)))
	}
	return a.Handler.Handle(ctx, r)
}

// stdToExpHandler wraps a standard library slog.Handler as exp/slog.Handler
type stdToExpHandler struct {
	Handler stdslog.Handler
}

func (s *stdToExpHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.Handler.Enabled(ctx, stdslog.Level(level))
}

func (s *stdToExpHandler) Handle(ctx context.Context, r slog.Record) error {
	// Convert exp/slog record to standard library record
	stdRecord := stdslog.Record{
		Time:    r.Time,
		Level:   stdslog.Level(r.Level),
		Message: r.Message,
		PC:      r.PC,
	}
	
	// Convert attributes
	r.Attrs(func(attr slog.Attr) bool {
		stdRecord.AddAttrs(stdslog.String(attr.Key, attr.Value.String()))
		return true
	})
	
	return s.Handler.Handle(ctx, stdRecord)
}

func (s *stdToExpHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	stdAttrs := make([]stdslog.Attr, len(attrs))
	for i, attr := range attrs {
		stdAttrs[i] = stdslog.String(attr.Key, attr.Value.String())
	}
	return &stdToExpHandler{Handler: s.Handler.WithAttrs(stdAttrs)}
}

func (s *stdToExpHandler) WithGroup(name string) slog.Handler {
	return &stdToExpHandler{Handler: s.Handler.WithGroup(name)}
}
