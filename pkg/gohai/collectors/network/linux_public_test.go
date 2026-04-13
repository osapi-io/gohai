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

package network_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	gpnet "github.com/shirou/gopsutil/v4/net"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
)

type NetworkLinuxPublicTestSuite struct {
	suite.Suite
}

func TestNetworkLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkLinuxPublicTestSuite))
}

// fsWith builds a memfs containing the given (path → contents) map.
func fsWith(
	t require.TestingT,
	files map[string]string,
) avfs.VFS {
	fs := memfs.New()
	for path, content := range files {
		require.NoError(t, fs.MkdirAll(dirOf(path), 0o755))
		require.NoError(t, fs.WriteFile(path, []byte(content), 0o644))
	}
	return fs
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "/"
}

// ipRouteExec returns a MockExecutor that canned-answers
// `ip -o -4 route show table main` and `ip -o -6 route show table main`.
// Any unmatched call returns an error (i.e., looks like ip is missing).
func ipRouteExec(
	t *testing.T,
	v4 []byte, v4Err error,
	v6 []byte, v6Err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-4", "route", "show", "table", "main").
		Return(v4, v4Err).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-6", "route", "show", "table", "main").
		Return(v6, v6Err).
		AnyTimes()
	return m
}

func (s *NetworkLinuxPublicTestSuite) TestCollect() {
	canonicalInterfaces := func(context.Context) (gpnet.InterfaceStatList, error) {
		return gpnet.InterfaceStatList{
			{
				Name: "lo", MTU: 65536, Flags: []string{"up", "loopback"},
				Addrs: gpnet.InterfaceAddrList{{Addr: "127.0.0.1/8"}, {Addr: "::1/128"}},
			},
			{
				Name: "eth0", MTU: 1500, HardwareAddr: "02:42:ac:11:00:02",
				Flags: []string{"up", "broadcast", "multicast"},
				Addrs: gpnet.InterfaceAddrList{
					{Addr: "10.0.0.5/24"},
					{Addr: "fe80::1/64"},
					{Addr: "not-a-cidr"},
				},
			},
		}, nil
	}
	zeroCounters := func(context.Context, bool) ([]gpnet.IOCountersStat, error) {
		return nil, nil
	}

	tests := []struct {
		name       string
		ifsFn      func(context.Context) (gpnet.InterfaceStatList, error)
		countersFn func(context.Context, bool) ([]gpnet.IOCountersStat, error)
		fs         avfs.VFS
		exec       executor.Executor
		wantErr    bool
		validate   func(*network.Info)
	}{
		{
			name:       "addresses parsed into family/prefixlen/netmask/scope/broadcast",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/net/lo/type":   "772\n",
				"/sys/class/net/eth0/type": "1\n",
			}),
			exec: ipRouteExec(s.T(), nil, errors.New("nope"), nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Interfaces, 2)
				lo := i.Interfaces[0]
				s.Equal("Loopback", lo.Encapsulation)
				s.Require().Len(lo.Addresses, 2)
				s.Equal("inet", lo.Addresses[0].Family)
				s.Equal(8, lo.Addresses[0].Prefixlen)
				s.Equal("Host", lo.Addresses[0].Scope)
				s.Equal("255.0.0.0", lo.Addresses[0].Netmask)
				s.Equal("inet6", lo.Addresses[1].Family)

				eth0 := i.Interfaces[1]
				s.Equal("Ethernet", eth0.Encapsulation)
				// "not-a-cidr" silently skipped → 2 addresses parsed.
				s.Require().Len(eth0.Addresses, 2)
				s.Equal("Global", eth0.Addresses[0].Scope)
				s.Equal("255.255.255.0", eth0.Addresses[0].Netmask)
				s.Equal("10.0.0.255", eth0.Addresses[0].Broadcast)
				s.Equal("Link", eth0.Addresses[1].Scope)
			},
		},
		{
			name:       "ip route populates default v4 + v6, per-interface routes attach",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("default via 10.0.0.1 dev eth0 proto dhcp metric 100 \n"+
					"10.0.0.0/24 dev eth0 proto kernel scope link src 10.0.0.5\n"), nil,
				[]byte("::/0 via fe80::1 dev eth0 proto ra metric 1024\n"+
					"fe80::/64 dev eth0 proto kernel metric 256\n"), nil),
			validate: func(i *network.Info) {
				s.Equal("eth0", i.DefaultInterface)
				s.Equal("10.0.0.1", i.DefaultGateway)
				s.Equal("eth0", i.DefaultInet6Interface)
				s.Equal("fe80::1", i.DefaultInet6Gateway)
				s.Require().Len(i.Routes, 4)
				eth0 := i.Interfaces[1]
				s.Len(eth0.Routes, 4)
			},
		},
		{
			name:       "multipath route expands into one Route per nexthop",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte(
					"default proto static metric 100 \\\n\tnexthop via 10.0.0.1 dev eth0 weight 1 \\\n\tnexthop via 10.0.0.2 dev lo weight 1\n",
				),
				nil,
				nil,
				errors.New("nope"),
			),
			validate: func(i *network.Info) {
				s.Require().Len(i.Routes, 2)
				s.Equal("default", i.Routes[0].Destination)
				s.Equal("10.0.0.1", i.Routes[0].Gateway)
				s.Equal("eth0", i.Routes[0].Interface)
				s.Equal("static", i.Routes[0].Proto, "proto carried from prefix")
				s.Equal(100, i.Routes[0].Metric, "metric carried from prefix")
				s.Equal("10.0.0.2", i.Routes[1].Gateway)
				s.Equal("lo", i.Routes[1].Interface)
				// Default route picks the first nexthop's interface/gateway.
				s.Equal("eth0", i.DefaultInterface)
				s.Equal("10.0.0.1", i.DefaultGateway)
			},
		},
		{
			name:       "route with src but no dev: interface inferred from local address match",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				// 10.0.0.5 is owned by eth0 in canonicalInterfaces.
				[]byte("10.0.0.0/24 proto kernel scope link src 10.0.0.5\n"),
				nil, nil, errors.New("nope"),
			),
			validate: func(i *network.Info) {
				s.Require().Len(i.Routes, 1)
				s.Equal("eth0", i.Routes[0].Interface, "inferred from src")
			},
		},
		{
			name:       "route with src but no dev and no matching local address: interface stays empty",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte("10.0.0.0/24 proto kernel scope link src 1.2.3.4\n"),
				nil, nil, errors.New("nope"),
			),
			validate: func(i *network.Info) {
				s.Require().Len(i.Routes, 1)
				s.Empty(i.Routes[0].Interface)
			},
		},
		{
			name:       "ip missing: routing fields stay empty, interfaces still populated",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				nil,
				errors.New("not found"),
				nil,
				errors.New("not found"),
			),
			validate: func(i *network.Info) {
				s.Empty(i.Routes)
				s.Empty(i.DefaultInterface)
				s.Len(i.Interfaces, 2)
			},
		},
		{
			name: "openvz guest: venet0:0 addresses merge under venet0",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return gpnet.InterfaceStatList{
					{Name: "venet0", MTU: 1500},
					{
						Name: "venet0:0", MTU: 1500,
						Addrs: gpnet.InterfaceAddrList{{Addr: "203.0.113.5/32"}},
					},
				}, nil
			},
			countersFn: zeroCounters,
			fs: fsWith(s.T(), map[string]string{
				"/proc/vz/version": "",
			}),
			exec: ipRouteExec(s.T(), nil, errors.New("nope"), nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Interfaces, 1)
				s.Equal("venet0", i.Interfaces[0].Name)
				s.Require().Len(i.Interfaces[0].Addresses, 1)
				s.Equal("203.0.113.5", i.Interfaces[0].Addresses[0].Addr)
			},
		},
		{
			name: "openvz host (/proc/bc/0 present): no merge",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return gpnet.InterfaceStatList{
					{Name: "venet0:0", Addrs: gpnet.InterfaceAddrList{{Addr: "1.2.3.4/32"}}},
				}, nil
			},
			countersFn: zeroCounters,
			fs: fsWith(s.T(), map[string]string{
				"/proc/vz/version": "",
				"/proc/bc/0":       "",
			}),
			exec: ipRouteExec(s.T(), nil, errors.New("nope"), nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Interfaces, 1)
				s.Equal("venet0:0", i.Interfaces[0].Name)
			},
		},
		{
			name: "openvz alias without matching base: kept as-is",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return gpnet.InterfaceStatList{
					{Name: "venet0:0", Addrs: gpnet.InterfaceAddrList{{Addr: "1.2.3.4/32"}}},
				}, nil
			},
			countersFn: zeroCounters,
			fs: fsWith(s.T(), map[string]string{
				"/proc/vz/version": "",
			}),
			exec: ipRouteExec(s.T(), nil, errors.New("nope"), nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Interfaces, 1)
				s.Equal("venet0:0", i.Interfaces[0].Name)
			},
		},
		{
			name:       "encapsulation: invalid sysfs content skipped",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/net/lo/type":   "not-a-number\n",
				"/sys/class/net/eth0/type": "9999\n", // unknown ARPHRD
			}),
			exec: ipRouteExec(s.T(), nil, errors.New("nope"), nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Empty(i.Interfaces[0].Encapsulation)
				s.Empty(i.Interfaces[1].Encapsulation)
			},
		},
		{
			name: "ip route line with metric, src, scope, proto: all parsed",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return gpnet.InterfaceStatList{
					{Name: "eth0", Addrs: gpnet.InterfaceAddrList{{Addr: "10.0.0.5/24"}}},
				}, nil
			},
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("10.0.0.0/24 dev eth0 proto kernel scope link src 10.0.0.5 metric 100\n"),
				nil, nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Routes, 1)
				r := i.Routes[0]
				s.Equal("10.0.0.0/24", r.Destination)
				s.Equal("kernel", r.Proto)
				s.Equal("link", r.Scope)
				s.Equal("10.0.0.5", r.Source)
				s.Equal(100, r.Metric)
			},
		},
		{
			name:       "ip route line that's just whitespace skipped",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("   \n\n"),
				nil, nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Empty(i.Routes)
			},
		},
		{
			name:       "ip route metric with non-int: silently kept zero",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("default via 10.0.0.1 dev eth0 metric oops\n"),
				nil, nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Routes, 1)
				s.Zero(i.Routes[0].Metric)
			},
		},
		{
			name:       "ip route attach to unknown interface: skipped silently",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("default via 10.0.0.1 dev nonexistent0\n"),
				nil, nil, errors.New("nope")),
			validate: func(i *network.Info) {
				s.Require().Len(i.Routes, 1)
				s.Equal("nonexistent0", i.Routes[0].Interface)
			},
		},
		{
			name:       "nil FS and nil Exec: gopsutil base only",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         nil,
			exec:       nil,
			validate: func(i *network.Info) {
				s.Len(i.Interfaces, 2)
				s.Empty(i.Routes)
			},
		},
		{
			name: "gopsutil error propagated",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return nil, errors.New("interfaces failed")
			},
			countersFn: zeroCounters,
			fs:         fsWith(s.T(), nil),
			exec:       ipRouteExec(s.T(), nil, nil, nil, nil),
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer network.SetInterfacesFn(tt.ifsFn)()
			defer network.SetIOCountersFn(tt.countersFn)()
			c := &network.Linux{FS: tt.fs, Exec: tt.exec}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*network.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}

func (s *NetworkLinuxPublicTestSuite) TestReadInterfaces() {
	tests := []struct {
		name       string
		ifsFn      func(context.Context) (gpnet.InterfaceStatList, error)
		countersFn func(context.Context, bool) ([]gpnet.IOCountersStat, error)
		wantErr    bool
		wantLen    int
	}{
		{
			name: "interfaces + counters merged",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return gpnet.InterfaceStatList{
					{Name: "eth0", MTU: 1500, HardwareAddr: "02:42:ac:11:00:02"},
				}, nil
			},
			countersFn: func(context.Context, bool) ([]gpnet.IOCountersStat, error) {
				return []gpnet.IOCountersStat{{Name: "eth0", BytesSent: 100}}, nil
			},
			wantLen: 1,
		},
		{
			name: "gopsutil error wrapped and returned",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return nil, errors.New("interfaces failed")
			},
			countersFn: func(context.Context, bool) ([]gpnet.IOCountersStat, error) {
				return nil, nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer network.SetInterfacesFn(tt.ifsFn)()
			defer network.SetIOCountersFn(tt.countersFn)()
			ifs, err := network.ReadInterfaces(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Len(ifs, tt.wantLen)
		})
	}
}
