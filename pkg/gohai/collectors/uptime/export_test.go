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

	"github.com/shirou/gopsutil/v4/host"
)

// ReadBase exposes the private readBase bridge.
var ReadBase = readBase

// SetHostInfoFn swaps the private gopsutil call backing readBase.
func SetHostInfoFn(fn func(context.Context) (*host.InfoStat, error)) (restore func()) {
	orig := hostInfoFn
	hostInfoFn = fn
	return func() { hostInfoFn = orig }
}

// SetReadBaseFn swaps the per-collector readBase seam the Linux and
// Darwin variants call directly.
func SetReadBaseFn(
	fn func(context.Context) (*Info, error),
) (restore func()) {
	orig := readBaseFn
	readBaseFn = fn
	return func() { readBaseFn = orig }
}
