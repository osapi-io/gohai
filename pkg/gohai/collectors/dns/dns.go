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

// Package dns collects the host's resolver configuration: the name
// servers, search domains and options of /etc/resolv.conf, and the name
// servers and domains of each interface where the host keeps them apart
// (systemd-resolved on Linux, the scoped resolvers of macOS).
package dns

import (
	"fmt"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info is the resolver configuration.
type Info struct {
	Nameservers []string `json:"nameservers,omitempty"` // resolv.conf nameserver lines
	Search      []string `json:"search,omitempty"`      // resolv.conf search (or domain)
	Options     []string `json:"options,omitempty"`     // resolv.conf options
	// Interfaces is the resolver of each interface that has one, by
	// interface name.
	Interfaces map[string]Resolver `json:"interfaces,omitempty"`
	// Global is systemd-resolved's resolver that belongs to no interface
	// (resolvectl's Global line). With systemd-resolved, resolv.conf names
	// only its stub, 127.0.0.53, so these are the upstream servers.
	Global *Resolver `json:"global,omitempty"`
}

// Resolver is the name servers and domains of one interface.
type Resolver struct {
	Nameservers []string `json:"nameservers,omitempty"`
	Domains     []string `json:"domains,omitempty"`
}

// Collector is the interface every dns variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

// Name is the collector's name.
func (base) Name() string { return "dns" }

// Category is the collector's category.
func (base) Category() collector.Category { return collector.CategoryNetwork }

// DefaultEnabled is false: most callers do not need it.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the variant for the real host.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src. A zero field of src is the real host's.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// resolvConf is the classic resolver file, on Linux and on macOS.
const resolvConf = "/etc/resolv.conf"

// readResolvConf returns an Info with the resolv.conf fields of fsys. A
// host with no resolv.conf gives an empty Info, and any other read
// error is returned.
func readResolvConf(
	fsys avfs.VFS,
) (*Info, error) {
	info := &Info{}
	b, err := fsys.ReadFile(resolvConf)
	if collector.Absent(err) {
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", resolvConf, err)
	}
	parseResolvConf(info, b)

	return info, nil
}

// parseResolvConf fills info from resolv.conf: nameserver lines, the
// last search or domain line, and options.
func parseResolvConf(
	info *Info,
	b []byte,
) {
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || strings.HasPrefix(f[0], "#") || strings.HasPrefix(f[0], ";") {
			continue
		}
		switch f[0] {
		case "nameserver":
			info.Nameservers = append(info.Nameservers, f[1])
		case "search", "domain":
			info.Search = f[1:]
		case "options":
			info.Options = append(info.Options, f[1:]...)
		}
	}
}

// setResolver adds values to the resolver of iface.
func setResolver(
	info *Info,
	iface string,
	nameservers []string,
	domains []string,
) {
	if info.Interfaces == nil {
		info.Interfaces = map[string]Resolver{}
	}
	r := info.Interfaces[iface]
	r.Nameservers = append(r.Nameservers, nameservers...)
	r.Domains = append(r.Domains, domains...)
	info.Interfaces[iface] = r
}
