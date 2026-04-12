//go:build darwin

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

package network

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/net"
)

var (
	interfacesFn = net.InterfacesWithContext
	ioCountersFn = net.IOCountersWithContext
)

func collect(
	ctx context.Context,
) (any, error) {
	return collectFromGopsutil(ctx, interfacesFn, ioCountersFn)
}

func collectFromGopsutil(
	ctx context.Context,
	ifaceFn func(context.Context) (net.InterfaceStatList, error),
	ioFn func(context.Context, bool) ([]net.IOCountersStat, error),
) (any, error) {
	ifaces, err := ifaceFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("net.Interfaces: %w", err)
	}
	counters, err := ioFn(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("net.IOCounters: %w", err)
	}
	countersByName := make(map[string]net.IOCountersStat, len(counters))
	for _, c := range counters {
		countersByName[c.Name] = c
	}
	out := &Info{Interfaces: make([]Interface, 0, len(ifaces))}
	for _, ifs := range ifaces {
		i := Interface{
			Name:         ifs.Name,
			MTU:          ifs.MTU,
			HardwareAddr: ifs.HardwareAddr,
			Flags:        ifs.Flags,
		}
		for _, a := range ifs.Addrs {
			i.Addresses = append(i.Addresses, Address{Addr: a.Addr})
		}
		if c, ok := countersByName[ifs.Name]; ok {
			i.Counters = &Counters{
				BytesSent: c.BytesSent, BytesRecv: c.BytesRecv,
				PacketsSent: c.PacketsSent, PacketsRecv: c.PacketsRecv,
				Errin: c.Errin, Errout: c.Errout,
				Dropin: c.Dropin, Dropout: c.Dropout,
			}
		}
		out.Interfaces = append(out.Interfaces, i)
	}
	return out, nil
}
