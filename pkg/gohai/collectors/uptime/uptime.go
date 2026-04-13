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

// Package uptime collects system uptime, boot time, and (on Linux)
// aggregate CPU idle time.
package uptime

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds uptime and boot time data.
type Info struct {
	Seconds     uint64 `json:"seconds"`                // seconds since boot
	BootTime    uint64 `json:"boot_time"`              // unix timestamp of boot
	Human       string `json:"human"`                  // human-readable uptime (e.g., "3d 4h 12m 5s")
	IdleSeconds uint64 `json:"idle_seconds,omitempty"` // aggregate CPU idle seconds (Linux only)
	IdleHuman   string `json:"idle_human,omitempty"`   // human-readable idle time (Linux only)
}

// Collector is the public interface every uptime variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "uptime" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the uptime collector variant for the host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
// Private — never leaked through a public Fn field. Swapped in tests
// via SetHostInfoFn (export_test.go).
var hostInfoFn = host.InfoWithContext

// readBaseFn is the per-collector seam the Linux and Darwin variants
// call directly. Points at readBase in production; tests swap via
// SetReadBaseFn to bypass gopsutil entirely.
var readBaseFn = readBase

// readBase wraps the private gopsutil call and maps the result onto
// our *Info so consumers of the collector never see gopsutil types.
func readBase(
	ctx context.Context,
) (*Info, error) {
	h, err := hostInfoFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Info: %w", err)
	}
	return &Info{
		Seconds:  h.Uptime,
		BootTime: h.BootTime,
		Human:    HumanDuration(h.Uptime),
	}, nil
}

// HumanDuration formats a second count as "Xd Yh Zm Ws" (omitting zero
// leading units). Exported for consumers that want to render durations
// consistently with uptime's own output.
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
