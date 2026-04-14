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

// Package hostname collects the short hostname, fully qualified domain
// name, DNS domain, and friendly machine name. Mirrors Ohai's
// `hostname -s` + `hostname` + DNS-canonicalization methodology; FQDN
// canonicalization tolerates transient resolver failures by retrying
// the forward+reverse DNS chain up to three times before falling back
// to the short hostname.
package hostname

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/internal/platform"
)

// Package-level injection seams. Kept private so importers don't
// transitively need gopsutil; swapped via the SetXFn helpers in
// export_test.go. Stdlib calls are swapped the same way so tests avoid
// real DNS traffic and don't sleep.
var (
	hostInfoFn      = host.InfoWithContext
	lookupHostFn    = net.LookupHost
	lookupAddrFn    = net.LookupAddr
	osHostnameFn    = os.Hostname
	resolverRetries = 3
	resolverBackoff = 100 * time.Millisecond
)

// readShortHostname returns just the short hostname from gopsutil's
// host.Info, stripped of the gopsutil types so callers never see them.
// Used as a fallback when the `hostname -s` exec fails.
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
	MachineName string `json:"machine_name,omitempty"` // raw `hostname` output (macOS: ComputerName-derived friendly name)
	FQDN        string `json:"fqdn,omitempty"`         // fully qualified (e.g., "web01.example.com")
	Domain      string `json:"domain,omitempty"`       // domain portion (e.g., "example.com")
}

// Collector is the public interface every hostname variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "hostname" }
func (base) Category() string       { return collector.CategorySystem }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the hostname variant for the host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// collectWithExec is the shared Collect body — Linux and Darwin run
// the exact same commands (`hostname -s` for the short name,
// `hostname` for the friendly machine name), matching Ohai's linux
// and darwin branches.
func collectWithExec(
	ctx context.Context,
	exec executor.Executor,
) (*Info, error) {
	short, shortOK := "", false
	machine, machineOK := "", false

	if exec != nil {
		if out, err := exec.Execute(ctx, "hostname", "-s"); err == nil {
			if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
				short, shortOK = trimmed, true
			}
		}
		if out, err := exec.Execute(ctx, "hostname"); err == nil {
			if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
				machine, machineOK = trimmed, true
			}
		}
	}

	if !shortOK {
		gpShort, err := readShortHostname(ctx)
		if err != nil {
			return nil, err
		}
		short = gpShort
	}
	if !machineOK {
		if raw, err := osHostnameFn(); err == nil {
			machine = raw
		} else {
			machine = short
		}
	}

	info := &Info{Name: short, MachineName: machine}
	info.FQDN, info.Domain = canonicalFQDN(short)
	return info, nil
}

// canonicalFQDN resolves short hostname → IP → PTR (reverse DNS) to
// find the canonical FQDN. Retries the chain up to resolverRetries
// times with resolverBackoff between attempts to ride over transient
// resolver blips (split-horizon resolvers, systemd-resolved startup
// races). Retries only fire on transient failures — a definitive
// "no such host" (net.DNSError.IsNotFound) or an empty answer stops
// the loop immediately. This matters on hosts without a PTR record
// (laptops, minimal containers) where the full retry budget used to
// add ~200ms of pure sleep. On final failure returns the short name
// as FQDN and empty domain, matching Ohai's
// canonicalize_hostname_with_retries behaviour.
func canonicalFQDN(
	short string,
) (fqdn, domain string) {
	if short == "" {
		return "", ""
	}
	for attempt := 0; attempt < resolverRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(resolverBackoff)
		}
		ips, err := lookupHostFn(short)
		if err != nil {
			if isDNSNotFound(err) {
				break
			}
			continue
		}
		if len(ips) == 0 {
			break
		}
		names, err := lookupAddrFn(ips[0])
		if err != nil {
			if isDNSNotFound(err) {
				break
			}
			continue
		}
		if len(names) == 0 {
			break
		}
		fqdn = strings.TrimSuffix(names[0], ".")
		if i := strings.Index(fqdn, "."); i >= 0 {
			domain = fqdn[i+1:]
		}
		return fqdn, domain
	}
	return short, ""
}

// isDNSNotFound reports whether err is a definitive DNS "no such host"
// answer. Used by canonicalFQDN to short-circuit the retry loop on
// failures that won't resolve on a second attempt — transient errors
// (timeouts, IO errors) still drive a retry.
func isDNSNotFound(
	err error,
) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}
