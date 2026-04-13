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

type NetworkDarwinPublicTestSuite struct {
	suite.Suite
}

func TestNetworkDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkDarwinPublicTestSuite))
}

func (s *NetworkDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		ifsFn      func(context.Context) (gpnet.InterfaceStatList, error)
		countersFn func(context.Context, bool) ([]gpnet.IOCountersStat, error)
		wantErr    bool
		wantLen    int
	}{
		{
			name: "en0 with MAC and global IPv4",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return gpnet.InterfaceStatList{
					{
						Name: "en0", MTU: 1500, HardwareAddr: "aa:bb:cc:dd:ee:ff",
						Flags: []string{"up"},
						Addrs: gpnet.InterfaceAddrList{{Addr: "192.168.1.5/24"}},
					},
				}, nil
			},
			countersFn: func(context.Context, bool) ([]gpnet.IOCountersStat, error) { return nil, nil },
			wantLen:    1,
		},
		{
			name: "gopsutil error propagated",
			ifsFn: func(context.Context) (gpnet.InterfaceStatList, error) {
				return nil, errors.New("net error")
			},
			countersFn: func(context.Context, bool) ([]gpnet.IOCountersStat, error) { return nil, nil },
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer network.SetInterfacesFn(tt.ifsFn)()
			defer network.SetIOCountersFn(tt.countersFn)()
			c := &network.Darwin{}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*network.Info)
			s.Require().True(ok)
			s.Len(info.Interfaces, tt.wantLen)
		})
	}
}
