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

package cpu

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"
)

// ReadCPU exposes the private readCPU bridge to the external cpu_test
// package.
var ReadCPU = readCPU

// SetInfoFn swaps the private gopsutil cpu.InfoWithContext call backing
// readCPU. Returns a restore func the caller must defer.
func SetInfoFn(fn func(context.Context) ([]cpu.InfoStat, error)) (restore func()) {
	orig := infoFn
	infoFn = fn
	return func() { infoFn = orig }
}

// SetCountsFn swaps the private gopsutil cpu.CountsWithContext call
// backing readCPU. Returns a restore func the caller must defer.
func SetCountsFn(fn func(context.Context, bool) (int, error)) (restore func()) {
	orig := countsFn
	countsFn = fn
	return func() { countsFn = orig }
}
