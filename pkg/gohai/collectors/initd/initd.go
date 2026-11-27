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

// Package initd detects the init system a Linux host is using, such as
// systemd, openrc or runit, by reading /proc/1/comm. A PID 1 named init
// is reported as init: sysvinit, upstart and busybox all use that name.
// On macOS the answer is always launchd.
package initd

import (
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// proc1CommPath is the file whose contents are the name of PID 1.
const proc1CommPath = "/proc/1/comm"

// Info holds init system identification.
type Info struct {
	Name string `json:"name"` // e.g., "systemd", "openrc", "runit", "init", "launchd"
}

// Collector is the public interface every init variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string                 { return "init" }
func (base) Category() collector.Category { return collector.CategorySystem }
func (base) DefaultEnabled() bool         { return true }
func (base) Dependencies() []string       { return nil }

// New returns the init variant for the host OS.
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

// renames maps a PID-1 comm that names a known init system another way
// to that system's name.
var renames = map[string]string{
	"openrc-init": "openrc",
}

// classify gives the init system's name from the raw PID-1 comm. It
// returns "init" as it is: sysvinit, upstart and busybox all run PID 1 as
// init, so the comm does not tell which one it is.
func classify(
	raw string,
) string {
	name := strings.TrimSpace(raw)
	if renamed, ok := renames[name]; ok {
		return renamed
	}

	return name
}
