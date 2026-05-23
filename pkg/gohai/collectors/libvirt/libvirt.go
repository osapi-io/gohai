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

// Package libvirt collects libvirt domain information from a KVM/QEMU host.
// It uses the virsh CLI (part of the libvirt-client / libvirt-bin package)
// rather than the ruby-libvirt gem used by Ohai, since the Go ecosystem has
// no maintained libvirt binding with equivalent coverage. When virsh is not on
// PATH or the connection fails, Collect returns nil with no error.
package libvirt

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Domain describes a single libvirt domain (virtual machine).
type Domain struct {
	Name      string `json:"name"`                 // domain name
	UUID      string `json:"uuid,omitempty"`       // domain UUID
	State     string `json:"state,omitempty"`      // running/paused/shut off/etc.
	VCPUs     int    `json:"vcpus,omitempty"`      // number of virtual CPUs
	MaxMemory string `json:"max_memory,omitempty"` // maximum memory allocation
	Autostart bool   `json:"autostart"`            // whether the domain autostarts
}

// Info holds libvirt host and domain data.
type Info struct {
	URI     string   `json:"uri,omitempty"`     // connection URI (e.g. "qemu:///system")
	Version string   `json:"version,omitempty"` // libvirt daemon version
	Domains []Domain `json:"domains,omitempty"` // list of all domains (running and stopped)
}

// Collector is the public interface every libvirt variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "libvirt".
func (base) Name() string { return "libvirt" }

// Category returns "virtualization".
func (base) Category() string { return collector.CategoryVirtualization }

// DefaultEnabled returns false — libvirt is opt-in only.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the libvirt collector variant appropriate to the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseVirshVersion extracts the libvirt version from `virsh version` output.
// Precedence: "Running against daemon: <ver>" first, then "Using library:
// libvirt <ver>" as a fallback (matches cases where virsh runs without a
// daemon, e.g. QEMU direct). Mirrors Ohai's libvirt_version reporting.
//
// Example virsh version output:
//
//	Compiled against library: libvirt 10.0.0
//	Using library: libvirt 10.0.0
//	Using API: QEMU 10.0.0
//	Running hypervisor: QEMU 8.2.1
//	Running against daemon: 10.0.0
func parseVirshVersion(
	output []byte,
) string {
	var libraryVer string
	sc := bufio.NewScanner(strings.NewReader(string(output)))
	for sc.Scan() {
		line := sc.Text()
		if after, ok := strings.CutPrefix(line, "Running against daemon:"); ok {
			return strings.TrimSpace(after)
		}
		if after, ok := strings.CutPrefix(line, "Using library:"); ok {
			v := strings.TrimSpace(after)
			if after2, ok2 := strings.CutPrefix(v, "libvirt "); ok2 {
				libraryVer = strings.TrimSpace(after2)
			} else {
				libraryVer = v
			}
		}
	}
	return libraryVer
}

// parseVirshURI extracts the connection URI from `virsh uri` output (a single
// line trimmed of whitespace).
func parseVirshURI(
	output []byte,
) string {
	return strings.TrimSpace(string(output))
}

// parseVirshList parses `virsh list --all` output into Domain stubs (name +
// state). Full domain detail (UUID, vCPUs, memory, autostart) is enriched
// per-domain via separate virsh dominfo calls.
//
// virsh list --all output format:
//
//	 Id   Name        State
//	-----------------------------
//	 1    myvm        running
//	 -    stopped-vm  shut off
func parseVirshList(
	output []byte,
) []Domain {
	var domains []Domain
	sc := bufio.NewScanner(strings.NewReader(string(output)))
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		// A valid data row has at least 3 fields: id, name, state (possibly multi-word).
		if len(fields) < 3 {
			continue
		}
		// Skip the header line — first field is "Id".
		if fields[0] == "Id" {
			continue
		}
		// fields[0] = id (or "-"), fields[1] = name, fields[2..] = state words.
		name := fields[1]
		state := strings.Join(fields[2:], " ")
		domains = append(domains, Domain{Name: name, State: state})
	}
	return domains
}

// parseVirshDominfo parses `virsh dominfo <name>` output and updates the
// provided Domain pointer with UUID, VCPUs, MaxMemory, and Autostart.
//
// Output format (key: value pairs):
//
//	Id:             1
//	Name:           myvm
//	UUID:           12345678-1234-1234-1234-123456789abc
//	OS Type:        hvm
//	State:          running
//	CPU(s):         2
//	Max memory:     2097152 KiB
//	Used memory:    2097152 KiB
//	Persistent:     yes
//	Autostart:      disable
//	Managed save:   no
func parseVirshDominfo(
	output []byte,
	d *Domain,
) {
	sc := bufio.NewScanner(strings.NewReader(string(output)))
	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		switch key {
		case "UUID":
			d.UUID = val
		case "CPU(s)":
			if n, err := strconv.Atoi(strings.Fields(val)[0]); err == nil {
				d.VCPUs = n
			}
		case "Max memory":
			d.MaxMemory = val
		case "Autostart":
			d.Autostart = val == "enable"
		}
	}
}
