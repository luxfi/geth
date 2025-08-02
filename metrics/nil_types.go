// Copyright 2021 The go-ethereum Authors
// This file is part of go-ethereum.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package metrics

import "time"

// NilCounter is a no-op implementation of Counter used when metrics are disabled.
type NilCounter struct{}

// Clear is a no-op.
func (NilCounter) Clear() {}

// Dec is a no-op.
func (NilCounter) Dec(int64) {}

// Inc is a no-op.
func (NilCounter) Inc(int64) {}

// Snapshot returns an empty snapshot.
func (NilCounter) Snapshot() CounterSnapshot { return CounterSnapshot(0) }

// NilCounter is a no-op Counter implementation.

// NilCounterFloat64 is a no-op implementation of CounterFloat64 used when metrics are disabled.
type NilCounterFloat64 struct{}

// Clear is a no-op.
func (NilCounterFloat64) Clear() {}

// Dec is a no-op.
func (NilCounterFloat64) Dec(float64) {}

// Inc is a no-op.
func (NilCounterFloat64) Inc(float64) {}

// Snapshot returns an empty snapshot.
func (NilCounterFloat64) Snapshot() CounterFloat64Snapshot { return CounterFloat64Snapshot(0) }

// NilGauge is a no-op implementation of Gauge used when metrics are disabled.
type NilGauge struct{}

// Clear is a no-op.
func (NilGauge) Clear() {}

// Dec is a no-op.
func (NilGauge) Dec(int64) {}

// Inc is a no-op.
func (NilGauge) Inc(int64) {}

// Snapshot returns an empty snapshot.
func (NilGauge) Snapshot() GaugeSnapshot { return GaugeSnapshot(0) }

// Update is a no-op.
func (NilGauge) Update(int64) {}

// Value returns 0.
func (NilGauge) Value() int64 { return 0 }

// NilGaugeFloat64 is a no-op implementation of GaugeFloat64 used when metrics are disabled.
type NilGaugeFloat64 struct{}

// Clear is a no-op.
func (NilGaugeFloat64) Clear() {}

// Dec is a no-op.
func (NilGaugeFloat64) Dec(float64) {}

// Inc is a no-op.
func (NilGaugeFloat64) Inc(float64) {}

// Snapshot returns an empty snapshot.
func (NilGaugeFloat64) Snapshot() GaugeFloat64Snapshot { return GaugeFloat64Snapshot(0) }

// Update is a no-op.
func (NilGaugeFloat64) Update(float64) {}

// Value returns 0.0.
func (NilGaugeFloat64) Value() float64 { return 0.0 }

// NilGaugeInfo is a no-op implementation of GaugeInfo used when metrics are disabled.
type NilGaugeInfo struct{}

// Clear is a no-op.
func (NilGaugeInfo) Clear() {}

// Snapshot returns an empty snapshot.
func (NilGaugeInfo) Snapshot() GaugeInfoSnapshot { return GaugeInfoSnapshot{} }

// Update is a no-op.
func (NilGaugeInfo) Update(GaugeInfoValue) {}

// Value returns an empty value.
func (NilGaugeInfo) Value() GaugeInfoValue { return GaugeInfoValue{} }

// NilEWMA is a no-op implementation of EWMA used when metrics are disabled.
type NilEWMA struct{}

// Rate returns 0.0.
func (NilEWMA) Rate() float64 { return 0.0 }

// Snapshot returns an empty snapshot.
func (NilEWMA) Snapshot() EWMASnapshot { return EWMASnapshot(0.0) }

// Tick is a no-op.
func (NilEWMA) Tick() {}

// Update is a no-op.
func (NilEWMA) Update(int64) {}

// NilHealthcheck is a no-op implementation of Healthcheck used when metrics are disabled.
type NilHealthcheck struct{}

// Check is a no-op.
func (NilHealthcheck) Check() {}

// Error returns nil.
func (NilHealthcheck) Error() error { return nil }

// Healthy returns true.
func (NilHealthcheck) Healthy() bool { return true }

// NilHistogram is a no-op implementation of Histogram used when metrics are disabled.
type NilHistogram struct{}

// Clear is a no-op.
func (NilHistogram) Clear() {}

// Count returns 0.
func (NilHistogram) Count() int64 { return 0 }

// Max returns 0.
func (NilHistogram) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilHistogram) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilHistogram) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilHistogram) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilHistogram) Percentiles([]float64) []float64 { return []float64{} }

// Sample returns a nil sample.
func (NilHistogram) Sample() Sample { return NilSample{} }

// Snapshot returns an empty snapshot.
func (NilHistogram) Snapshot() HistogramSnapshot { return NilHistogramSnapshot{} }

// StdDev returns 0.0.
func (NilHistogram) StdDev() float64 { return 0.0 }

// Sum returns 0.
func (NilHistogram) Sum() int64 { return 0 }

// Update is a no-op.
func (NilHistogram) Update(int64) {}

// Variance returns 0.0.
func (NilHistogram) Variance() float64 { return 0.0 }

// NilMeter is a no-op implementation of Meter used when metrics are disabled.
type NilMeter struct{}

// Count returns 0.
func (NilMeter) Count() int64 { return 0 }

// Mark is a no-op.
func (NilMeter) Mark(int64) {}

// Rate1 returns 0.0.
func (NilMeter) Rate1() float64 { return 0.0 }

// Rate5 returns 0.0.
func (NilMeter) Rate5() float64 { return 0.0 }

// Rate15 returns 0.0.
func (NilMeter) Rate15() float64 { return 0.0 }

// RateMean returns 0.0.
func (NilMeter) RateMean() float64 { return 0.0 }

// Snapshot returns an empty snapshot.
func (NilMeter) Snapshot() *MeterSnapshot { return &MeterSnapshot{} }

// Stop is a no-op.
func (NilMeter) Stop() {}

// NilResettingTimer is a no-op implementation of ResettingTimer used when metrics are disabled.
type NilResettingTimer struct{}

// Count returns 0.
func (NilResettingTimer) Count() int64 { return 0 }

// Max returns 0.
func (NilResettingTimer) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilResettingTimer) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilResettingTimer) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilResettingTimer) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilResettingTimer) Percentiles([]float64) []float64 { return []float64{} }

// Snapshot returns an empty snapshot.
func (NilResettingTimer) Snapshot() *ResettingTimerSnapshot { return &ResettingTimerSnapshot{} }

// StdDev returns 0.0.
func (NilResettingTimer) StdDev() float64 { return 0.0 }

// Sum returns 0.
func (NilResettingTimer) Sum() int64 { return 0 }

// Time is a no-op and returns a no-op function.
func (NilResettingTimer) Time() func() { return func() {} }

// Update is a no-op.
func (NilResettingTimer) Update(time.Duration) {}

// UpdateSince is a no-op.
func (NilResettingTimer) UpdateSince(time.Time) {}

// Variance returns 0.0.
func (NilResettingTimer) Variance() float64 { return 0.0 }

// NilSample is a no-op implementation of Sample used when metrics are disabled.
type NilSample struct{}

// Clear is a no-op.
func (NilSample) Clear() {}

// Count returns 0.
func (NilSample) Count() int64 { return 0 }

// Max returns 0.
func (NilSample) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilSample) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilSample) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilSample) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilSample) Percentiles([]float64) []float64 { return []float64{} }

// Size returns 0.
func (NilSample) Size() int { return 0 }

// Snapshot returns an empty snapshot.
func (NilSample) Snapshot() *sampleSnapshot { return &sampleSnapshot{} }

// StdDev returns 0.0.
func (NilSample) StdDev() float64 { return 0.0 }

// Sum returns 0.
func (NilSample) Sum() int64 { return 0 }

// Update is a no-op.
func (NilSample) Update(int64) {}

// Values returns empty slice.
func (NilSample) Values() []int64 { return []int64{} }

// Variance returns 0.0.
func (NilSample) Variance() float64 { return 0.0 }

// NilTimer is a no-op implementation of Timer used when metrics are disabled.
type NilTimer struct{}

// Count returns 0.
func (NilTimer) Count() int64 { return 0 }

// Max returns 0.
func (NilTimer) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilTimer) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilTimer) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilTimer) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilTimer) Percentiles([]float64) []float64 { return []float64{} }

// Rate1 returns 0.0.
func (NilTimer) Rate1() float64 { return 0.0 }

// Rate5 returns 0.0.
func (NilTimer) Rate5() float64 { return 0.0 }

// Rate15 returns 0.0.
func (NilTimer) Rate15() float64 { return 0.0 }

// RateMean returns 0.0.
func (NilTimer) RateMean() float64 { return 0.0 }

// Snapshot returns an empty snapshot.
func (NilTimer) Snapshot() *TimerSnapshot { return &TimerSnapshot{} }

// StdDev returns 0.0.
func (NilTimer) StdDev() float64 { return 0.0 }

// Stop is a no-op.
func (NilTimer) Stop() {}

// Sum returns 0.
func (NilTimer) Sum() int64 { return 0 }

// Time is a no-op and returns a no-op function.
func (NilTimer) Time() func() { return func() {} }

// Update is a no-op.
func (NilTimer) Update(time.Duration) {}

// UpdateSince is a no-op.
func (NilTimer) UpdateSince(time.Time) {}

// Variance returns 0.0.
func (NilTimer) Variance() float64 { return 0.0 }

// Nil snapshot implementations

// NilHistogramSnapshot is a no-op implementation of HistogramSnapshot.
type NilHistogramSnapshot struct{}

// Count returns 0.
func (NilHistogramSnapshot) Count() int64 { return 0 }

// Max returns 0.
func (NilHistogramSnapshot) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilHistogramSnapshot) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilHistogramSnapshot) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilHistogramSnapshot) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilHistogramSnapshot) Percentiles([]float64) []float64 { return []float64{} }

// Size returns 0.
func (NilHistogramSnapshot) Size() int { return 0 }

// StdDev returns 0.0.
func (NilHistogramSnapshot) StdDev() float64 { return 0.0 }

// Sum returns 0.
func (NilHistogramSnapshot) Sum() int64 { return 0 }

// Variance returns 0.0.
func (NilHistogramSnapshot) Variance() float64 { return 0.0 }

// NilMeterSnapshot is a no-op implementation of MeterSnapshot.
type NilMeterSnapshot struct{}

// Count returns 0.
func (NilMeterSnapshot) Count() int64 { return 0 }

// Rate1 returns 0.0.
func (NilMeterSnapshot) Rate1() float64 { return 0.0 }

// Rate5 returns 0.0.
func (NilMeterSnapshot) Rate5() float64 { return 0.0 }

// Rate15 returns 0.0.
func (NilMeterSnapshot) Rate15() float64 { return 0.0 }

// RateMean returns 0.0.
func (NilMeterSnapshot) RateMean() float64 { return 0.0 }

// NilResettingTimerSnapshot is a no-op implementation of ResettingTimerSnapshot.
type NilResettingTimerSnapshot struct{}

// Count returns 0.
func (NilResettingTimerSnapshot) Count() int64 { return 0 }

// Max returns 0.
func (NilResettingTimerSnapshot) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilResettingTimerSnapshot) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilResettingTimerSnapshot) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilResettingTimerSnapshot) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilResettingTimerSnapshot) Percentiles([]float64) []float64 { return []float64{} }

// StdDev returns 0.0.
func (NilResettingTimerSnapshot) StdDev() float64 { return 0.0 }

// Sum returns 0.
func (NilResettingTimerSnapshot) Sum() int64 { return 0 }

// Variance returns 0.0.
func (NilResettingTimerSnapshot) Variance() float64 { return 0.0 }

// NilTimerSnapshot is a no-op implementation of TimerSnapshot.
type NilTimerSnapshot struct{}

// Count returns 0.
func (NilTimerSnapshot) Count() int64 { return 0 }

// Max returns 0.
func (NilTimerSnapshot) Max() int64 { return 0 }

// Mean returns 0.0.
func (NilTimerSnapshot) Mean() float64 { return 0.0 }

// Min returns 0.
func (NilTimerSnapshot) Min() int64 { return 0 }

// Percentile returns 0.0.
func (NilTimerSnapshot) Percentile(float64) float64 { return 0.0 }

// Percentiles returns empty slice.
func (NilTimerSnapshot) Percentiles([]float64) []float64 { return []float64{} }

// Rate1 returns 0.0.
func (NilTimerSnapshot) Rate1() float64 { return 0.0 }

// Rate5 returns 0.0.
func (NilTimerSnapshot) Rate5() float64 { return 0.0 }

// Rate15 returns 0.0.
func (NilTimerSnapshot) Rate15() float64 { return 0.0 }

// RateMean returns 0.0.
func (NilTimerSnapshot) RateMean() float64 { return 0.0 }

// StdDev returns 0.0.
func (NilTimerSnapshot) StdDev() float64 { return 0.0 }

// Sum returns 0.
func (NilTimerSnapshot) Sum() int64 { return 0 }

// Variance returns 0.0.
func (NilTimerSnapshot) Variance() float64 { return 0.0 }