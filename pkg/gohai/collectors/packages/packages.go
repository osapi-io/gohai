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

// Package packages reports the list of installed packages on the host.
// Linux: reads from dpkg-query (Debian/Ubuntu) or rpm (RHEL/Fedora).
// macOS: reads from Homebrew via `brew list --versions`.
package packages

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Package holds metadata for a single installed package.
type Package struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Architecture   string `json:"architecture,omitempty"`
	PackageManager string `json:"package_manager"`
}

// Info holds the list of installed packages.
type Info struct {
	Packages []Package `json:"packages"`
}

// Collector is the public interface every packages variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields shared by every OS variant.
type base struct{}

// Name returns "packages".
func (base) Name() string { return "packages" }

// Category returns "software".
func (base) Category() string { return collector.CategorySoftware }

// DefaultEnabled returns false — full package inventory is expensive.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the packages collector variant for the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}
