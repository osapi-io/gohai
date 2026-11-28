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

package tc_test

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/tc"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*tc.Linux)(nil)
	_ collector.Collector = (*tc.Darwin)(nil)
)

// tcOut is representative `tc -s qdisc show` output.
const tcOut = `qdisc noqueue 0: dev lo root refcnt 2
 Sent 0 bytes 0 pkt (dropped 0, overlimits 0 requeues 0)
qdisc fq_codel 0: dev eth0 root refcnt 2 limit 10240p flows 1024 quantum 1514
 Sent 4096 bytes 32 pkt
qdisc pfifo_fast 0: dev eth1 parent 1:1 bands 3 priomap 1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1
`

type TCPublicTestSuite struct {
	suite.Suite
}

func TestTCPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TCPublicTestSuite))
}

func tcExec(
	t *testing.T,
	out []byte,
	err error,
) source.Runner {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().Execute(gomock.Any(), "tc", "-s", "qdisc", "show").
		Return(out, err).AnyTimes()
	return m
}

func (s *TCPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(tc.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c tc.Collector) {
				_, ok := c.(*tc.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c tc.Collector) {
				_, ok := c.(*tc.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c tc.Collector) {
				_, ok := c.(*tc.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c tc.Collector) {
				_, ok := c.(*tc.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c tc.Collector) {
				_, ok := c.(*tc.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := tc.New()
			s.Equal("tc", c.Name())
			s.Equal("linux", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *TCPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		exec         source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: statistics and backlog of each qdisc",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc htb 1: dev eth0 root refcnt 2 r2q 10 default 0x30
 Sent 98765 bytes 432 pkt (dropped 7, overlimits 21 requeues 3)
 backlog 1514b 1p requeues 3
qdisc fq_codel 10: dev eth0 parent 1:10 limit 10240p flows 1024
 Sent 1024 bytes 8 pkt (dropped 0, overlimits 0 requeues 0)
 backlog 10Kb 7p requeues 0
  maxpacket 1514 drop_overlimit 0 new_flow_count 4 ecn_mark 0
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{
					{
						Name: "eth0",
						QDiscs: []tc.QDisc{
							{
								Kind:   "htb",
								Handle: "1",
								Parent: "root",
								Stats: &tc.Stats{
									SentBytes:      98765,
									SentPackets:    432,
									Dropped:        7,
									Overlimits:     21,
									Requeues:       3,
									BacklogBytes:   1514,
									BacklogPackets: 1,
								},
							},
							{
								Kind:   "fq_codel",
								Handle: "10",
								Parent: "1:10",
								Stats: &tc.Stats{
									SentBytes:      1024,
									SentPackets:    8,
									BacklogBytes:   10 * 1024,
									BacklogPackets: 7,
								},
							},
						},
					},
				}, info.Interfaces)
			},
		},
		{
			name:    "linux: a sent count that is not a number is 0",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 Sent x bytes 2 pkt (dropped 1, overlimits 0 requeues 0)
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{SentPackets: 2, Dropped: 1}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: a backlog in Mb is in bytes",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog 2Mb 0p
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(
					&tc.Stats{BacklogBytes: 2 * 1024 * 1024},
					info.Interfaces[0].QDiscs[0].Stats,
				)
			},
		},
		{
			name:    "linux: a backlog in Gb is in bytes",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog 1Gb 3p
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(
					&tc.Stats{BacklogBytes: 1 << 30, BacklogPackets: 3},
					info.Interfaces[0].QDiscs[0].Stats,
				)
			},
		},
		{
			name:    "linux: backlog packets that are not a number are 0",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog 9b zp
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{BacklogBytes: 9}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: a backlog size with an unknown unit is 0",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog 5Tb 2p
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{BacklogPackets: 2}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: a negative backlog size is 0",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog -2b -1p
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: a backlog size with no unit is 0",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog 12 4p
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{BacklogPackets: 4}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: a backlog line with no size gives empty stats",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: a backlog line with no packets keeps its size",
			variant: "linux",
			exec: tcExec(s.T(), []byte(`qdisc pfifo 1: dev eth0 root
 backlog 9b
`), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Require().Len(info.Interfaces, 1)
				s.Require().Len(info.Interfaces[0].QDiscs, 1)
				s.Equal(&tc.Stats{BacklogBytes: 9}, info.Interfaces[0].QDiscs[0].Stats)
			},
		},
		{
			name:    "linux: tc returns qdisc data",
			variant: "linux",
			exec:    tcExec(s.T(), []byte(tcOut), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{
					{
						Name: "lo",
						QDiscs: []tc.QDisc{
							{
								Kind:   "noqueue",
								Handle: "0",
								Parent: "root",
								Stats:  &tc.Stats{},
							},
						},
					},
					{
						Name: "eth0",
						QDiscs: []tc.QDisc{
							{
								Kind:   "fq_codel",
								Handle: "0",
								Parent: "root",
								Stats:  &tc.Stats{SentBytes: 4096, SentPackets: 32},
							},
						},
					},
					{
						Name: "eth1",
						QDiscs: []tc.QDisc{
							{Kind: "pfifo_fast", Handle: "0", Parent: "1:1"},
						},
					},
				}, info.Interfaces)
			},
		},
		{
			name:    "linux: no tc on the host, empty list",
			variant: "linux",
			exec:    tcExec(s.T(), nil, fmt.Errorf("exec tc: %w", exec.ErrNotFound)),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{}, info.Interfaces)
			},
		},
		{
			name:    "linux: tc fails, the error is returned",
			variant: "linux",
			exec:    tcExec(s.T(), nil, errors.New("exit status 2")),
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "tc -s qdisc show: exit status 2")
			},
		},
		{
			name:    "linux: nil Exec returns empty list",
			variant: "linux",
			exec:    nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{}, info.Interfaces)
			},
		},
		{
			name:    "linux: empty output returns empty list",
			variant: "linux",
			exec:    tcExec(s.T(), []byte(""), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{}, info.Interfaces)
			},
		},
		{
			name:    "linux: short qdisc lines skipped",
			variant: "linux",
			exec:    tcExec(s.T(), []byte("qdisc noqueue 0:\n"), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{}, info.Interfaces)
			},
		},
		{
			name:    "linux: non-qdisc lines and blank lines skipped",
			variant: "linux",
			exec: tcExec(
				s.T(),
				[]byte("\n Sent 0 bytes\nqdisc fq_codel 0: dev eth0 root\n"),
				nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{
					{
						Name: "eth0",
						QDiscs: []tc.QDisc{
							{Kind: "fq_codel", Handle: "0", Parent: "root"},
						},
					},
				}, info.Interfaces)
			},
		},
		{
			name:    "linux: qdisc line without dev keyword skipped",
			variant: "linux",
			exec:    tcExec(s.T(), []byte("qdisc noqueue 0: root refcnt 2\n"), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface{}, info.Interfaces)
			},
		},
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Exactly(nil, got)
					return
				}
				info, ok := got.(*tc.Info)
				s.Require().True(ok)
				s.Equal([]tc.Interface(nil), info.Interfaces)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c tc.Collector
			switch tt.variant {
			case "linux":
				c = &tc.Linux{Exec: tt.exec}
			case "darwin":
				c = tc.NewDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
