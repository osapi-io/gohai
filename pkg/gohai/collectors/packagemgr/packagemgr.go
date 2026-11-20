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
	"context"
	"strings"

	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/pkg/gohai/family"
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
func (base) Category() string       { return collector.CategorySoftware }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the package_mgr variant for the host OS. The distro
// family drives the variant: Debian → apt; RHEL, Fedora and Amazon →
// dnf/yum; macOS → brew. Generic linux tries each manager in order.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src.
func NewWith(
	src source.Sources,
) Collector {
	switch collector.Family(src) {
	case family.Darwin:
		return collector.Wire(NewDarwin(), src)
	case family.Debian:
		return collector.Wire(NewDebian(), src)
	case family.RHEL, family.Fedora, family.Amazon:
		return collector.Wire(NewRHEL(), src)
	default:
		return collector.Wire(NewLinux(), src)
	}
}

// firstFound is the first of names that the host's shell finds on its
// PATH, and its path: `sh -c "command -v <name>"`, the POSIX form of
// Ohai's `which`. Both are "" when exec is nil or none is found.
func firstFound(
	ctx context.Context,
	exec executor.Executor,
	names ...string,
) (string, string) {
	if exec == nil {
		return "", ""
	}
	for _, n := range names {
		out, err := exec.Execute(ctx, "sh", "-c", "command -v "+n)
		if p := strings.TrimSpace(string(out)); err == nil && strings.HasPrefix(p, "/") {
			return n, p
		}
	}

	return "", ""
}
