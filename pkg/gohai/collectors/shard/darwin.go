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

package shard

import (
	"context"
	"os"
)

// Darwin computes a shard seed on macOS from IOPlatformUUID (via
// gopsutil under the hood) + os.Hostname. MachineIDFn is typed in
// `string` so importers don't need gopsutil.
type Darwin struct {
	base

	MachineIDFn func(context.Context) (string, error)
	HostnameFn  func() (string, error)
}

// NewDarwin returns a Darwin variant wired to production bridges.
func NewDarwin() *Darwin {
	return &Darwin{
		MachineIDFn: readMachineUUID,
		HostnameFn:  os.Hostname,
	}
}

// Collect derives the shard seed. macOS has no /etc/machine-id; we use
// the IOPlatformUUID via MachineIDFn. A lookup error is treated as
// "no stable ID" — the seed still computes deterministically over the
// hostname (matches the Linux collector's behavior when
// /etc/machine-id is missing).
func (d *Darwin) Collect(ctx context.Context) (any, error) {
	mid, _ := d.MachineIDFn(ctx)
	host, _ := d.HostnameFn()
	return &Info{Seed: computeSeed(mid, host)}, nil
}
