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
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

// errAbsent is the error of a command that is not on the host.
var errAbsent = fmt.Errorf("exec: %w", exec.ErrNotFound)

var (
	_ collector.Collector = (*network.Linux)(nil)
	_ collector.Collector = (*network.Darwin)(nil)
)

type NetworkPublicTestSuite struct {
	suite.Suite
}

func TestNetworkPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkPublicTestSuite))
}

func (s *NetworkPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(network.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c network.Collector) {
				_, ok := c.(*network.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c network.Collector) {
				_, ok := c.(*network.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c network.Collector) {
				_, ok := c.(*network.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c network.Collector) {
				_, ok := c.(*network.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c network.Collector) {
				_, ok := c.(*network.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c network.Collector) {
				// New picks the variant for the host the test runs on.
				_, darwin := c.(*network.Darwin)
				s.Equal(runtime.GOOS == "darwin", darwin)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := network.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = network.New()
			}
			s.Equal("network", c.Name())
			s.Equal("network", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

// ipRouteExec returns a MockRunner that canned-answers
// `ip -o -4 route show table main` and `ip -o -6 route show table main`.
// Any unmatched call returns an error (i.e., looks like ip is missing).
func ipRouteExec(
	t *testing.T,
	v4 []byte, v4Err error,
	v6 []byte, v6Err error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-4", "route", "show", "table", "main").
		Return(v4, v4Err).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-6", "route", "show", "table", "main").
		Return(v6, v6Err).
		AnyTimes()
	// `ip -d link` is invoked by applyIPLinkExtras for vlan / tunnel /
	// xdp annotations. Default to "not found" so existing tests that
	// don't care about those signals keep passing.
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return(nil, errAbsent).
		AnyTimes()
	// ethtool calls default to "command not found", so existing tests
	// that don't care about ethtool keep passing. Tests that exercise
	// the ethtool path use ipRouteAndEthtoolExec / ipRouteAndEthtoolFullExec.
	// Two arities to cover -i (one flag) and -g/-l/-c/-k/-a (all single flag).
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		Return(nil, errors.New("ethtool: not found")).
		AnyTimes()
	return m
}

// ipRouteAndEthtoolFullExec extends ipRouteAndEthtoolExec with canned
// outputs for the full ethtool tuning surface (-i / -g / -l / -c /
// -k / -a). Each map keyed interface name → canned stdout for the
// corresponding subcommand. Missing entries error out (looks like
// the kernel doesn't support that query for that device).
func ipRouteAndEthtoolFullExec(
	t *testing.T,
	driverInfo, ring, channel, coalesce, offload, pause map[string]string,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-4", "route", "show", "table", "main").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-6", "route", "show", "table", "main").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) < 2 {
				return nil, errors.New("usage: ethtool -<flag> <iface>")
			}
			flag, iface := args[0], args[1]
			var src map[string]string
			switch flag {
			case "-i":
				src = driverInfo
			case "-g":
				src = ring
			case "-l":
				src = channel
			case "-c":
				src = coalesce
			case "-k":
				src = offload
			case "-a":
				src = pause
			default:
				return nil, errors.New("ethtool: unsupported flag")
			}
			out, ok := src[iface]
			if !ok {
				return nil, errors.New(
					"exec ethtool: exit status 94: Cannot get driver information: Operation not supported",
				)
			}
			return []byte(out), nil
		}).
		AnyTimes()
	return m
}

// ipRouteAndIPLinkExec returns a mock that returns `out` for the
// `ip -d link` invocation and routes-not-found / ethtool-not-found
// for everything else. Used by tests exercising the vlan / tunnel /
// xdp annotation path.
func ipRouteAndIPLinkExec(
	t *testing.T,
	ipLinkOut string,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-4", "route", "show", "table", "main").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-6", "route", "show", "table", "main").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return([]byte(ipLinkOut), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		Return(nil, errors.New("ethtool: not found")).
		AnyTimes()
	return m
}

// ethtoolOnlyFor answers `ethtool -i` for name, and fails the test for an
// ethtool run on any other interface.
func ethtoolOnlyFor(
	t *testing.T,
	name string,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if args[1] != name {
				t.Errorf("ethtool ran for %s", args[1])
			}
			if args[0] == "-i" {
				return []byte("driver: igb\n"), nil
			}
			return nil, errors.New(
				"exec ethtool: exit status 94: Cannot get device settings: Operation not supported",
			)
		}).
		AnyTimes()
	return m
}

// driverInfoOnlyFor answers `ethtool -i name` with driver, and fails the
// test for any other ethtool run: a bond or a bridge gets no tuning runs.
func driverInfoOnlyFor(
	t *testing.T,
	name string,
	driver string,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if args[0] != "-i" || args[1] != name {
				t.Errorf("ethtool %s ran for %s", args[0], args[1])
			}
			return []byte("driver: " + driver + "\n"), nil
		}).
		AnyTimes()
	return m
}

// noEthtool answers the ip reads and fails the test for any ethtool run.
func noEthtool(
	t *testing.T,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		Times(0)
	return m
}

// ipRouteAndEthtoolExec extends ipRouteExec with a canned `ethtool -i
// <iface>` response. driverInfo maps interface name → ethtool stdout;
// missing interfaces fall back to the "not found" error. Other
// ethtool subcommands (-g/-l/-c/-k/-a) all return errors so the
// tuning enrichment cleanly skips when this helper is in use.
func ipRouteAndEthtoolExec(
	t *testing.T,
	driverInfo map[string]string,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-4", "route", "show", "table", "main").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-6", "route", "show", "table", "main").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-d", "link").
		Return(nil, errAbsent).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) < 2 || args[0] != "-i" {
				return nil, errors.New(
					"exec ethtool: exit status 94: Cannot get ring parameters: Operation not supported",
				)
			}
			out, ok := driverInfo[args[1]]
			if !ok {
				return nil, errors.New(
					"exec ethtool: exit status 1: Cannot get driver information: Operation not supported",
				)
			}
			return []byte(out), nil
		}).
		AnyTimes()
	return m
}

// TestSources covers the reads that the TestCollect fixtures do not
// reach: the edges of sysfs, /proc/net/dev, ip and the macOS commands.

func (s *NetworkPublicTestSuite) TestCollect() {
	canonicalInterfaces := func() ([]tIface, error) {
		return []tIface{
			{
				Index: 1,
				Name:  "lo", MTU: 65536, Flags: []string{"up", "loopback"},
				Addrs: []string{"127.0.0.1/8", "::1/128"},
			},
			{
				Index: 2,
				Name:  "eth0", MTU: 1500, MAC: "02:42:ac:11:00:02",
				Flags: []string{"up", "broadcast", "multicast"},
				Addrs: []string{"10.0.0.5/24", "fe80::1/64", "not-a-cidr"},
			},
		}, nil
	}
	adminDownInterfaces := func() ([]tIface, error) {
		return []tIface{
			{
				Index: 7,
				Name:  "eth7",
				MTU:   1500,
				Flags: []string{"broadcast"},
			}, // no "up" → admin-down
		}, nil
	}
	zeroCounters := func() ([]tCounters, error) {
		return nil, nil
	}

	tests := []struct {
		name       string
		variant    string
		ifsFn      func() ([]tIface, error)
		countersFn func() ([]tCounters, error)
		fs         avfs.VFS
		exec       source.Runner
		budget     time.Duration
		ctx        context.Context
		// files and runner give a row its host through the sources: the
		// sysfs and /proc files, and the commands. A row with a runner
		// does not use the interface seams above.
		files        map[string]string
		runner       *sourcetest.Commands
		nilExec      bool
		validateFunc func(any, error)
	}{
		{
			name:       "darwin: no Exec gives no fact",
			variant:    "darwin",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			nilExec:    true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:       "linux: addresses parsed into family/prefixlen/netmask/scope/broadcast",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/lo/type":   "772\n",
				"/sys/class/net/eth0/type": "1\n",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 2)
				lo := i.Interfaces[0]
				s.Equal("Loopback", lo.Encapsulation)
				s.Equal(1, lo.Number)
				s.Equal("up", lo.State)
				s.Require().Len(lo.Addresses, 2)
				s.Equal("inet", lo.Addresses[0].Family)
				s.Equal(8, lo.Addresses[0].Prefixlen)
				s.Equal("Host", lo.Addresses[0].Scope)
				s.Equal("255.0.0.0", lo.Addresses[0].Netmask)
				s.Equal("inet6", lo.Addresses[1].Family)

				eth0 := i.Interfaces[1]
				s.Equal("Ethernet", eth0.Encapsulation)
				s.Equal(2, eth0.Number)
				s.Equal("up", eth0.State)
				// "not-a-cidr" silently skipped → 2 addresses parsed.
				s.Require().Len(eth0.Addresses, 2)
				s.Equal("Global", eth0.Addresses[0].Scope)
				s.Equal("255.255.255.0", eth0.Addresses[0].Netmask)
				s.Equal("10.0.0.255", eth0.Addresses[0].Broadcast)
				s.Equal("Link", eth0.Addresses[1].Scope)
			},
		},
		{
			name:       "linux: admin-down interface reports state=down",
			ifsFn:      adminDownInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec:       ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal(7, i.Interfaces[0].Number)
				s.Equal("down", i.Interfaces[0].State)
			},
		},
		{
			name:       "linux: ethtool -i populates DriverInfo on Ethernet interfaces only",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/lo/type":            "772\n", // Loopback
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n", // Ethernet
			}),
			exec: ipRouteAndEthtoolExec(s.T(), map[string]string{
				"eth0": "driver: e1000e\n" +
					"version: 5.15.0-91-generic\n" +
					"firmware-version: 0.13-4\n" +
					"bus-info: 0000:00:19.0\n" +
					"supports-statistics: yes\n" +
					"supports-priv-flags: no\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 2)

				// lo is Loopback, so the ethtool path is skipped.
				s.Nil(i.Interfaces[0].Ethtool)

				// eth0 is Ethernet, so DriverInfo is populated.
				s.Require().NotNil(i.Interfaces[1].Ethtool)
				di := i.Interfaces[1].Ethtool.DriverInfo
				s.Equal("e1000e", di["driver"])
				s.Equal("5.15.0-91-generic", di["version"])
				// hyphens in source keys translated to underscores
				s.Equal("0.13-4", di["firmware_version"])
				s.Equal("0000:00:19.0", di["bus_info"])
				s.Equal("yes", di["supports_statistics"])
				s.Equal("no", di["supports_priv_flags"])
			},
		},
		{
			name: "linux: ethtool skips a virtual interface, which no device backs",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{
						Index: 2,
						Name:  "eth0", MTU: 1500, MAC: "02:42:ac:11:00:02",
						Flags: []string{"up", "broadcast", "multicast"},
					},
					{
						Index: 3,
						Name:  "veth0", MTU: 1500, MAC: "02:42:ac:11:00:03",
						Flags: []string{"up", "broadcast", "multicast"},
					},
				}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=igb\n",
				"/sys/class/net/veth0/type":         "1\n",
			}),
			exec: ethtoolOnlyFor(s.T(), "eth0"),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 2)
				s.Require().NotNil(i.Interfaces[0].Ethtool)
				s.Equal("igb", i.Interfaces[0].Ethtool.DriverInfo["driver"])
				s.Nil(i.Interfaces[1].Ethtool)
			},
		},
		{
			name: "linux: a bond gets ethtool -i, and no tuning run",
			ifsFn: func() ([]tIface, error) {
				return []tIface{{
					Index: 4,
					Name:  "bond0", MTU: 1500, MAC: "02:42:ac:11:00:04",
					Flags: []string{"up", "broadcast", "multicast"},
				}}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/bond0/type":         "1\n",
				"/sys/class/net/bond0/bonding/mode": "active-backup 1\n",
			}),
			exec: driverInfoOnlyFor(s.T(), "bond0", "bonding"),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Require().NotNil(i.Interfaces[0].Ethtool)
				s.Equal("bonding", i.Interfaces[0].Ethtool.DriverInfo["driver"])
				s.Nil(i.Interfaces[0].Ethtool.RingParams)
			},
		},
		{
			name: "linux: a bridge gets ethtool -i",
			ifsFn: func() ([]tIface, error) {
				return []tIface{{
					Index: 5,
					Name:  "br0", MTU: 1500, MAC: "02:42:ac:11:00:05",
					Flags: []string{"up", "broadcast", "multicast"},
				}}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/br0/type":             "1\n",
				"/sys/class/net/br0/bridge/stp_state": "0\n",
			}),
			exec: driverInfoOnlyFor(s.T(), "br0", "bridge"),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().NotNil(i.Interfaces[0].Ethtool)
				s.Equal("bridge", i.Interfaces[0].Ethtool.DriverInfo["driver"])
			},
		},
		{
			name:       "linux: a spent ethtool budget runs no ethtool",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec:   noEthtool(s.T()),
			budget: time.Nanosecond,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Nil(i.Interfaces[1].Ethtool)
			},
		},
		{
			name:       "linux: a budget whose deadline passed before its timer fired runs no ethtool",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: noEthtool(s.T()),
			ctx:  unfiredDeadline{context.Background()},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Nil(i.Interfaces[1].Ethtool)
			},
		},
		{
			name:       "linux: an ethtool query the driver doesn't support is skipped, Ethtool stays nil",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: ipRouteAndEthtoolExec(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 2)
				s.Nil(i.Interfaces[0].Ethtool)
				s.Nil(i.Interfaces[1].Ethtool)
			},
		},
		{
			name:       "linux: an ethtool run that fails for another reason is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: func() source.Runner {
				m := mocks.NewMockRunner(gomock.NewController(s.T()))
				m.EXPECT().
					Execute(gomock.Any(), "ip", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errAbsent).
					AnyTimes()
				m.EXPECT().
					Execute(gomock.Any(), "ethtool", "-i", "eth0").
					Return(nil, errors.New("exec ethtool: exit status 1: Cannot get driver information: Operation not permitted"))
				return m
			}(),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "ethtool -i eth0:")
				s.ErrorContains(err, "Operation not permitted")
			},
		},
		{
			name:       "linux: a context cancelled before ethtool is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: ipRouteAndEthtoolExec(s.T(), map[string]string{"eth0": "driver: e1000e\n"}),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorIs(err, context.Canceled)
				s.ErrorContains(err, "ethtool:")
			},
		},
		{
			name:       "linux: ethtool empty output yields nil DriverInfo",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: ipRouteAndEthtoolExec(s.T(), map[string]string{
				// whitespace-only lines + a colon-only line exercise both
				// the "no colon" continue and the "empty key" continue.
				"eth0": "\n   \n: only-value\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Nil(i.Interfaces[1].Ethtool)
			},
		},
		{
			name:       "linux: ethtool tuning bundle populates ring/channel/coalesce/offload/pause",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: ipRouteAndEthtoolFullExec(s.T(),
				map[string]string{"eth0": "driver: igb\n"},
				map[string]string{"eth0": "Ring parameters for eth0:\n" +
					"Pre-set maximums:\n" +
					"RX:\t\t4096\n" +
					"RX Mini:\t0\n" +
					"RX Jumbo:\t0\n" +
					"TX:\t\t4096\n" +
					"Current hardware settings:\n" +
					"RX:\t\t256\n" +
					"RX Mini:\t0\n" +
					"RX Jumbo:\t0\n" +
					"TX:\t\t256\n"},
				map[string]string{"eth0": "Channel parameters for eth0:\n" +
					"Pre-set maximums:\n" +
					"RX:\t\t0\n" +
					"TX:\t\t0\n" +
					"Other:\t\t1\n" +
					"Combined:\t8\n" +
					"Current hardware settings:\n" +
					"RX:\t\t0\n" +
					"TX:\t\t0\n" +
					"Other:\t\t1\n" +
					"Combined:\t4\n"},
				map[string]string{"eth0": "Coalesce parameters for eth0:\n" +
					"Adaptive RX: on  TX: off\n" +
					"rx-usecs: 50\n" +
					"tx-usecs: 8\n" +
					"tx-max-coalesced-frames: 64\n"},
				map[string]string{"eth0": "Features for eth0:\n" +
					"rx-checksumming: on\n" +
					"tx-checksumming: on [fixed]\n" +
					"generic-receive-offload: off\n"},
				map[string]string{"eth0": "Pause parameters for eth0:\n" +
					"Autonegotiate: on\n" +
					"RX: on\n" +
					"TX: off\n"}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				et := i.Interfaces[1].Ethtool
				s.Require().NotNil(et)

				// ring
				s.Equal(4096, et.RingParams["max_rx"])
				s.Equal(256, et.RingParams["current_rx"])
				s.Equal(0, et.RingParams["current_rx_mini"])

				// channel
				s.Equal(8, et.ChannelParams["max_combined"])
				s.Equal(4, et.ChannelParams["current_combined"])

				// coalesce: mixed types
				s.Equal("on", et.CoalesceParams["adaptive_rx"])
				s.Equal("off", et.CoalesceParams["adaptive_tx"])
				s.Equal(50, et.CoalesceParams["rx_usecs"])
				s.Equal(64, et.CoalesceParams["tx_max_coalesced_frames"])

				// offload: the [fixed] annotation is stripped
				s.Equal("on", et.OffloadParams["rx_checksumming"])
				s.Equal("on", et.OffloadParams["tx_checksumming"])
				s.Equal("off", et.OffloadParams["generic_receive_offload"])

				// pause: booleans
				s.True(et.PauseParams["autonegotiate"])
				s.True(et.PauseParams["rx"])
				s.False(et.PauseParams["tx"])

				// loopback got nothing
				s.Nil(i.Interfaces[0].Ethtool)
			},
		},
		{
			name:       "linux: ethtool tuning malformed lines silently skipped",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/eth0/type":          "1\n",
				"/sys/class/net/eth0/device/uevent": "DRIVER=e1000e\n",
			}),
			exec: ipRouteAndEthtoolFullExec(
				s.T(),
				nil, // no -i
				map[string]string{"eth0": "Ring parameters for eth0:\n" +
					"junk before any section\n" +
					"Pre-set maximums:\n" +
					"RX: not-a-number\n" +
					"RX Mini:   \n" + // empty value after colon
					"  \n" +
					"no-colon-here\n" +
					"RX:\t\t4096\n"},
				map[string]string{"eth0": "Channel parameters for eth0:\n"}, // no sections
				map[string]string{"eth0": "Coalesce parameters for eth0:\n" +
					"Adaptive bad-shape\n" + // adaptive parse fails (no TX:)
					"Adaptive no-rx-colon TX: off\n" + // adaptive: TX: present but RX side has no colon
					"missing-colon line\n" + // no colon at all
					"empty-value:   \n" + // empty value
					"non-numeric: not-int\n"}, // ParseInt fails
				map[string]string{"eth0": "Features for eth0:\n" +
					"no-colon-line\n" +
					"weird:    \n" + // value blank after strip
					"only-bracket: [fixed]\n"}, // value becomes empty after [ strip
				map[string]string{"eth0": "Pause parameters for eth0:\n" +
					"broken line\n" +
					"empty-value:   \n"}, // empty after trim
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				et := i.Interfaces[1].Ethtool
				// ring got the one valid RX after the bad lines
				s.Require().NotNil(et)
				s.Equal(4096, et.RingParams["max_rx"])
				// channel/coalesce/offload/pause: all empty → nil
				s.Nil(et.ChannelParams)
				s.Nil(et.CoalesceParams)
				s.Nil(et.OffloadParams)
				s.Nil(et.PauseParams)
			},
		},
		{
			name:       "linux: ip -d link reads the xdp mode from the header line",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(
				s.T(),
				`1: lo: <LOOPBACK,UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00 promiscuity 0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 xdp qdisc mq state UP mode DEFAULT group default qlen 1000
    link/ether 52:54:00:12:34:56 brd ff:ff:ff:ff:ff:ff promiscuity 0
    prog/xdp id 41 tag 3b185187f1855c4c jited
`,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				var eth0 *network.Interface
				for j := range i.Interfaces {
					if i.Interfaces[j].Name == "eth0" {
						eth0 = &i.Interfaces[j]
						break
					}
				}
				s.Require().NotNil(eth0)
				s.Require().NotNil(eth0.XDP)
				s.Require().Len(eth0.XDP.Attached, 1)
				// The bare `xdp` header token is native mode, xdpdrv,
				// and the `prog/xdp` line takes it.
				s.Equal("xdpdrv", eth0.XDP.Attached[0].Mode)
				s.Equal("41", eth0.XDP.Attached[0].ID)
				s.Equal("3b185187f1855c4c", eth0.XDP.Attached[0].Tag)
			},
		},
		{
			name:       "linux: ip -d link reads xdpmulti programs with their own modes",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(
				s.T(),
				`2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 xdpmulti qdisc mq state UP mode DEFAULT group default qlen 1000
    link/ether 52:54:00:12:34:56 brd ff:ff:ff:ff:ff:ff promiscuity 0
    prog/xdpgeneric id 17 tag 4d0e1f3a jited
    prog/xdpdrv id 18 tag cafebabe jited
`,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				var eth0 *network.Interface
				for j := range i.Interfaces {
					if i.Interfaces[j].Name == "eth0" {
						eth0 = &i.Interfaces[j]
						break
					}
				}
				s.Require().NotNil(eth0)
				s.Require().NotNil(eth0.XDP)
				s.Require().Len(eth0.XDP.Attached, 2)
				s.Equal("xdpgeneric", eth0.XDP.Attached[0].Mode)
				s.Equal("17", eth0.XDP.Attached[0].ID)
				s.Equal("xdpdrv", eth0.XDP.Attached[1].Mode)
				s.Equal("18", eth0.XDP.Attached[1].ID)
			},
		},
		{
			name:       "linux: ip -d link keeps a prog line that has no xdp header token",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(
				s.T(),
				`2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP
    link/ether 52:54:00:12:34:56 brd ff:ff:ff:ff:ff:ff
    prog/xdpgeneric id 17 tag 4d0e1f3a
`,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				var eth0 *network.Interface
				for j := range i.Interfaces {
					if i.Interfaces[j].Name == "eth0" {
						eth0 = &i.Interfaces[j]
						break
					}
				}
				s.Require().NotNil(eth0)
				s.Require().NotNil(eth0.XDP)
				s.Require().Len(eth0.XDP.Attached, 1)
				s.Equal("xdpgeneric", eth0.XDP.Attached[0].Mode)
			},
		},
		{
			name: "linux: ip -d link annotates eth0 vlan + ip6tun0 tunnel when present in sysfs",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Index: 1, Name: "eth0.100", MTU: 1500, Flags: []string{"up"}},
					{Index: 2, Name: "ip6tun0", MTU: 1452, Flags: []string{}},
				}, nil
			},
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(s.T(), `1: eth0.100@eth0: <BROADCAST,MULTICAST,UP> mtu 1500
    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff
    vlan id 100 LOOSE_BINDING REORDER_HDR
2: ip6tun0: <NOARP> mtu 1452 qdisc noop
    link/tunnel6 :: brd ::
    ip6tnl any remote 2001:db8::1 local 2001:db8::2 encaplimit 4 hoplimit 64 tclass 0x00 flowlabel 0x00000 addrgenmode eui64 numtxqueues 1 numrxqueues 1 gso_max_size 65536 gso_max_segs 65535 external
3: xdp_only: <BROADCAST,UP> mtu 1500
    link/ether aa:bb:cc:dd:ee:ff brd ff:ff:ff:ff:ff:ff
    xdp xdpdrv/prog/xdpdrv id 42 tag deadbeef
`),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				var v, t *network.Interface
				for j := range i.Interfaces {
					switch i.Interfaces[j].Name {
					case "eth0.100":
						v = &i.Interfaces[j]
					case "ip6tun0":
						t = &i.Interfaces[j]
					}
				}
				s.Require().NotNil(v)
				s.Require().NotNil(v.VLAN)
				s.Equal("100", v.VLAN.ID)
				s.Empty(v.VLAN.Protocol) // "vlan id" form, no protocol qualifier
				s.Contains(v.VLAN.Flags, "LOOSE_BINDING")
				s.Contains(v.VLAN.Flags, "REORDER_HDR")

				s.Require().NotNil(t)
				s.Require().NotNil(t.TunnelInfo)
				s.Equal("any", t.TunnelInfo.Proto)
				s.Equal("2001:db8::1", t.TunnelInfo.Remote)
				s.Equal("2001:db8::2", t.TunnelInfo.Local)
				s.Equal("4", t.TunnelInfo.EncapLimit)
				s.Equal("64", t.TunnelInfo.HopLimit)
				s.Equal("0x00", t.TunnelInfo.TClass)
				s.Equal("0x00000", t.TunnelInfo.Flowlabel)
				s.Equal("eui64", t.TunnelInfo.AddrGenMode)
				s.Equal("1", t.TunnelInfo.NumTxQueues)
				s.Equal("1", t.TunnelInfo.NumRxQueues)
				s.Equal("65536", t.TunnelInfo.GsoMaxSize)
				s.Equal("65535", t.TunnelInfo.GsoMaxSegs)
				s.True(t.TunnelInfo.External)
			},
		},
		{
			name:       "linux: ip -d link error leaves vlan/tunnel/xdp nil",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				nil,
				errAbsent,
				nil,
				errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				for j := range i.Interfaces {
					s.Nil(i.Interfaces[j].VLAN)
					s.Nil(i.Interfaces[j].TunnelInfo)
					s.Nil(i.Interfaces[j].XDP)
				}
			},
		},
		{
			name:       "linux: ip -d link with no annotations matches no interfaces",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(s.T(), `1: lo: <LOOPBACK,UP> mtu 65536
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: eth0: <BROADCAST,MULTICAST,UP> mtu 1500
    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff
`),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				for j := range i.Interfaces {
					s.Nil(i.Interfaces[j].VLAN)
					s.Nil(i.Interfaces[j].TunnelInfo)
					s.Nil(i.Interfaces[j].XDP)
				}
			},
		},
		{
			name: "linux: ip -d link vlan protocol-qualified form + tunnel trailing key",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Index: 1, Name: "vlan10", MTU: 1500, Flags: []string{"up"}},
					{Index: 2, Name: "tun0", MTU: 1452, Flags: []string{}},
				}, nil
			},
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(s.T(), `1: vlan10@eth0: <UP> mtu 1500
    vlan protocol 802.1ad id 10
2: tun0: <NOARP> mtu 1452
    ipip ipip6 remote
`),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				var v, t *network.Interface
				for j := range i.Interfaces {
					switch i.Interfaces[j].Name {
					case "vlan10":
						v = &i.Interfaces[j]
					case "tun0":
						t = &i.Interfaces[j]
					}
				}
				s.Require().NotNil(v.VLAN)
				s.Equal("802.1ad", v.VLAN.Protocol)
				s.Equal("10", v.VLAN.ID)
				s.Empty(v.VLAN.Flags)

				// `remote` is the last word, with no value after it, so
				// the key is recognised but no field set. proto stays.
				s.Require().NotNil(t.TunnelInfo)
				s.Equal("ipip6", t.TunnelInfo.Proto)
				s.Empty(t.TunnelInfo.Remote)
			},
		},
		{
			name:       "linux: ip -d link continuation lines before any header are dropped",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(s.T(), `   vlan id 999
1: lo: <LOOPBACK,UP> mtu 65536
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
`),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				for j := range i.Interfaces {
					s.Nil(i.Interfaces[j].VLAN)
				}
			},
		},
		{
			name: "linux: ip -d link malformed vlan + tunnel + xdp lines silently skipped",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Index: 1, Name: "weird", MTU: 1500, Flags: []string{"up"}},
				}, nil
			},
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteAndIPLinkExec(s.T(), `1: weird: <UP> mtu 1500
    vlan
    vlan id no-digits-here
    ip6tnl
    xdp
`),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				// Bare keywords with no payload: parsers see substring matches but
				// extract nothing; per-iface annotations stay nil.
				s.Nil(i.Interfaces[0].VLAN)
				s.Nil(i.Interfaces[0].TunnelInfo)
				s.Nil(i.Interfaces[0].XDP)
			},
		},
		{
			name:       "ip route populates default v4 + v6, per-interface routes attach",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("default via 10.0.0.1 dev eth0 proto dhcp metric 100 \n"+
					"10.0.0.0/24 dev eth0 proto kernel scope link src 10.0.0.5\n"), nil,
				[]byte("::/0 via fe80::1 dev eth0 proto ra metric 1024\n"+
					"fe80::/64 dev eth0 proto kernel metric 256\n"), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
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
			name:       "the default with the lowest metric wins, and a blackhole default is not one",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("blackhole default metric 10\n"+
					"default via 10.0.0.9 dev lo metric 200\n"+
					"default via 10.0.0.1 dev eth0 metric 100\n"+
					"default via 10.0.0.2 dev lo metric 100\n"+
					"unreachable 192.0.2.0/24\n"), nil,
				nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Equal("eth0", i.DefaultInterface)
				s.Equal("10.0.0.1", i.DefaultGateway)
				s.Require().Len(i.Routes, 5)
				s.Equal("default", i.Routes[0].Destination)
				s.Equal("blackhole", i.Routes[0].Type)
				s.Equal("192.0.2.0/24", i.Routes[4].Destination)
				s.Equal("unreachable", i.Routes[4].Type)
				s.Empty(i.Routes[1].Type)
			},
		},
		{
			name:       "multipath route expands into one Route per nexthop",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte(
					"default proto static metric 100 \\\n\tnexthop via 10.0.0.1 dev eth0 weight 1 \\\n\tnexthop via 10.0.0.2 dev lo weight 1\n",
				),
				nil,
				nil,
				errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 2)
				s.Equal("default", i.Routes[0].Destination)
				s.Equal("10.0.0.1", i.Routes[0].Gateway)
				s.Equal("eth0", i.Routes[0].Interface)
				s.Equal("static", i.Routes[0].Proto)
				s.Equal(100, i.Routes[0].Metric)
				s.Equal("10.0.0.2", i.Routes[1].Gateway)
				s.Equal("lo", i.Routes[1].Interface)
				// Default route picks the first nexthop's interface/gateway.
				s.Equal("eth0", i.DefaultInterface)
				s.Equal("10.0.0.1", i.DefaultGateway)
			},
		},
		{
			name:       "a route line that starts with nexthop gives no route",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte(
					"default via 10.0.0.1 dev eth0\nnexthop via 10.0.0.2 dev eth0 weight 1\n",
				),
				nil,
				nil,
				errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 1)
				s.Equal("default", i.Routes[0].Destination)
				s.Equal("10.0.0.1", i.Routes[0].Gateway)
			},
		},
		{
			name:       "multipath splits on whole nexthop tokens and skips an empty one",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte(
					"default proto static nexthop nexthop via 10.0.0.1 dev nexthop0 weight 1\n",
				),
				nil,
				nil,
				errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 1)
				s.Equal("default", i.Routes[0].Destination)
				s.Equal("10.0.0.1", i.Routes[0].Gateway)
				s.Equal("nexthop0", i.Routes[0].Interface)
				s.Equal("static", i.Routes[0].Proto)
			},
		},
		{
			name:       "route with src but no dev: interface inferred from local address match",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				// 10.0.0.5 is owned by eth0 in canonicalInterfaces.
				[]byte("10.0.0.0/24 proto kernel scope link src 10.0.0.5\n"),
				nil, nil, errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 1)
				s.Equal("eth0", i.Routes[0].Interface)
			},
		},
		{
			name:       "route with src but no dev and no matching local address: interface stays empty",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte("10.0.0.0/24 proto kernel scope link src 1.2.3.4\n"),
				nil, nil, errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 1)
				s.Empty(i.Routes[0].Interface)
			},
		},
		{
			name:       "ip missing: routing fields stay empty, interfaces still populated",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				nil,
				errAbsent,
				nil,
				errAbsent,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Empty(i.Routes)
				s.Empty(i.DefaultInterface)
				s.Len(i.Interfaces, 2)
			},
		},
		{
			name: "openvz guest: venet0:0 addresses merge under venet0",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Name: "venet0", MTU: 1500},
					{
						Name: "venet0:0", MTU: 1500,
						Addrs: []string{"203.0.113.5/32"},
					},
				}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/vz/version": "",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("venet0", i.Interfaces[0].Name)
				s.Require().Len(i.Interfaces[0].Addresses, 1)
				s.Equal("203.0.113.5", i.Interfaces[0].Addresses[0].Addr)
			},
		},
		{
			name: "openvz guest: an alias after an earlier merged alias still merges into its base",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Name: "venet0:1", Addrs: []string{"203.0.113.1/32"}},
					{Name: "venet0", MTU: 1500},
					{Name: "venet1", MTU: 1500},
					{Name: "venet1:1", Addrs: []string{"203.0.113.2/32"}},
				}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/vz/version": "",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 2)
				s.Equal("venet0", i.Interfaces[0].Name)
				s.Require().Len(i.Interfaces[0].Addresses, 1)
				s.Equal("203.0.113.1", i.Interfaces[0].Addresses[0].Addr)
				s.Equal("venet1", i.Interfaces[1].Name)
				s.Require().Len(i.Interfaces[1].Addresses, 1)
				s.Equal("203.0.113.2", i.Interfaces[1].Addresses[0].Addr)
			},
		},
		{
			name: "openvz host (/proc/bc/0 present): no merge",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Name: "venet0:0", Addrs: []string{"1.2.3.4/32"}},
				}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/vz/version": "",
				"/proc/bc/0":       "",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("venet0:0", i.Interfaces[0].Name)
			},
		},
		{
			name: "openvz alias without matching base: kept as-is",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Name: "venet0:0", Addrs: []string{"1.2.3.4/32"}},
				}, nil
			},
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/vz/version": "",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("venet0:0", i.Interfaces[0].Name)
			},
		},
		{
			name:       "encapsulation: invalid sysfs content skipped",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/class/net/lo/type":   "not-a-number\n",
				"/sys/class/net/eth0/type": "9999\n", // unknown ARPHRD
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Empty(i.Interfaces[0].Encapsulation)
				s.Empty(i.Interfaces[1].Encapsulation)
			},
		},
		{
			name: "ip route line with metric, src, scope, proto: all parsed",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{Name: "eth0", Addrs: []string{"10.0.0.5/24"}},
				}, nil
			},
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("10.0.0.0/24 dev eth0 proto kernel scope link src 10.0.0.5 metric 100\n"),
				nil, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
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
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("   \n\n"),
				nil, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Empty(i.Routes)
			},
		},
		{
			name:       "ip route metric with non-int: silently kept zero",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("default via 10.0.0.1 dev eth0 metric oops\n"),
				nil, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 1)
				s.Zero(i.Routes[0].Metric)
			},
		},
		{
			name:       "ip route attach to unknown interface: skipped silently",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(s.T(),
				[]byte("default via 10.0.0.1 dev nonexistent0\n"),
				nil, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Routes, 1)
				s.Equal("nonexistent0", i.Routes[0].Interface)
			},
		},
		{
			name:       "nil Exec: the interfaces of sysfs, and no routes",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Len(i.Interfaces, 2)
				s.Empty(i.Routes)
				s.Empty(i.Neighbours)
			},
		},
		{
			name:       "nil FS: an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         nil,
			exec:       nil,
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "no filesystem")
			},
		},
		{
			name:       "sysfs speed, duplex and driver per interface",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.Links(s.T(), sourcetest.FS(s.T(), nil), map[string]string{
				"/sys/class/net/eth0/device/driver": "../../../../bus/pci/drivers/e1000e",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				eth0 := i.Interfaces[1]
				s.Equal("e1000e", eth0.Driver)
				s.Equal("1000Mb/s", eth0.Speed)
				s.Equal("Full", eth0.Duplex)
			},
		},
		{
			name:       "no sysfs speed or duplex: both empty, Driver from sysfs still works",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.Links(s.T(), sourcetest.FS(s.T(), nil), map[string]string{
				"/sys/class/net/eth0/device/driver": "../../../../bus/pci/drivers/virtio_net",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				eth0 := i.Interfaces[1]
				s.Equal("virtio_net", eth0.Driver)
				s.Empty(eth0.Speed)
				s.Empty(eth0.Duplex)
			},
		},
		{
			name:       "sysfs driver symlink without slash: target used as-is",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs: sourcetest.Links(s.T(), sourcetest.FS(s.T(), nil), map[string]string{
				"/sys/class/net/eth0/device/driver": "ixgbe",
			}),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Equal("ixgbe", i.Interfaces[1].Driver)
			},
		},
		{
			name:       "neighbour list populates Info.Neighbours",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec:       ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Neighbours, 2)
				s.Equal("10.0.0.1", i.Neighbours[0].Address)
				s.Equal("inet", i.Neighbours[0].Family)
				s.Equal("REACHABLE", i.Neighbours[0].State)
			},
		},
		{
			name:       "linux: a failed ip -o addr show is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: func() source.Runner {
				m := ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent).(*mocks.MockRunner)
				m.EXPECT().Execute(gomock.Any(), "ip", "-o", "addr", "show").
					Return(nil, errors.New("exit status 1")).AnyTimes()
				return m
			}(),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "ip -o addr show: exit status 1")
			},
		},
		{
			name:       "linux: a failed ip -4 route is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec:       ipRouteExec(s.T(), nil, errors.New("exit status 2"), nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "ip -o -4 route show: exit status 2")
			},
		},
		{
			name:       "linux: a failed ip -6 route with IPv6 on is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), map[string]string{"/proc/net/if_inet6": ""}),
			exec:       ipRouteExec(s.T(), nil, nil, nil, errors.New("exit status 2")),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "ip -o -6 route show: exit status 2")
			},
		},
		{
			name:       "linux: a failed ip -6 route with IPv6 off has no v6 routes",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: ipRouteExec(
				s.T(),
				[]byte("default via 10.0.0.1 dev eth0\n"), nil,
				nil, errors.New("exit status 2"),
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Equal("eth0", i.DefaultInterface)
				s.Empty(i.DefaultInet6Interface)
			},
		},
		{
			name:       "linux: a failed ip -d link is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: func() source.Runner {
				m := mocks.NewMockRunner(gomock.NewController(s.T()))
				m.EXPECT().
					Execute(gomock.Any(), "ip", "-o", gomock.Any(), "route", "show", "table", "main").
					Return(nil, errAbsent).
					AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "ip", "-d", "link").
					Return(nil, errors.New("exit status 1")).AnyTimes()
				m.EXPECT().Execute(gomock.Any(), "ethtool", gomock.Any(), gomock.Any()).
					Return(nil, errAbsent).AnyTimes()
				return m
			}(),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "ip -d link: exit status 1")
			},
		},
		{
			name:       "linux: a failed ip neigh show is an error",
			ifsFn:      canonicalInterfaces,
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec: func() source.Runner {
				m := ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent).(*mocks.MockRunner)
				m.EXPECT().Execute(gomock.Any(), "ip", "neigh", "show").
					Return(nil, errors.New("exit status 1")).AnyTimes()
				return m
			}(),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "ip neigh show: exit status 1")
			},
		},
		{
			name:  "linux: a /proc/net/dev that cannot be read is an error",
			ifsFn: canonicalInterfaces,
			countersFn: func() ([]tCounters, error) {
				return nil, nil
			},
			fs: func() avfs.VFS {
				f := sourcetest.FS(s.T(), nil)
				require.NoError(s.T(), f.MkdirAll("/proc/net/dev", 0o755))
				return f
			}(),
			exec: ipRouteExec(s.T(), nil, errAbsent, nil, errAbsent),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "read /proc/net/dev")
			},
		},
		{
			name: "linux: no /sys/class/net",
			ifsFn: func() ([]tIface, error) {
				return nil, errors.New("interfaces failed")
			},
			countersFn: zeroCounters,
			fs:         sourcetest.FS(s.T(), nil),
			exec:       ipRouteExec(s.T(), nil, nil, nil, nil),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: en0 with MAC and global IPv4",
			variant: "darwin",
			ifsFn: func() ([]tIface, error) {
				return []tIface{
					{
						Name: "en0", MTU: 1500, MAC: "aa:bb:cc:dd:ee:ff",
						Flags: []string{"up"},
						Addrs: []string{"192.168.1.5/24"},
					},
				}, nil
			},
			countersFn: zeroCounters,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("en0", i.Interfaces[0].Name)
			},
		},
		{
			name:    "darwin: ifconfig fails, and the error names the command",
			variant: "darwin",
			ifsFn: func() ([]tIface, error) {
				return nil, errors.New("net error")
			},
			countersFn: zeroCounters,
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "ifconfig -a -v: ")
			},
		},
		{
			name:    "linux: an unknown speed and duplex, a peer address, odd counters and neighbours",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/tun0/ifindex": "4\n",
				"/sys/class/net/tun0/speed":   "-1\n",
				"/sys/class/net/tun0/duplex":  "unknown\n",
				"/proc/net/dev": "Inter-|   Receive\n face |bytes\n" +
					"  tun0: 1 2\n" +
					"  tun0: 10 20 0 0 0 0 0 0 30 40 0 0 0 0 0 0\n",
			},
			runner: &sourcetest.Commands{Out: map[string]string{
				"ip -o addr show": "4: tun0    inet 10.8.0.1 peer 10.8.0.2/32 scope global tun0\n" +
					"4: tun0    link/none\n",
				"ip neigh show": "\nnot-an-ip dev eth0\n10.0.0.9 dev tun0 FAILED\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				tun := i.Interfaces[0]
				s.Equal("Unknown!", tun.Speed)
				s.Equal("Unknown!", tun.Duplex)
				s.Require().Len(tun.Addresses, 1)
				s.Equal("10.8.0.1", tun.Addresses[0].Addr)
				s.Equal(32, tun.Addresses[0].Prefixlen)
				s.Equal("10.8.0.2", tun.Addresses[0].Peer)
				s.Require().NotNil(tun.Counters)
				s.Equal(uint64(30), tun.Counters.BytesSent)
				s.Equal(uint64(10), tun.Counters.BytesRecv)
				s.Require().Len(i.Neighbours, 1)
				s.Equal("FAILED", i.Neighbours[0].State)
				s.Empty(i.Neighbours[0].MAC)
			},
		},
		{
			name:    "linux: the scope is the one that ip prints, also when an admin set it",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/eth0/ifindex": "2\n",
			},
			runner: &sourcetest.Commands{Out: map[string]string{
				"ip -o addr show": "2: eth0    inet 192.0.2.7/24 brd 192.0.2.255 scope link eth0\n" +
					"2: eth0    inet 198.51.100.7/24 scope weird eth0\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Require().Len(i.Interfaces[0].Addresses, 2)
				s.Equal("Link", i.Interfaces[0].Addresses[0].Scope)
				s.Equal("Global", i.Interfaces[0].Addresses[1].Scope)
				s.Empty(i.Interfaces[0].Addresses[0].Peer)
			},
		},
		{
			name:    "darwin: routes and defaults from netstat -rn",
			variant: "darwin",
			runner: &sourcetest.Commands{Out: map[string]string{
				"ifconfig -a -v": "en0: flags=8863<UP,BROADCAST,RUNNING,MULTICAST> mtu 1500 index 14\n" +
					"\tinet 192.168.0.40 netmask 0xffff0000\n",
				"netstat -rn": "Routing tables\n\nnot a table row before any table\n" +
					"Internet:\n" +
					"Destination        Gateway            Flags               Netif Expire\n" +
					"default            link#24            UCSIg               utun6\n" +
					"default            192.168.0.1        UGScg                 en0\n" +
					"default            192.168.0.254      UGScIg                en0\n" +
					"127                127.0.0.1          UCS                   lo0\n" +
					"short\n\n" +
					"Internet6:\n" +
					"Destination                             Gateway                                 Flags               Netif Expire\n" +
					"default                                 fe80::1a90:88ff:fe6b:f272%en0           UGcg                  en0\n" +
					"default                                 fe80::%utun0                            UGcIg               utun0\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				// The first default with a gateway.
				s.Equal("en0", i.DefaultInterface)
				s.Equal("192.168.0.1", i.DefaultGateway)
				s.Equal("en0", i.DefaultInet6Interface)
				// No zone.
				s.Equal("fe80::1a90:88ff:fe6b:f272", i.DefaultInet6Gateway)
				s.Require().Len(i.Routes, 6)
				s.Equal(network.Route{
					Destination: "127", Family: "inet", Gateway: "127.0.0.1", Interface: "lo0",
				}, i.Routes[3])
				s.Require().Len(i.Interfaces, 1)
				// The routes of en0 attach to it.
				s.Len(i.Interfaces[0].Routes, 3)
			},
		},
		{
			name:    "darwin: a netstat -ibdnW that fails is an error",
			variant: "darwin",
			runner: &sourcetest.Commands{
				Out: map[string]string{
					"ifconfig -a -v": "en0: flags=8863<UP,BROADCAST,RUNNING,MULTICAST> mtu 1500 index 14\n",
				},
				Err: map[string]error{"netstat -ibdnW": errors.New("exit status 1")},
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "netstat -ibdnW: exit status 1")
			},
		},
		{
			name:    "darwin: a netstat -rn that fails is an error",
			variant: "darwin",
			runner: &sourcetest.Commands{
				Out: map[string]string{
					"ifconfig -a -v": "en0: flags=8863<UP,BROADCAST,RUNNING,MULTICAST> mtu 1500 index 14\n",
				},
				Err: map[string]error{"netstat -rn": errors.New("exit status 1")},
			},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "netstat -rn: exit status 1")
			},
		},
		{
			name:    "darwin: ifconfig and netstat on a Mac",
			variant: "darwin",
			runner: &sourcetest.Commands{Out: map[string]string{
				"ifconfig -a -v": "\tbefore any interface\n" +
					"lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384 index 1\n" +
					"\tinet 127.0.0.1 netmask 0xff000000\n" +
					"\tinet6 ::1 prefixlen 128\n" +
					"\tinet6 fe80::1%lo0 prefixlen 64 scopeid 0x1\n" +
					"gif0: flags=8010<POINTOPOINT,MULTICAST> mtu 1280 index 2\n" +
					"\tinet 10.1.1.1 --> 10.1.1.2\n" +
					"\tinet6 fe80::2\n" +
					"\tinet bogus netmask 0xff000000\n" +
					"\tinet6 bogus prefixlen 64\n" +
					"\tstatus: active\n",
				"netstat -ibdnW": "Name  Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll Drop\n" +
					"lo0   16384 <Link#1>                        869107     0  169411755   869107     0  169411755     0   0\n" +
					"lo0   16384 127           127.0.0.1         869107     -  169411755   869107     -  169411755     -   -\n" +
					"gif0* 1280  <Link#2>                             1     2          3        4     5          6     0   7\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 2)
				lo, gif := i.Interfaces[0], i.Interfaces[1]
				s.Equal(1, lo.Number)
				s.Equal([]string{"up", "loopback", "multicast"}, lo.Flags)
				s.Require().Len(lo.Addresses, 3)
				s.Equal(8, lo.Addresses[0].Prefixlen)
				s.Equal("fe80::1", lo.Addresses[2].Addr)
				s.Equal(uint64(169411755), lo.Counters.BytesSent)
				s.Equal(2, gif.Number)
				s.Equal("down", gif.State)
				s.Equal([]string{"pointtopoint", "multicast"}, gif.Flags)
				s.Require().Len(gif.Addresses, 2)
				// No netmask.
				s.Equal(32, gif.Addresses[0].Prefixlen)
				// No prefixlen.
				s.Equal(128, gif.Addresses[1].Prefixlen)
				s.Equal(uint64(3), gif.Counters.BytesRecv)
				s.Equal(uint64(7), gif.Counters.Dropout)
			},
		},
		{
			name:    "linux kind: loopback",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/lo/type": "772\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("lo", i.Interfaces[0].Name)
				s.Equal("loopback", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a bridge",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/br0/bridge/bridge_id": "8000.x\n",
				"/sys/class/net/br0/type":             "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("br0", i.Interfaces[0].Name)
				s.Equal("bridge", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a bond",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/bond0/bonding/mode": "active-backup 1\n",
				"/sys/class/net/bond0/type":         "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("bond0", i.Interfaces[0].Name)
				s.Equal("bond", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: Wi-Fi, by its wireless directory",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/wlan0/wireless/link": "0\n",
				"/sys/class/net/wlan0/type":          "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("wlan0", i.Interfaces[0].Name)
				s.Equal("wifi", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: Wi-Fi, by its phy",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/wlp2s0/phy80211/name": "phy0\n",
				"/sys/class/net/wlp2s0/type":          "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("wlp2s0", i.Interfaces[0].Name)
				s.Equal("wifi", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a VLAN, by its uevent",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/eth0.10/uevent": "DEVTYPE=vlan\nINTERFACE=eth0.10\n",
				"/sys/class/net/eth0.10/type":   "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("eth0.10", i.Interfaces[0].Name)
				s.Equal("vlan", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a uevent with a DEVTYPE that has no kind, on a NIC",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/eth1/uevent":        "DEVTYPE=other\n",
				"/sys/class/net/eth1/device/vendor": "0x8086\n",
				"/sys/class/net/eth1/type":          "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("eth1", i.Interfaces[0].Name)
				s.Equal("ethernet", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a tun device",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/tun0/tun_flags": "0x1001\n",
				"/sys/class/net/tun0/type":      "512\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("tun0", i.Interfaces[0].Name)
				s.Equal("tun", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a NIC",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/eth0/device/vendor": "0x8086\n",
				"/sys/class/net/eth0/type":          "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("eth0", i.Interfaces[0].Name)
				s.Equal("ethernet", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: Ethernet that software makes, such as a dummy",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/dummy0/type": "1\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("dummy0", i.Interfaces[0].Name)
				s.Equal("virtual", i.Interfaces[0].Kind)
			},
		},
		{
			name:    "linux kind: a link that is not Ethernet",
			variant: "linux",
			files: map[string]string{
				"/sys/class/net/ppp0/type": "512\n",
			},
			runner: &sourcetest.Commands{},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*network.Info)
				s.Require().True(ok)
				s.Require().Len(i.Interfaces, 1)
				s.Equal("ppp0", i.Interfaces[0].Name)
				s.Equal("", i.Interfaces[0].Kind)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.runner != nil {
				var c network.Collector = &network.Darwin{Exec: tt.runner.Mock(s.T())}
				if tt.variant == "linux" {
					c = &network.Linux{
						FS:   sourcetest.FS(s.T(), tt.files),
						Exec: tt.runner.Mock(s.T()),
					}
				}
				tt.validateFunc(c.Collect(context.Background(), nil))

				return
			}
			ifs, ifsErr := tt.ifsFn()
			counters, _ := tt.countersFn()
			var c network.Collector
			switch tt.variant {
			case "darwin":
				r := sourcetest.Commands{Out: map[string]string{"ifconfig -a -v": ifconfigOf(ifs)}}
				if ifsErr != nil {
					r.Err = map[string]error{"ifconfig -a -v": ifsErr}
				}
				d := &network.Darwin{Exec: r.Mock(s.T())}
				if tt.nilExec {
					d.Exec = nil
				}
				c = d
			default:
				if tt.fs != nil && ifsErr == nil {
					writeSysfs(
						s.T(),
						tt.fs,
						ifs,
						counters,
						tt.name != "no sysfs speed or duplex: both empty, Driver from sysfs still works",
					)
				}
				var exec source.Runner
				if tt.exec != nil {
					// The row's mock answers route, link and ethtool; the
					// address and neighbour reads come from the row's
					// interfaces.
					mock := tt.exec.(*mocks.MockRunner)
					mock.EXPECT().
						Execute(gomock.Any(), "ip", "-o", "addr", "show").
						Return([]byte(ipAddrOf(ifs)), nil).
						AnyTimes()
					mock.EXPECT().
						Execute(gomock.Any(), "ip", "neigh", "show").
						Return([]byte("10.0.0.1 dev eth0 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n"+
							"fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:02 STALE\n"), nil).
						AnyTimes()
					exec = mock
				}
				l := &network.Linux{FS: tt.fs, Exec: exec}
				l.SetEthtoolBudget(tt.budget)
				c = l
			}
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			tt.validateFunc(c.Collect(ctx, nil))
		})
	}
}

// unfiredDeadline is a context whose deadline has passed, but whose timer
// has not fired: Err is still nil. That is the state that a 1ns budget is
// in until its timer runs on another goroutine.
type unfiredDeadline struct {
	context.Context
}

func (unfiredDeadline) Deadline() (time.Time, bool) {
	return time.Unix(0, 0), true
}

// tIface is an interface as a test row gives it; writeSysfs, ipAddrOf
// and ifconfigOf turn it into what each host shows.
type tIface struct {
	Index int
	Name  string
	MTU   int
	MAC   string
	Flags []string
	Addrs []string
}

// tCounters is an interface's row of /proc/net/dev.
type tCounters struct {
	Name                     string
	BytesSent, BytesRecv     uint64
	PacketsSent, PacketsRecv uint64
}

// iff is the IFF_* bit of each flag.
var iff = map[string]uint64{
	"up":           0x1,
	"broadcast":    0x2,
	"loopback":     0x8,
	"pointtopoint": 0x10,
	"multicast":    0x1000,
}

// writeSysfs writes each interface's directory under /sys/class/net, an
// eth0 speed and duplex when nic is true, and /proc/net/dev.
func writeSysfs(
	t *testing.T,
	fs avfs.VFS,
	ifs []tIface,
	counters []tCounters,
	nic bool,
) {
	write := func(path, content string) {
		require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, fs.WriteFile(path, []byte(content), 0o644))
	}
	require.NoError(t, fs.MkdirAll("/sys/class/net", 0o755))
	for _, i := range ifs {
		var bits uint64
		for _, f := range i.Flags {
			bits |= iff[f]
		}
		dir := "/sys/class/net/" + i.Name + "/"
		write(dir+"ifindex", fmt.Sprintf("%d\n", i.Index))
		write(dir+"mtu", fmt.Sprintf("%d\n", i.MTU))
		write(dir+"flags", fmt.Sprintf("0x%x\n", bits))
		write(dir+"address", i.MAC+"\n")
		if nic && i.Name == "eth0" {
			write(dir+"speed", "1000\n")
			write(dir+"duplex", "full\n")
		}
	}
	if len(counters) > 0 {
		dev := "Inter-|   Receive\n face |bytes\n"
		for _, c := range counters {
			dev += fmt.Sprintf("%s: %d %d 0 0 0 0 0 0 %d %d 0 0 0 0 0 0\n",
				c.Name, c.BytesRecv, c.PacketsRecv, c.BytesSent, c.PacketsSent)
		}
		write("/proc/net/dev", dev)
	}
}

// ipAddrOf is `ip -o addr show` for the interfaces.
func ipAddrOf(
	ifs []tIface,
) string {
	var out string
	for _, i := range ifs {
		for _, a := range i.Addrs {
			fam := "inet"
			if strings.Contains(a, ":") {
				fam = "inet6"
			}
			// The scope that ip prints: host for loopback, link for
			// link-local, global for the rest.
			scope := "global"
			if ip, _, err := net.ParseCIDR(a); err == nil {
				switch {
				case ip.IsLoopback():
					scope = "host"
				case ip.IsLinkLocalUnicast():
					scope = "link"
				}
			}
			out += fmt.Sprintf(
				"%d: %s    %s %s scope %s %s\n",
				i.Index,
				i.Name,
				fam,
				a,
				scope,
				i.Name,
			)
		}
	}

	return out
}

// ifconfigOf is `ifconfig -a -v` for the interfaces.
func ifconfigOf(
	ifs []tIface,
) string {
	var out string
	for _, i := range ifs {
		out += fmt.Sprintf("%s: flags=8863<%s> mtu %d index %d\n",
			i.Name, strings.ToUpper(strings.Join(i.Flags, ",")), i.MTU, i.Index)
		if i.MAC != "" {
			out += "\tether " + i.MAC + "\n"
		}
		for _, a := range i.Addrs {
			ip, prefix, _ := strings.Cut(a, "/")
			if strings.Contains(ip, ":") {
				out += "\tinet6 " + ip + " prefixlen " + prefix + "\n"
				continue
			}
			n, _ := strconv.Atoi(prefix)
			out += fmt.Sprintf("\tinet %s netmask 0x%08x\n", ip, ^uint32(0)<<(32-n))
		}
	}

	return out
}
