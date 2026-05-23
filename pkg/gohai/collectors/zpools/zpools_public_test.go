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

package zpools_test

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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/zpools"
)

var (
	_ collector.Collector = (*zpools.Linux)(nil)
	_ collector.Collector = (*zpools.Darwin)(nil)
)

// zpoolExec returns a MockExecutor that canned-answers `zpool list ...`.
func zpoolExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "zpool", "list", "-H", "-o", "name,size,alloc,free,health,altroot").
		Return(out, err).
		AnyTimes()
	return m
}

type ZpoolsPublicTestSuite struct {
	suite.Suite
}

func TestZpoolsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ZpoolsPublicTestSuite))
}

func (s *ZpoolsPublicTestSuite) TestNew() {
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
			c := zpools.New()
			s.Equal("zpools", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*zpools.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*zpools.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *ZpoolsPublicTestSuite) TestCollect() {
	// Typical output from `zpool list -H -o name,size,alloc,free,health,altroot`
	onePool := []byte("tank\t1.82T\t672G\t1.17T\tONLINE\t-\n")
	twoPools := []byte(
		"data\t3.62T\t1.20T\t2.42T\tONLINE\t-\n" +
			"backup\t931G\t450G\t481G\tDEGRADED\t/mnt/alt\n",
	)
	altRootPool := []byte("tank\t1.82T\t672G\t1.17T\tONLINE\t/alternate\n")
	dashFields := []byte("pool1\t-\t-\t-\tOFFLINE\t-\n")
	malformedLine := []byte("notEnoughFields\tonly3\n")

	tests := []struct {
		name    string
		variant string
		exec    func(*testing.T) executor.Executor
		want    []zpools.Pool
	}{
		{
			name:    "linux: single pool all fields populated",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, onePool, nil) },
			want: []zpools.Pool{
				{Name: "tank", Size: "1.82T", Alloc: "672G", Free: "1.17T", Health: "ONLINE"},
			},
		},
		{
			name:    "linux: two pools returned in order",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, twoPools, nil) },
			want: []zpools.Pool{
				{Name: "data", Size: "3.62T", Alloc: "1.20T", Free: "2.42T", Health: "ONLINE"},
				{
					Name:    "backup",
					Size:    "931G",
					Alloc:   "450G",
					Free:    "481G",
					Health:  "DEGRADED",
					Altroot: "/mnt/alt",
				},
			},
		},
		{
			name:    "linux: dash fields sanitized to empty strings",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, dashFields, nil) },
			want: []zpools.Pool{
				{Name: "pool1", Health: "OFFLINE"},
			},
		},
		{
			name:    "linux: altroot present",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, altRootPool, nil) },
			want: []zpools.Pool{
				{
					Name:    "tank",
					Size:    "1.82T",
					Alloc:   "672G",
					Free:    "1.17T",
					Health:  "ONLINE",
					Altroot: "/alternate",
				},
			},
		},
		{
			name:    "linux: malformed line skipped",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, malformedLine, nil) },
			want:    []zpools.Pool{},
		},
		{
			name:    "linux: empty output returns empty list",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, []byte{}, nil) },
			want:    []zpools.Pool{},
		},
		{
			name:    "linux: zpool not installed returns empty list",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return zpoolExec(t, nil, errors.New("exec: zpool not found"))
			},
			want: []zpools.Pool{},
		},
		{
			name:    "linux: nil executor returns empty list",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			want:    []zpools.Pool{},
		},
		{
			name:    "darwin: single pool",
			variant: "darwin",
			exec:    func(t *testing.T) executor.Executor { return zpoolExec(t, onePool, nil) },
			want: []zpools.Pool{
				{Name: "tank", Size: "1.82T", Alloc: "672G", Free: "1.17T", Health: "ONLINE"},
			},
		},
		{
			name:    "darwin: zpool not installed returns empty list",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return zpoolExec(t, nil, errors.New("exec: zpool not found"))
			},
			want: []zpools.Pool{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c zpools.Collector
			switch tt.variant {
			case "linux":
				c = &zpools.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &zpools.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*zpools.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Pools)
		})
	}
}
