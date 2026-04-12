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

package platform

import (
	"context"
	"runtime"

	"github.com/shirou/gopsutil/v4/host"
)

// Linux collects platform identification on Linux hosts.
// Wraps gopsutil.host.Info and applies Ohai's platform-ID remap so
// distro names are canonical (amzn→amazon, rhel→redhat, etc.).
type Linux struct {
	base

	HostInfoFn func(context.Context) (*host.InfoStat, error)
}

// NewLinux returns a Linux variant wired to gopsutil.
func NewLinux() *Linux {
	return &Linux{HostInfoFn: host.InfoWithContext}
}

// Collect returns platform Info. Applies canonicalizePlatform to the
// platform name so consumers get a stable identifier.
func (l *Linux) Collect(ctx context.Context) (any, error) {
	h, err := l.HostInfoFn(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return &Info{OS: runtime.GOOS, Architecture: runtime.GOARCH}, nil
	}
	return &Info{
		OS:           runtime.GOOS,
		Name:         canonicalizePlatform(h.Platform),
		Version:      h.PlatformVersion,
		Family:       h.PlatformFamily,
		Architecture: runtime.GOARCH,
	}, nil
}
