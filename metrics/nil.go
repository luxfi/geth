// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

// Nil types are empty structs used to detect disabled/nil metrics
// in type switches (used by prometheus gatherer)

type NilCounter struct{}
type NilCounterFloat64 struct{}
type NilEWMA struct{}
type NilGauge struct{}
type NilGaugeFloat64 struct{}
type NilGaugeInfo struct{}
type NilHealthcheck struct{}
type NilHistogram struct{}
type NilMeter struct{}
type NilResettingTimer struct{}
type NilSample struct{}
type NilTimer struct{}
