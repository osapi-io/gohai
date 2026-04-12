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

	gpnet "github.com/shirou/gopsutil/v4/net"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
)

type NetworkLinuxPublicTestSuite struct {
	suite.Suite
}

func TestNetworkLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkLinuxPublicTestSuite))
}

func (s *NetworkLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		ifs     []network.Interface
		fnErr   error
		wantErr bool
		want    network.Info
	}{
		{
			name: "eth0 with addresses and counters",
			ifs: []network.Interface{
				{
					Name: "eth0", MTU: 1500, HardwareAddr: "02:42:ac:11:00:02",
					Flags:     []string{"up", "broadcast"},
					Addresses: []network.Address{{Addr: "10.0.0.5/24"}},
					Counters:  &network.Counters{BytesSent: 100, BytesRecv: 200},
				},
			},
			want: network.Info{Interfaces: []network.Interface{
				{
					Name: "eth0", MTU: 1500, HardwareAddr: "02:42:ac:11:00:02",
					Flags:     []string{"up", "broadcast"},
					Addresses: []network.Address{{Addr: "10.0.0.5/24"}},
					Counters:  &network.Counters{BytesSent: 100, BytesRecv: 200},
				},
			}},
		},
		{
			name:    "gopsutil error wrapped and returned",
			fnErr:   errors.New("net interfaces error"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &network.Linux{
				InterfacesFn: func(context.Context) ([]network.Interface, error) {
					if tt.fnErr != nil {
						return nil, tt.fnErr
					}
					return tt.ifs, nil
				},
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*network.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *NetworkLinuxPublicTestSuite) TestReadInterfaces() {
	okIfs := gpnet.InterfaceStatList{
		{
			Name:         "eth0",
			MTU:          1500,
			HardwareAddr: "02:42:ac:11:00:02",
			Flags:        []string{"up"},
			Addrs:        gpnet.InterfaceAddrList{{Addr: "10.0.0.5/24"}},
		},
	}

	tests := []struct {
		name        string
		ifsFn       func(context.Context) (gpnet.InterfaceStatList, error)
		countersFn  func(context.Context, bool) ([]gpnet.IOCountersStat, error)
		wantErr     bool
		wantLen     int
		wantCounter bool
	}{
		{
			name:  "interfaces + counters merged",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) { return okIfs, nil },
			countersFn: func(context.Context, bool) ([]gpnet.IOCountersStat, error) {
				return []gpnet.IOCountersStat{{Name: "eth0", BytesSent: 100, BytesRecv: 200}}, nil
			},
			wantLen:     1,
			wantCounter: true,
		},
		{
			name:        "counters error tolerated",
			ifsFn:       func(context.Context) (gpnet.InterfaceStatList, error) { return okIfs, nil },
			countersFn:  func(context.Context, bool) ([]gpnet.IOCountersStat, error) { return nil, errors.New("counters failed") },
			wantLen:     1,
			wantCounter: false,
		},
		{
			name:       "interfaces error propagated",
			ifsFn:      func(context.Context) (gpnet.InterfaceStatList, error) { return nil, errors.New("interfaces failed") },
			countersFn: func(context.Context, bool) ([]gpnet.IOCountersStat, error) { return nil, nil },
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreI := network.SetInterfacesFn(tt.ifsFn)
			defer restoreI()
			restoreC := network.SetIOCountersFn(tt.countersFn)
			defer restoreC()
			got, err := network.ReadInterfaces(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Len(got, tt.wantLen)
			if tt.wantCounter {
				s.NotNil(got[0].Counters)
			} else {
				s.Nil(got[0].Counters)
			}
		})
	}
}
