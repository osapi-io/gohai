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
	"time"

	"github.com/shirou/gopsutil/v4/host"
)

// ReadShortHostname exposes the private readShortHostname bridge to
// the external hostname_test package.
var ReadShortHostname = readShortHostname

// CanonicalFQDN exposes the private canonicalFQDN helper.
var CanonicalFQDN = canonicalFQDN

// SetHostInfoFn swaps the private gopsutil host.InfoWithContext call.
func SetHostInfoFn(
	fn func(context.Context) (*host.InfoStat, error),
) (restore func()) {
	orig := hostInfoFn
	hostInfoFn = fn
	return func() { hostInfoFn = orig }
}

// SetLookupHostFn swaps the private net.LookupHost seam.
func SetLookupHostFn(
	fn func(string) ([]string, error),
) (restore func()) {
	orig := lookupHostFn
	lookupHostFn = fn
	return func() { lookupHostFn = orig }
}

// SetLookupAddrFn swaps the private net.LookupAddr seam.
func SetLookupAddrFn(
	fn func(string) ([]string, error),
) (restore func()) {
	orig := lookupAddrFn
	lookupAddrFn = fn
	return func() { lookupAddrFn = orig }
}

// SetOSHostnameFn swaps the private os.Hostname seam.
func SetOSHostnameFn(
	fn func() (string, error),
) (restore func()) {
	orig := osHostnameFn
	osHostnameFn = fn
	return func() { osHostnameFn = orig }
}

// SetResolverRetries swaps the retry count used by canonicalFQDN.
func SetResolverRetries(
	n int,
) (restore func()) {
	orig := resolverRetries
	resolverRetries = n
	return func() { resolverRetries = orig }
}

// SetResolverBackoff swaps the backoff between resolver retries.
func SetResolverBackoff(
	d time.Duration,
) (restore func()) {
	orig := resolverBackoff
	resolverBackoff = d
	return func() { resolverBackoff = orig }
}
