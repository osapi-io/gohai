//go:build darwin

// Copyright (c) 2024 John Dewey

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
	"fmt"
	"net"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

var hostInfoFn = host.InfoWithContext

var lookupCNAMEFn = net.LookupCNAME

var fqdnLookupFn = func(
	hostname string,
) string {
	cname, err := lookupCNAMEFn(hostname)
	if err != nil || cname == "" {
		return hostname
	}
	return strings.TrimSuffix(cname, ".")
}

func collect(
	ctx context.Context,
) (any, error) {
	info, err := hostInfoFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Info: %w", err)
	}
	return buildInfo(info.Hostname, fqdnLookupFn), nil
}

func buildInfo(
	short string,
	resolve func(string) string,
) *Info {
	fqdn := resolve(short)
	info := &Info{Hostname: short, FQDN: fqdn}
	if i := strings.IndexByte(fqdn, '.'); i > 0 && i < len(fqdn)-1 {
		info.Domain = fqdn[i+1:]
	}
	return info
}
