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
	"fmt"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects libvirt domain data on Linux hosts via the virsh CLI. virsh
// must be on PATH and able to connect to the system daemon, qemu:///system,
// which every call names, so root and other users see the same domains. When virsh is absent or `virsh version` does not
// answer, Collect returns nil with no error: not every Linux host is a
// KVM hypervisor. Once the daemon answers, a failed read is an error.
type Linux struct {
	base

	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands()}
}

// Collect queries virsh for version, URI, and full domain list. Returns nil
// (no error) when virsh is absent or cannot connect to the daemon, and an
// error when the URI or the domain list cannot be read after that.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return nil, nil
	}

	// Probe: if virsh version fails, virsh is absent or the daemon is down.
	verOut, err := l.virsh(ctx, "version")
	if err != nil {
		return nil, nil
	}

	info := &Info{
		Version: parseVirshVersion(verOut),
	}

	uriOut, err := l.virsh(ctx, "uri")
	if err != nil {
		return nil, fmt.Errorf("read virsh uri: %w", err)
	}
	info.URI = parseVirshURI(uriOut)

	listOut, err := l.virsh(ctx, "list", "--all")
	if err != nil {
		return nil, fmt.Errorf("read virsh list: %w", err)
	}
	domains := parseVirshList(listOut)

	// Enrich each domain with UUID, vCPUs, memory, and autostart via
	// dominfo. A domain can go away between the list and its dominfo, so
	// a failed dominfo keeps the domain with its name and state only.
	for i := range domains {
		diOut, diErr := l.virsh(ctx, "dominfo", domains[i].Name)
		if diErr == nil {
			parseVirshDominfo(diOut, &domains[i])
		}
	}

	if len(domains) > 0 {
		info.Domains = domains
	}
	return info, nil
}

// systemURI is the libvirt connection of the system daemon. virsh with no
// -c connects to qemu:///session for a user who is not root, which has
// other domains, usually none, than qemu:///system for root.
const systemURI = "qemu:///system"

// virsh runs virsh against the system daemon with args.
func (l *Linux) virsh(
	ctx context.Context,
	args ...string,
) ([]byte, error) {
	return l.Exec.Execute(ctx, "virsh", append([]string{"-c", systemURI}, args...)...)
}
