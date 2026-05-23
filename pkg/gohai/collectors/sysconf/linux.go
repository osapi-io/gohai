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

package sysconf

import (
	"context"
	"fmt"

	gosysconf "github.com/tklauser/go-sysconf"

	"github.com/osapi-io/gohai/internal/collector"
)

// sysconfFn is the injection seam for go-sysconf's Sysconf call.
// Tests swap this via SetSysconfFn in export_test.go to avoid
// syscall dependency.
var sysconfFn = gosysconf.Sysconf

// Linux collects POSIX sysconf values on Linux using
// github.com/tklauser/go-sysconf — a pure Go wrapper around the
// POSIX sysconf(3) syscall. Covers the same four constants as Ohai's
// sysconf plugin (CLK_TCK, PAGESIZE, NPROCESSORS_CONF, NPROCESSORS_ONLN).
type Linux struct {
	base
}

// NewLinux returns a Linux variant.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect calls sysconf for four constants and returns an Info.
// Individual lookup failures are reported as errors.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return collectSysconf()
}

// collectSysconf is the shared implementation for both Linux and Darwin.
// Both platforms expose the same four SC_* constants via POSIX sysconf(3).
func collectSysconf() (*Info, error) {
	clkTck, err := sysconfFn(gosysconf.SC_CLK_TCK)
	if err != nil {
		return nil, fmt.Errorf("sysconf SC_CLK_TCK: %w", err)
	}
	pagesize, err := sysconfFn(gosysconf.SC_PAGESIZE)
	if err != nil {
		return nil, fmt.Errorf("sysconf SC_PAGESIZE: %w", err)
	}
	nconf, err := sysconfFn(gosysconf.SC_NPROCESSORS_CONF)
	if err != nil {
		return nil, fmt.Errorf("sysconf SC_NPROCESSORS_CONF: %w", err)
	}
	nonln, err := sysconfFn(gosysconf.SC_NPROCESSORS_ONLN)
	if err != nil {
		return nil, fmt.Errorf("sysconf SC_NPROCESSORS_ONLN: %w", err)
	}
	return &Info{
		ClkTck:          clkTck,
		Pagesize:        pagesize,
		NprocessorsConf: nconf,
		NprocessorsOnln: nonln,
	}, nil
}
