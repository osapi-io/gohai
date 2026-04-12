// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package uptime collects system uptime and boot time facts.
package uptime

import (
	"context"
	"fmt"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds uptime and boot time data.
type Info struct {
	Seconds     uint64 `json:"seconds"`                // seconds since boot
	BootTime    uint64 `json:"boot_time"`              // unix timestamp of boot
	Human       string `json:"human"`                  // human-readable uptime (e.g., "3d 4h 12m 5s")
	IdleSeconds uint64 `json:"idle_seconds,omitempty"` // seconds CPUs have been idle (Linux only; aggregate across cores)
	IdleHuman   string `json:"idle_human,omitempty"`   // human-readable idle time (Linux only)
}

// Collector implements the collector.Collector interface for uptime facts.
type Collector struct{}

// New returns a new uptime Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "uptime".
func (c *Collector) Name() string {
	return "uptime"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers uptime facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}

// HumanDuration formats a second count as "Xd Yh Zm Ws" (omitting zero
// leading units).
func HumanDuration(
	seconds uint64,
) string {
	const (
		minute = 60
		hour   = 60 * minute
		day    = 24 * hour
	)
	d := seconds / day
	h := (seconds % day) / hour
	m := (seconds % hour) / minute
	s := seconds % minute
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", d, h, m, s)
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
