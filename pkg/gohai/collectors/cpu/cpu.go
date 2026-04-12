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

// Package cpu collects CPU topology and feature facts.
package cpu

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds CPU information. Includes aggregate facts (total logical,
// physical cores, socket count) plus model/feature data from the first
// CPU (assumed homogeneous — accurate for ~99% of hosts).
type Info struct {
	Total     int      `json:"total"`                // logical CPU count
	Real      int      `json:"real"`                 // physical socket count
	Cores     int      `json:"cores"`                // physical core count
	ModelName string   `json:"model_name,omitempty"` // human-readable CPU name
	VendorID  string   `json:"vendor_id,omitempty"`
	Family    string   `json:"family,omitempty"`
	Model     string   `json:"model,omitempty"`
	Stepping  int32    `json:"stepping,omitempty"`
	Mhz       float64  `json:"mhz,omitempty"`
	CacheSize int32    `json:"cache_size,omitempty"` // KB
	Flags     []string `json:"flags,omitempty"`
}

// Collector is the public interface every cpu variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "cpu" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the cpu variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// infoFn is the injection seam for gopsutil's cpu.InfoWithContext.
// Kept private so importers don't transitively need gopsutil. Swapped
// in tests via SetInfoFn (export_test.go).
var infoFn = cpu.InfoWithContext

// countsFn is the injection seam for gopsutil's cpu.CountsWithContext.
// Kept private alongside infoFn. Swapped via SetCountsFn.
var countsFn = cpu.CountsWithContext

// readCPU is the production bridge to gopsutil.
func readCPU(
	ctx context.Context,
) (*Info, error) {
	stats, err := infoFn(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{}
	// Total logical CPUs.
	if logical, err := countsFn(ctx, true); err == nil {
		info.Total = logical
	}
	// Physical socket count — distinct PhysicalID values. On macOS
	// gopsutil returns one InfoStat, so sockets == 1.
	sockets := map[string]struct{}{}
	var totalCores int32
	for _, s := range stats {
		sockets[s.PhysicalID] = struct{}{}
		totalCores += s.Cores
	}
	info.Real = len(sockets)
	info.Cores = int(totalCores)
	if len(stats) > 0 {
		s := stats[0]
		info.ModelName = s.ModelName
		info.VendorID = s.VendorID
		info.Family = s.Family
		info.Model = s.Model
		info.Stepping = s.Stepping
		info.Mhz = s.Mhz
		info.CacheSize = s.CacheSize
		info.Flags = s.Flags
	}
	return info, nil
}
