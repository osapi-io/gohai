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

// Package hostname collects hostname, machine name, FQDN, and domain
// identification.
package hostname

import (
	"context"
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
// Kept private so importers don't transitively need gopsutil. Swapped
// via SetHostInfoFn (export_test.go).
var hostInfoFn = host.InfoWithContext

// readShortHostname is the production bridge. Wraps the private
// gopsutil call and returns just the short hostname string — callers
// never see gopsutil types.
func readShortHostname(
	ctx context.Context,
) (string, error) {
	h, err := hostInfoFn(ctx)
	if err != nil {
		return "", fmt.Errorf("host.Info: %w", err)
	}
	if h == nil {
		return "", nil
	}
	return h.Hostname, nil
}

// Info holds hostname identification data.
type Info struct {
	Name        string `json:"name"`                   // short hostname (OCSF: device.hostname — leaf matches collector, stripped per CLAUDE.md)
	MachineName string `json:"machine_name,omitempty"` // raw `hostname` output (may be FQDN depending on OS config)
	FQDN        string `json:"fqdn,omitempty"`         // fully qualified (e.g., "web01.example.com")
	Domain      string `json:"domain,omitempty"`       // domain portion (e.g., "example.com")
}

// Collector is the public interface every hostname variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "hostname" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the hostname variant for the host OS. gopsutil and
// net.LookupAddr work cross-platform so Linux and Darwin share logic
// via the shared resolve helper — each struct just wires in the
// right stdlib calls.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// resolve performs the host-identity lookups:
//   - Hostname: short name from gopsutil host.Info (via injected
//     shortHostnameFn).
//   - MachineName: raw os.Hostname (whatever the kernel returns — may
//     or may not be FQDN depending on OS configuration).
//   - FQDN: forward+reverse DNS canonicalization (matches `hostname -f`
//     behavior and Ohai's canonicalize_hostname_with_retries).
//   - Domain: everything after the first `.` in FQDN.
//
// Injectable so tests don't need DNS.
func resolve(
	ctx context.Context,
	shortHostnameFn func(context.Context) (string, error),
	osHostnameFn func() (string, error),
	lookupHostFn func(string) ([]string, error),
	lookupAddrFn func(string) ([]string, error),
) (*Info, error) {
	short, err := shortHostnameFn(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{Name: short}
	if raw, err := osHostnameFn(); err == nil {
		info.MachineName = raw
	}

	info.FQDN, info.Domain = canonicalFQDN(info.Name, lookupHostFn, lookupAddrFn)
	return info, nil
}

// canonicalFQDN resolves short hostname → IP → PTR (reverse DNS) to
// find the canonical FQDN. Falls back to the short name if no reverse
// record exists. Matches `hostname -f` semantics and Ohai.
func canonicalFQDN(
	short string,
	lookupHost func(string) ([]string, error),
	lookupAddr func(string) ([]string, error),
) (fqdn, domain string) {
	if short == "" {
		return "", ""
	}
	ips, err := lookupHost(short)
	if err != nil || len(ips) == 0 {
		return short, ""
	}
	names, err := lookupAddr(ips[0])
	if err != nil || len(names) == 0 {
		return short, ""
	}
	fqdn = strings.TrimSuffix(names[0], ".")
	if i := strings.Index(fqdn, "."); i >= 0 {
		domain = fqdn[i+1:]
	}
	return fqdn, domain
}
