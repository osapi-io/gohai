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

package linode

import "net"

// SetAptSourcesPath swaps the apt sources.list path tests point at.
func SetAptSourcesPath(
	path string,
) func() {
	orig := aptSourcesPath
	aptSourcesPath = path
	return func() { aptSourcesPath = orig }
}

// SetInterfaceAddrs swaps the interface-address lookup seam tests
// use to inject canned addresses without touching real NICs.
func SetInterfaceAddrs(
	fn func(name string) ([]net.Addr, error),
) func() {
	orig := interfaceAddrs
	interfaceAddrs = fn
	return func() { interfaceAddrs = orig }
}

// DefaultInterfaceAddrs exposes the production lookup for tests so
// the real net.InterfaceByName → iface.Addrs() code path executes.
var DefaultInterfaceAddrs = defaultInterfaceAddrs
