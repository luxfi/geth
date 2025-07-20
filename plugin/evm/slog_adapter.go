// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"golang.org/x/exp/slog"
	stdslog "log/slog"
)

// slogAdapter adapts between golang.org/x/exp/slog and log/slog
type slogAdapter struct {
	handler slog.Handler
}

// Enabled implements log/slog.Handler
func (s *slogAdapter) Enabled(ctx context.Context, level stdslog.Level) bool {
	// Convert standard library slog level to exp/slog level
	expLevel := slog.Level(level)
	return s.handler.Enabled(ctx, expLevel)
}

// Handle implements log/slog.Handler
func (s *slogAdapter) Handle(ctx context.Context, record stdslog.Record) error {
	// Convert standard library record to exp/slog record
	expRecord := slog.Record{
		Time:    record.Time,
		Level:   slog.Level(record.Level),
		Message: record.Message,
		PC:      record.PC,
	}

	// Copy attributes
	record.Attrs(func(attr stdslog.Attr) bool {
		expRecord.Add(slog.String(attr.Key, attr.Value.String()))
		return true
	})

	return s.handler.Handle(ctx, expRecord)
}

// WithAttrs implements log/slog.Handler
func (s *slogAdapter) WithAttrs(attrs []stdslog.Attr) stdslog.Handler {
	expAttrs := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		expAttrs[i] = slog.String(attr.Key, attr.Value.String())
	}
	return &slogAdapter{handler: s.handler.WithAttrs(expAttrs)}
}

// WithGroup implements log/slog.Handler
func (s *slogAdapter) WithGroup(name string) stdslog.Handler {
	return &slogAdapter{handler: s.handler.WithGroup(name)}
}
