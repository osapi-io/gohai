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

// Package network collects network interface and I/O counter data.
package network

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds network interface and I/O data.
type Info struct {
	Interfaces []Interface `json:"interfaces"`
}

// Interface describes a single network interface with its I/O counters.
type Interface struct {
	Name         string    `json:"name"`
	MTU          int       `json:"mtu"`
	HardwareAddr string    `json:"hardware_addr,omitempty"`
	Flags        []string  `json:"flags,omitempty"`
	Addresses    []Address `json:"addresses,omitempty"`
	Counters     *Counters `json:"counters,omitempty"`
}

// Address is an IP address on an interface.
type Address struct {
	Addr string `json:"addr"`
}

// Counters holds per-interface I/O counter values.
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

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new network Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "network".
func (c *Collector) Name() string {
	return "network"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers network facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
