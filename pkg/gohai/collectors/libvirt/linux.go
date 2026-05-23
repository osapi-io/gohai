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

package libvirt

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects libvirt domain data on Linux hosts via the virsh CLI. virsh
// must be on PATH and able to connect to the libvirt daemon (typically
// qemu:///system). When virsh is absent or the connection fails Collect
// returns nil with no error — not every Linux host is a KVM hypervisor.
type Linux struct {
	base

	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: executor.New()}
}

// Collect queries virsh for version, URI, and full domain list. Returns nil
// (no error) when virsh is absent or cannot connect to the daemon.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return nil, nil
	}

	// Probe: if virsh version fails, virsh is absent or the daemon is down.
	verOut, err := l.Exec.Execute(ctx, "virsh", "version")
	if err != nil {
		return nil, nil
	}

	info := &Info{
		Version: parseVirshVersion(verOut),
	}

	// Collect the connection URI.
	if uriOut, uriErr := l.Exec.Execute(ctx, "virsh", "uri"); uriErr == nil {
		info.URI = parseVirshURI(uriOut)
	}

	// Enumerate domains; errors yield an empty list rather than a failure.
	listOut, listErr := l.Exec.Execute(ctx, "virsh", "list", "--all")
	if listErr != nil {
		return info, nil
	}
	domains := parseVirshList(listOut)

	// Enrich each domain with UUID, vCPUs, memory, and autostart via dominfo.
	for i := range domains {
		diOut, diErr := l.Exec.Execute(ctx, "virsh", "dominfo", domains[i].Name)
		if diErr == nil {
			parseVirshDominfo(diOut, &domains[i])
		}
	}

	if len(domains) > 0 {
		info.Domains = domains
	}
	return info, nil
}
