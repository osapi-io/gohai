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

// Package sysconf reports POSIX sysconf(3) values for clock tick rate,
// page size, and processor counts. Mirrors Ohai's sysconf plugin but
// uses github.com/tklauser/go-sysconf (already in the module graph as
// a transitive dep) instead of shelling out to getconf — faster, no
// child process, identical semantics. DefaultEnabled is false — niche
// fact most consumers don't need.
package sysconf

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the four POSIX sysconf values gohai collects.
type Info struct {
	// ClkTck is the number of clock ticks per second (SC_CLK_TCK).
	// Used to convert /proc/stat jiffies to wall-clock seconds.
	ClkTck int64 `json:"clk_tck"`

	// Pagesize is the system memory page size in bytes (SC_PAGESIZE).
	Pagesize int64 `json:"pagesize"`

	// NprocessorsConf is the total number of CPUs configured in the
	// kernel (SC_NPROCESSORS_CONF). Includes offline CPUs.
	NprocessorsConf int64 `json:"nprocessors_conf"`

	// NprocessorsOnln is the number of CPUs currently online
	// (SC_NPROCESSORS_ONLN).
	NprocessorsOnln int64 `json:"nprocessors_onln"`
}

// Collector is the public interface every sysconf variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "sysconf" }
func (base) Category() string       { return collector.CategoryMisc }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the sysconf variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}
