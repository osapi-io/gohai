//go:build linux

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

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
)

type NetworkLinuxPublicTestSuite struct {
	suite.Suite
}

func TestNetworkLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkLinuxPublicTestSuite))
}

func (s *NetworkLinuxPublicTestSuite) TestCollectFromGopsutil() {
	ifacesOk := func(_ context.Context) (gnet.InterfaceStatList, error) {
		return gnet.InterfaceStatList{
			{
				Index:        1,
				Name:         "eth0",
				MTU:          1500,
				HardwareAddr: "aa:bb:cc:dd:ee:ff",
				Flags:        []string{"up"},
				Addrs:        []gnet.InterfaceAddr{{Addr: "10.0.0.5"}},
			},
			{
				Index: 2,
				Name:  "lo",
				MTU:   65536,
				Flags: []string{"up", "loopback"},
				Addrs: []gnet.InterfaceAddr{{Addr: "127.0.0.1"}},
			},
		}, nil
	}
	ioOk := func(_ context.Context, _ bool) ([]gnet.IOCountersStat, error) {
		return []gnet.IOCountersStat{
			{Name: "eth0", BytesSent: 1000, BytesRecv: 2000, PacketsSent: 10, PacketsRecv: 20},
		}, nil
	}

	tests := []struct {
		name    string
		ifaceFn func(context.Context) (gnet.InterfaceStatList, error)
		ioFn    func(context.Context, bool) ([]gnet.IOCountersStat, error)
		wantErr bool
		wantLen int
	}{
		{"happy path with counters", ifacesOk, ioOk, false, 2},
		{
			"interfaces error",
			func(_ context.Context) (gnet.InterfaceStatList, error) { return nil, errors.New("boom") },
			ioOk,
			true,
			0,
		},
		{
			"io counters error",
			ifacesOk,
			func(_ context.Context, _ bool) ([]gnet.IOCountersStat, error) { return nil, errors.New("boom") },
			true,
			0,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := network.CollectFromGopsutil(context.Background(), tt.ifaceFn, tt.ioFn)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*network.Info)
			s.Require().True(ok)
			s.Len(info.Interfaces, tt.wantLen)
			if tt.wantLen > 0 {
				s.NotNil(info.Interfaces[0].Counters) // eth0 has counters
				s.Nil(info.Interfaces[1].Counters)    // lo doesn't
			}
		})
	}
}

func (s *NetworkLinuxPublicTestSuite) TestCollectDefault() {
	got, err := network.Collect(context.Background())
	s.Require().NoError(err)
	_, ok := got.(*network.Info)
	s.True(ok)
}
