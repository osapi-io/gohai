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

// Package network collects network interface data — MTU, MAC, addresses,
// I/O counters.
//
// Known limitation vs. Ohai: current shape is interfaces-only. Ohai
// additionally exposes default_interface/default_gateway, routes, and
// enriched per-address data (family/prefixlen/netmask/scope). Those
// are tracked as follow-ups.
package network

import (
	"context"

	"github.com/shirou/gopsutil/v4/net"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds network interface data and per-interface I/O counters.
type Info struct {
	Interfaces []Interface `json:"interfaces"`
}

// Interface describes a single network interface.
type Interface struct {
	Name         string    `json:"name"`
	MTU          int       `json:"mtu"`
	HardwareAddr string    `json:"hardware_addr,omitempty"` // OCSF: network_interface.mac
	Flags        []string  `json:"flags,omitempty"`
	Addresses    []Address `json:"addresses,omitempty"`
	Counters     *Counters `json:"counters,omitempty"`
}

// Address represents a single IP bound to an interface. Currently just
// the CIDR string from gopsutil; future extension will split into
// family/prefixlen/netmask/scope fields to match Ohai/OCSF.
type Address struct {
	Addr string `json:"addr"`
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

// readInterfaces is the production bridge to gopsutil. Enumerates
// interfaces and merges per-interface I/O counters (matched by name).
func readInterfaces(
	ctx context.Context,
) ([]Interface, error) {
	ifs, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	counts, _ := net.IOCountersWithContext(ctx, true)
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
			item.Addresses = append(item.Addresses, Address{Addr: a.Addr})
		}
		if c, ok := countersByName[i.Name]; ok {
			item.Counters = c
		}
		out = append(out, item)
	}
	return out, nil
}
