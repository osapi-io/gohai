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

// Package load collects system load averages (1/5/15-minute).
package load

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/load"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the 1/5/15-minute load averages. Matches `uptime(1)` /
// `getloadavg(3)` output — values are runnable+uninterruptible task
// counts averaged over each window.
type Info struct {
	One     float64 `json:"one"`     // 1-minute load average
	Five    float64 `json:"five"`    // 5-minute load average
	Fifteen float64 `json:"fifteen"` // 15-minute load average
}

// Collector is the public interface every load variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "load" }
func (base) Category() string       { return collector.CategoryMisc }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the load collector variant for the host OS. Both variants
// are thin wrappers around gopsutil's load.AvgWithContext (which calls
// getloadavg(3) on both Linux and macOS); the per-OS split exists for
// consistency with the rest of the codebase, not because the collection
// logic differs.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// avgFn is the injection seam for gopsutil's load.AvgWithContext.
// Kept private — never exposed on the public SDK surface so callers
// don't transitively need to depend on gopsutil types. Swapped in
// tests via SetAvgFn (export_test.go).
var avgFn = load.AvgWithContext

// readAverages is the production bridge. Wraps the private gopsutil
// call and maps its result onto our Info type — callers of the
// collector never see gopsutil types.
func readAverages(
	ctx context.Context,
) (*Info, error) {
	a, err := avgFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("load.Avg: %w", err)
	}
	return &Info{One: a.Load1, Five: a.Load5, Fifteen: a.Load15}, nil
}
