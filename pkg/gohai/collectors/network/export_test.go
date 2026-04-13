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
	"net"

	"github.com/jaypipes/ghw"
	gpnet "github.com/shirou/gopsutil/v4/net"
	"github.com/vishvananda/netlink"
)

// ReadInterfaces exposes the private readInterfaces bridge to the
// external network_test package.
var ReadInterfaces = readInterfaces

// SetInterfacesFn swaps the private gopsutil net.InterfacesWithContext
// call backing readInterfaces. Returns a restore func the caller must
// defer.
func SetInterfacesFn(
	fn func(context.Context) (gpnet.InterfaceStatList, error),
) (restore func()) {
	orig := interfacesFn
	interfacesFn = fn
	return func() { interfacesFn = orig }
}

// SetIOCountersFn swaps the private gopsutil net.IOCountersWithContext
// call backing readInterfaces. Returns a restore func the caller must
// defer.
func SetIOCountersFn(
	fn func(context.Context, bool) ([]gpnet.IOCountersStat, error),
) (restore func()) {
	orig := ioCountersFn
	ioCountersFn = fn
	return func() { ioCountersFn = orig }
}

// SetNICFn swaps the ghw-backed link-detail probe (returns name →
// {Speed, Duplex} map).
func SetNICFn(
	fn func() (map[string]NICStat, error),
) (restore func()) {
	orig := nicFn
	nicFn = fn
	return func() { nicFn = orig }
}

// SetNeighListFn swaps the netlink-backed neighbour-list probe.
func SetNeighListFn(
	fn func() ([]Neighbour, error),
) (restore func()) {
	orig := neighListFn
	neighListFn = fn
	return func() { neighListFn = orig }
}

// NeighFamily / NeighState expose the private mapping helpers so
// tests can assert per-input outputs without going through netlink.
var (
	NeighFamily = neighFamily
	NeighState  = neighState
)

// NICMapFromGHW / NeighboursFromNetlink expose the pure conversion
// helpers so tests can exercise them without calling ghw / netlink.
var (
	NICMapFromGHW         = nicMapFromGHW
	NeighboursFromNetlink = neighboursFromNetlink
)

// ReadNIC / ReadNeighbours expose the production wrappers so tests
// can drive them with swapped upstream calls.
var (
	ReadNIC          = readNIC
	ReadNeighbours   = readNeighbours
	IndexToIfaceName = indexToInterfaceName
)

// SetGHWNetworkFn / SetNetlinkNeighListFn / SetNetInterfaceByIndex
// swap the upstream library calls for unit-testing readNIC /
// readNeighbours / indexToInterfaceName without touching the host.
func SetGHWNetworkFn(fn func(...any) (*ghw.NetworkInfo, error)) (restore func()) {
	orig := ghwNetworkFn
	ghwNetworkFn = fn
	return func() { ghwNetworkFn = orig }
}

func SetNetlinkNeighListFn(
	fn func(linkIndex, family int) ([]netlink.Neigh, error),
) (restore func()) {
	orig := netlinkNeighListFn
	netlinkNeighListFn = fn
	return func() { netlinkNeighListFn = orig }
}

func SetNetInterfaceByIndex(fn func(int) (*net.Interface, error)) (restore func()) {
	orig := netInterfaceByIndex
	netInterfaceByIndex = fn
	return func() { netInterfaceByIndex = orig }
}
