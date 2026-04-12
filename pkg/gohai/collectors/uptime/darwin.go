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

	"github.com/shirou/gopsutil/v4/host"
)

// Darwin collects uptime on macOS. Wraps gopsutil.host.Info.
// No idle-time equivalent — macOS doesn't expose an aggregate CPU idle
// counter equivalent to Linux's /proc/uptime[1].
type Darwin struct {
	base

	HostInfoFn func(context.Context) (*host.InfoStat, error)
}

// NewDarwin returns a Darwin variant wired to gopsutil.
func NewDarwin() *Darwin {
	return &Darwin{HostInfoFn: host.InfoWithContext}
}

// Collect returns uptime facts. IdleSeconds/IdleHuman stay empty on
// darwin (no kernel-level aggregate idle counter).
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	info, err := d.HostInfoFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Info: %w", err)
	}
	return &Info{
		Seconds:  info.Uptime,
		BootTime: info.BootTime,
		Human:    HumanDuration(info.Uptime),
	}, nil
}
