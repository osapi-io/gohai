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
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux reads the sysconf values with getconf, as Ohai's sysconf plugin
// does.
type Linux struct {
	base

	Exec executor.Executor
}

// NewLinux returns a Linux variant that runs the host's getconf.
func NewLinux() *Linux {
	return &Linux{Exec: executor.New()}
}

// Collect returns the sysconf Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	return collectSysconf(ctx, l.Exec)
}

// collectSysconf runs `getconf <name>` for each value. Linux and macOS
// both have the four names.
func collectSysconf(
	ctx context.Context,
	exec executor.Executor,
) (*Info, error) {
	info := &Info{}
	for _, v := range []struct {
		name string
		dst  *int64
	}{
		{"CLK_TCK", &info.ClkTck},
		{"PAGESIZE", &info.Pagesize},
		{"_NPROCESSORS_CONF", &info.NprocessorsConf},
		{"_NPROCESSORS_ONLN", &info.NprocessorsOnln},
	} {
		out, err := exec.Execute(ctx, "getconf", v.name)
		if err != nil {
			return nil, fmt.Errorf("getconf %s: %w", v.name, err)
		}
		n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("getconf %s: %w", v.name, err)
		}
		*v.dst = n
	}

	return info, nil
}
