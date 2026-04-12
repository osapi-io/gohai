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

package network

import (
	"context"

	"github.com/shirou/gopsutil/v4/net"
)

// ReadInterfaces exposes the private readInterfaces bridge to the
// external network_test package.
var ReadInterfaces = readInterfaces

// SetInterfacesFn swaps the private gopsutil net.InterfacesWithContext
// call backing readInterfaces. Returns a restore func the caller must
// defer.
func SetInterfacesFn(
	fn func(context.Context) (net.InterfaceStatList, error),
) (restore func()) {
	orig := interfacesFn
	interfacesFn = fn
	return func() { interfacesFn = orig }
}

// SetIOCountersFn swaps the private gopsutil net.IOCountersWithContext
// call backing readInterfaces. Returns a restore func the caller must
// defer.
func SetIOCountersFn(
	fn func(context.Context, bool) ([]net.IOCountersStat, error),
) (restore func()) {
	orig := ioCountersFn
	ioCountersFn = fn
	return func() { ioCountersFn = orig }
}
