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
	"errors"
	"fmt"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects libvirt domain data on Linux hosts via the virsh CLI. virsh
// must be on PATH and able to connect to the system daemon, qemu:///system,
// which every call names, so root and other users see the same domains.
// When virsh is absent or `virsh version` does not answer, Collect returns
// nil with no error: not every Linux host is a KVM hypervisor. Once the
// daemon answers, a failed read is an error.
type Linux struct {
	base

	// FS is where the collector looks for the daemons' sockets. A nil FS
	// skips that check, and only the text of virsh's error decides.
	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect queries virsh for version, URI, and full domain list. Returns nil
// (no error) when virsh is absent or the daemon is not running, and an
// error when virsh fails another way, such as no permission on the
// daemon's socket, or when the URI or the domain list cannot be read.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return nil, nil
	}

	// Probe: virsh absent or the daemon down gives no fact; any other
	// failure, such as no permission on the daemon's socket, is an error.
	verOut, err := l.virsh(ctx, "version")
	if err != nil {
		if l.noHypervisor(err) && ctx.Err() == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("read virsh version: %w", err)
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
	// dominfo. A failed dominfo keeps the domain with its name and state
	// only, and its error goes into the joined error, so the result is
	// partial. A context that is done stops the collection.
	var errs []error
	for i := range domains {
		diOut, diErr := l.virsh(ctx, "dominfo", domains[i].Name)
		if diErr != nil {
			errs = append(errs, fmt.Errorf("read virsh dominfo %s: %w", domains[i].Name, diErr))
			continue
		}
		parseVirshDominfo(diOut, &domains[i])
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("read virsh dominfo: %w", err)
	}

	if len(domains) > 0 {
		info.Domains = domains
	}
	return info, errors.Join(errs...)
}

// systemURI is the libvirt connection of the system daemon. virsh with no
// -c connects to qemu:///session for a user who is not root, which has
// other domains, usually none, than qemu:///system for root.
const systemURI = "qemu:///system"

// daemonSockets are the sockets that the libvirt daemons listen on: the
// monolithic libvirtd and the modular virtqemud, under /run and under its
// older /var/run name.
var daemonSockets = []string{
	"/run/libvirt/libvirt-sock",
	"/run/libvirt/virtqemud-sock",
	"/var/run/libvirt/libvirt-sock",
	"/var/run/libvirt/virtqemud-sock",
}

// noHypervisor reports whether err from `virsh version` says that there
// is no hypervisor to ask: virsh is not on the host, or its daemon is not
// running, so its socket is missing or refuses the connection.
//
// The text of virsh's error is in English when virsh runs with LC_ALL=C,
// as source.Commands does. A Runner that keeps the host's locale gets a
// translated message, so a host with none of the daemons' sockets also
// counts as no hypervisor, whatever the message says.
func (l *Linux) noHypervisor(
	err error,
) bool {
	msg := err.Error()

	return collector.Absent(err) ||
		strings.Contains(msg, "No such file or directory") ||
		strings.Contains(msg, "Connection refused") ||
		l.noDaemonSocket()
}

// noDaemonSocket reports whether none of daemonSockets is on the host. It
// is false when the FS is nil, or when a socket cannot be checked.
func (l *Linux) noDaemonSocket() bool {
	if l.FS == nil {
		return false
	}
	for _, path := range daemonSockets {
		if _, err := l.FS.Stat(path); !collector.Absent(err) {
			return false
		}
	}

	return true
}

// virsh runs virsh against the system daemon with args.
func (l *Linux) virsh(
	ctx context.Context,
	args ...string,
) ([]byte, error) {
	return l.Exec.Execute(ctx, "virsh", append([]string{"-c", systemURI}, args...)...)
}
