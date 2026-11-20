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

package uptime

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// bootSecRE is the seconds of sysctl kern.boottime:
// "{ sec = 1767229145, usec = 0 } Thu Jan  1 00:59:05 2026".
var bootSecRE = regexp.MustCompile(`sec = (\d+)`)

// Darwin collects uptime on macOS: the boot time from sysctl
// kern.boottime, and the seconds since it by the clock. It has no idle
// time.
type Darwin struct {
	base

	Exec executor.Executor
	Now  func() time.Time
}

// NewDarwin returns a Darwin variant that runs the host's sysctl and reads
// its clock.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New(), Now: time.Now}
}

// Collect returns uptime facts.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	out, err := d.Exec.Execute(ctx, "sysctl", "-n", "kern.boottime")
	if err != nil {
		return nil, fmt.Errorf("sysctl kern.boottime: %w", err)
	}
	m := bootSecRE.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("sysctl kern.boottime: no seconds in %q", out)
	}
	boot, _ := strconv.ParseInt(string(m[1]), 10, 64)
	up := d.Now().Unix() - boot
	if up < 0 {
		up = 0
	}

	return &Info{Seconds: uint64(up), BootTime: uint64(boot), Human: HumanDuration(uint64(up))}, nil
}
