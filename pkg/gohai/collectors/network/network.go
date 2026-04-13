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

// Package network reports network interfaces, structured per-address
// data (family / prefix / scope / netmask / broadcast), per-interface
// I/O counters, the routing table, and top-level default interface +
// gateway facts (v4 + v6). On Linux we additionally derive the
// canonical encapsulation name from sysfs ARPHRD types and merge
// OpenVZ `venet0:N` aliases under the primary `venet0` interface.
package network

import (
	"context"
	"net"
	"strconv"
	"strings"

	gpnet "github.com/shirou/gopsutil/v4/net"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds network interfaces plus top-level routing facts.
type Info struct {
	Interfaces            []Interface `json:"interfaces"`
	Routes                []Route     `json:"routes,omitempty"`
	DefaultInterface      string      `json:"default_interface,omitempty"`
	DefaultGateway        string      `json:"default_gateway,omitempty"`
	DefaultInet6Interface string      `json:"default_inet6_interface,omitempty"`
	DefaultInet6Gateway   string      `json:"default_inet6_gateway,omitempty"`
}

// Interface describes a single network interface.
type Interface struct {
	Name          string    `json:"name"`
	MTU           int       `json:"mtu"`
	HardwareAddr  string    `json:"hardware_addr,omitempty"` // OCSF: network_interface.mac
	Encapsulation string    `json:"encapsulation,omitempty"` // canonical: Ethernet / Loopback / PPP / SLIP / IPIP / 6to4
	Flags         []string  `json:"flags,omitempty"`
	Addresses     []Address `json:"addresses,omitempty"`
	Routes        []Route   `json:"routes,omitempty"`
	Counters      *Counters `json:"counters,omitempty"`
}

// Address represents a single IP bound to an interface, structured
// the way Ohai emits it: family, prefix length, netmask (IPv4),
// broadcast (IPv4), scope.
type Address struct {
	Addr      string `json:"addr"`
	Family    string `json:"family"`              // inet | inet6
	Prefixlen int    `json:"prefixlen"`           // 24, 64, ...
	Netmask   string `json:"netmask,omitempty"`   // IPv4 only
	Broadcast string `json:"broadcast,omitempty"` // IPv4 only
	Scope     string `json:"scope,omitempty"`     // Global | Link | Host | Site
}

// Route is one entry from the kernel routing table.
type Route struct {
	Destination string `json:"destination"`
	Family      string `json:"family"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Source      string `json:"source,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Proto       string `json:"proto,omitempty"`
	Metric      int    `json:"metric,omitempty"`
}

// Counters holds I/O counters for one interface.
type Counters struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Errin       uint64 `json:"errin,omitempty"`
	Errout      uint64 `json:"errout,omitempty"`
	Dropin      uint64 `json:"dropin,omitempty"`
	Dropout     uint64 `json:"dropout,omitempty"`
}

// Collector is the public interface every network variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "network" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the network variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// Package-level injection seams for gopsutil (kept private so
// importers don't transitively need gopsutil).
var (
	interfacesFn = gpnet.InterfacesWithContext
	ioCountersFn = gpnet.IOCountersWithContext
)

// readInterfaces is the production bridge to gopsutil. Enumerates
// interfaces, structures each address, and merges per-interface I/O
// counters (matched by name).
func readInterfaces(
	ctx context.Context,
) ([]Interface, error) {
	ifs, err := interfacesFn(ctx)
	if err != nil {
		return nil, err
	}
	counts, _ := ioCountersFn(ctx, true)
	countersByName := map[string]*Counters{}
	for _, c := range counts {
		countersByName[c.Name] = &Counters{
			BytesSent: c.BytesSent, BytesRecv: c.BytesRecv,
			PacketsSent: c.PacketsSent, PacketsRecv: c.PacketsRecv,
			Errin: c.Errin, Errout: c.Errout,
			Dropin: c.Dropin, Dropout: c.Dropout,
		}
	}
	out := make([]Interface, 0, len(ifs))
	for _, i := range ifs {
		item := Interface{
			Name:         i.Name,
			MTU:          i.MTU,
			HardwareAddr: i.HardwareAddr,
			Flags:        i.Flags,
		}
		for _, a := range i.Addrs {
			if addr, ok := parseAddress(a.Addr); ok {
				item.Addresses = append(item.Addresses, addr)
			}
		}
		if c, ok := countersByName[i.Name]; ok {
			item.Counters = c
		}
		out = append(out, item)
	}
	return out, nil
}

// parseAddress turns a CIDR string into a structured Address.
// gopsutil emits everything as `<ip>/<prefix>`; non-parseable input
// (defensive) returns ok=false and is skipped by the caller.
func parseAddress(
	cidr string,
) (Address, bool) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return Address{}, false
	}
	ones, _ := ipnet.Mask.Size()
	addr := Address{Addr: ip.String(), Prefixlen: ones}
	if v4 := ip.To4(); v4 != nil {
		addr.Family = "inet"
		addr.Netmask = net.IP(net.CIDRMask(ones, 32)).String()
		addr.Broadcast = ipv4Broadcast(v4, ones)
	} else {
		addr.Family = "inet6"
	}
	addr.Scope = scopeOf(ip)
	return addr, true
}

// ipv4Broadcast computes the broadcast address for the given IPv4 +
// prefix length.
func ipv4Broadcast(
	ip net.IP,
	prefixlen int,
) string {
	mask := net.CIDRMask(prefixlen, 32)
	bcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		bcast[i] = ip[i] | ^mask[i]
	}
	return bcast.String()
}

// scopeOf classifies an IP into Ohai's title-cased scope buckets.
// Multicast classifications are intentionally collapsed into the
// nearest single answer — Ohai uses Global for routable, Link for
// link-local-anything, Host for loopback. We don't emit Site (no
// caller needs IPv6 site-local distinction in 2026).
func scopeOf(
	ip net.IP,
) string {
	switch {
	case ip.IsLoopback():
		return "Host"
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return "Link"
	}
	return "Global"
}

// arphrdEncapsulation maps the ARPHRD_* integer (read from
// /sys/class/net/<iface>/type) to Ohai's canonical encapsulation
// name.
//
// Source: https://github.com/torvalds/linux/blob/master/include/uapi/linux/if_arp.h
var arphrdEncapsulation = map[int]string{
	1:   "Ethernet", // ARPHRD_ETHER
	24:  "PPP",      // ARPHRD_PPP (kernel uses 512 — keep both)
	512: "PPP",
	256: "SLIP", // ARPHRD_SLIP
	257: "VJSLIP",
	768: "IPIP", // ARPHRD_TUNNEL
	769: "6to4", // ARPHRD_TUNNEL6
	772: "Loopback",
}

// isOpenVZAlias reports whether name looks like `<base>:<n>` — the
// venet0 alias pattern OpenVZ guests use.
func isOpenVZAlias(
	name string,
) (string, bool) {
	if i := strings.Index(name, ":"); i > 0 {
		if _, err := strconv.Atoi(name[i+1:]); err == nil {
			return name[:i], true
		}
	}
	return "", false
}
