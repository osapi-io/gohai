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

// Package packagemgr detects which package manager the host is
// configured to use — apt on Debian-family, dnf/yum on RHEL-family,
// brew on macOS, etc. Matches Ohai's conceptual "platform family →
// package manager" mapping.
package packagemgr

import (
	"os/exec"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds package manager identification.
type Info struct {
	Name string `json:"name"`           // "apt", "dnf", "yum", "zypper", "pacman", "apk", "brew"
	Path string `json:"path,omitempty"` // absolute path to the manager binary
}

// Collector is the public interface every package_mgr variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "package_mgr" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the package_mgr variant for the host OS. Distro
// family drives the variant: Debian → apt, RHEL → dnf/yum, macOS →
// brew. Generic linux tries each manager in order.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	case "debian":
		return NewDebian()
	case "rhel":
		return NewRHEL()
	default:
		return NewLinux()
	}
}

// probe looks up a binary via exec.LookPath. Named helper so
// factories assign by reference. Returns the empty string for missing
// binaries — no error distinction needed.
func probe(
	name string,
) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

// firstFound returns the first (name, path) pair from the ordered
// list where the binary is present on the host. Empty name + path if
// none found.
func firstFound(
	probeFn func(string) string,
	names ...string,
) (string, string) {
	for _, n := range names {
		if p := probeFn(n); p != "" {
			return n, p
		}
	}
	return "", ""
}
