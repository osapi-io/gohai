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
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/tc"
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
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().Execute(gomock.Any(), "tc", "-s", "qdisc", "show").
		Return(out, err).AnyTimes()
	return m
}

func (s *TCPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"rhel dispatches to Linux", "rhel", "linux"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := tc.New()
			s.Equal("tc", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*tc.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*tc.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *TCPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		exec    executor.Executor
		wantNil bool
		want    []tc.Interface
	}{
		{
			name:    "linux: tc returns qdisc data",
			variant: "linux",
			exec:    tcExec(s.T(), []byte(tcOut), nil),
			want: []tc.Interface{
				{
					Name: "lo",
					QDiscs: []tc.QDisc{
						{Kind: "noqueue", Handle: "0"},
					},
				},
				{
					Name: "eth0",
					QDiscs: []tc.QDisc{
						{Kind: "fq_codel", Handle: "0"},
					},
				},
				{
					Name: "eth1",
					QDiscs: []tc.QDisc{
						{Kind: "pfifo_fast", Handle: "0", Parent: "1:1"},
					},
				},
			},
		},
		{
			name:    "linux: tc fails, empty list",
			variant: "linux",
			exec:    tcExec(s.T(), nil, errors.New("not found")),
			want:    []tc.Interface{},
		},
		{
			name:    "linux: nil Exec returns empty list",
			variant: "linux",
			exec:    nil,
			want:    []tc.Interface{},
		},
		{
			name:    "linux: empty output returns empty list",
			variant: "linux",
			exec:    tcExec(s.T(), []byte(""), nil),
			want:    []tc.Interface{},
		},
		{
			name:    "linux: short qdisc lines skipped",
			variant: "linux",
			exec:    tcExec(s.T(), []byte("qdisc noqueue 0:\n"), nil),
			want:    []tc.Interface{},
		},
		{
			name:    "linux: non-qdisc lines and blank lines skipped",
			variant: "linux",
			exec: tcExec(
				s.T(),
				[]byte("\n Sent 0 bytes\nqdisc fq_codel 0: dev eth0 root\n"),
				nil,
			),
			want: []tc.Interface{
				{
					Name: "eth0",
					QDiscs: []tc.QDisc{
						{Kind: "fq_codel", Handle: "0"},
					},
				},
			},
		},
		{
			name:    "linux: qdisc line without dev keyword skipped",
			variant: "linux",
			exec:    tcExec(s.T(), []byte("qdisc noqueue 0: root refcnt 2\n"), nil),
			want:    []tc.Interface{},
		},
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			wantNil: true,
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
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*tc.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Interfaces)
		})
	}
}
