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

// Linux collects hostname facts on Linux. Wraps gopsutil.host.Info
// (short hostname), os.Hostname (machine_name), net.LookupHost +
// net.LookupAddr (FQDN canonicalization via reverse DNS).
type Linux struct {
	base

	HostInfoFn   func(context.Context) (*host.InfoStat, error)
	OSHostnameFn func() (string, error)
	LookupHostFn func(string) ([]string, error)
	LookupAddrFn func(string) ([]string, error)
}

// NewLinux returns a Linux variant wired to stdlib + gopsutil.
func NewLinux() *Linux {
	return &Linux{
		HostInfoFn:   host.InfoWithContext,
		OSHostnameFn: os.Hostname,
		LookupHostFn: net.LookupHost,
		LookupAddrFn: net.LookupAddr,
	}
}

// Collect returns hostname facts.
func (l *Linux) Collect(ctx context.Context) (any, error) {
	return resolve(ctx, l.HostInfoFn, l.OSHostnameFn, l.LookupHostFn, l.LookupAddrFn)
}
