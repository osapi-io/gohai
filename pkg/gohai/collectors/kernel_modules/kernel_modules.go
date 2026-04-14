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

// Package kernelmodules enumerates loaded kernel modules (Linux) / kexts
// (macOS). Split from the kernel collector so kernel identity stays
// cheap — enumeration shells out to kextstat on macOS (~280ms) and
// walks /sys/module/<name>/version on Linux, neither of which most
// consumers need. Opt in via --collector.kernel_modules when you need
// the list for CVE correlation, EDR audit, or unsigned-module checks.
package kernelmodules

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the loaded-module map.
type Info struct {
	Modules map[string]Module `json:"modules,omitempty"` // loaded kernel modules (Linux: /proc/modules; macOS: kextstat)
}

// Module describes a single loaded kernel module.
type Module struct {
	Size     uint64 `json:"size,omitempty"`     // bytes
	RefCount int    `json:"refcount,omitempty"` // instances currently loaded
	Version  string `json:"version,omitempty"`  // Linux: /sys/module/<m>/version; macOS: parens field from kextstat
}

// Collector is the public interface every kernel_modules variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "kernel_modules" }
func (base) Category() string       { return collector.CategorySystem }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the kernel_modules variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}
