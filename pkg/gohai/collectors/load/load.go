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
	"fmt"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds the 1/5/15-minute load averages. Matches `uptime(1)` /
// `getloadavg(3)` output: the values are runnable+uninterruptible task
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

func (base) Name() string                 { return "load" }
func (base) Category() collector.Category { return collector.CategoryMisc }
func (base) DefaultEnabled() bool         { return true }
func (base) Dependencies() []string       { return nil }

// New returns the load collector variant for the host OS: Linux reads
// /proc/loadavg, and macOS runs sysctl -n vm.loadavg, as Ohai does.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// parse is the first three numbers of text, such as /proc/loadavg's
// "0.32 0.28 0.25 1/412 12345" or vm.loadavg's "{ 0.32 0.28 0.25 }".
func parse(
	text string,
) (*Info, error) {
	fields := strings.Fields(strings.Trim(strings.TrimSpace(text), "{}"))
	if len(fields) < 3 {
		return nil, fmt.Errorf("no load averages in %q", text)
	}
	var v [3]float64
	for i := range v {
		n, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return nil, fmt.Errorf("load average %q: %w", fields[i], err)
		}
		v[i] = n
	}

	return &Info{One: v[0], Five: v[1], Fifteen: v[2]}, nil
}
