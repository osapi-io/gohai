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

// Package memory collects virtual and swap memory usage.
package memory

import (
	"context"

	"github.com/shirou/gopsutil/v4/mem"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds memory usage data. Byte-valued fields are native bytes —
// Ohai emits kB-suffixed strings; we chose bytes for Go ergonomics,
// documented as a deliberate deviation in docs/collectors/memory.md.
type Info struct {
	Total       uint64  `json:"total"` // bytes
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Free        uint64  `json:"free"`
	Buffers     uint64  `json:"buffers,omitempty"` // Linux only
	Cached      uint64  `json:"cached,omitempty"`  // Linux only
	Swap        *Swap   `json:"swap,omitempty"`
}

// Swap holds swap memory usage.
type Swap struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// Collector is the public interface every memory variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "memory" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the memory variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// readMemory is the production bridge to gopsutil. Combines the
// VirtualMemory + SwapMemory calls and maps into our Info.
func readMemory(
	ctx context.Context,
) (*Info, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{
		Total:       vm.Total,
		Available:   vm.Available,
		Used:        vm.Used,
		UsedPercent: vm.UsedPercent,
		Free:        vm.Free,
		Buffers:     vm.Buffers,
		Cached:      vm.Cached,
	}
	if sm, err := mem.SwapMemoryWithContext(ctx); err == nil && sm.Total > 0 {
		info.Swap = &Swap{
			Total:       sm.Total,
			Used:        sm.Used,
			Free:        sm.Free,
			UsedPercent: sm.UsedPercent,
		}
	}
	return info, nil
}
