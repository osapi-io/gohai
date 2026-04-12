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

package hostname

import (
	"context"
	"net"
	"os"

	"github.com/shirou/gopsutil/v4/host"
)

// Darwin collects hostname facts on macOS. Same wiring as Linux — the
// underlying syscalls are cross-platform. Separate struct type
// preserves the osapi dispatch pattern and lets macOS diverge if Apple
// introduces an API change.
type Darwin struct {
	base

	HostInfoFn   func(context.Context) (*host.InfoStat, error)
	OSHostnameFn func() (string, error)
	LookupHostFn func(string) ([]string, error)
	LookupAddrFn func(string) ([]string, error)
}

// NewDarwin returns a Darwin variant wired to stdlib + gopsutil.
func NewDarwin() *Darwin {
	return &Darwin{
		HostInfoFn:   host.InfoWithContext,
		OSHostnameFn: os.Hostname,
		LookupHostFn: net.LookupHost,
		LookupAddrFn: net.LookupAddr,
	}
}

// Collect returns hostname facts.
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	return resolve(ctx, d.HostInfoFn, d.OSHostnameFn, d.LookupHostFn, d.LookupAddrFn)
}
