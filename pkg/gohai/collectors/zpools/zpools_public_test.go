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
	"fmt"
	osexec "os/exec"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/zpools"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*zpools.Linux)(nil)
	_ collector.Collector = (*zpools.Darwin)(nil)
)

// The command lines that the rows answer, as sourcetest.Commands keys them.
const (
	zpoolLine = "zpool list -H -o name,size,alloc,free,health,altroot"
)

type ZpoolsPublicTestSuite struct {
	suite.Suite
}

func TestZpoolsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ZpoolsPublicTestSuite))
}

func (s *ZpoolsPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(zpools.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c zpools.Collector) {
				_, ok := c.(*zpools.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c zpools.Collector) {
				_, ok := c.(*zpools.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c zpools.Collector) {
				_, ok := c.(*zpools.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c zpools.Collector) {
				_, ok := c.(*zpools.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c zpools.Collector) {
				_, ok := c.(*zpools.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c zpools.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := zpools.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = zpools.New()
			}
			s.Equal("zpools", c.Name())
			s.Equal("linux", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
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
		name         string
		variant      string
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: single pool all fields populated",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{zpoolLine: string(onePool)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{
					{Name: "tank", Size: "1.82T", Alloc: "672G", Free: "1.17T", Health: "ONLINE"},
				}, info.Pools)
			},
		},
		{
			name:    "linux: two pools returned in order",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{zpoolLine: string(twoPools)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{
					{Name: "data", Size: "3.62T", Alloc: "1.20T", Free: "2.42T", Health: "ONLINE"},
					{
						Name:    "backup",
						Size:    "931G",
						Alloc:   "450G",
						Free:    "481G",
						Health:  "DEGRADED",
						Altroot: "/mnt/alt",
					},
				}, info.Pools)
			},
		},
		{
			name:    "linux: dash fields sanitized to empty strings",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{zpoolLine: string(dashFields)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{
					{Name: "pool1", Health: "OFFLINE"},
				}, info.Pools)
			},
		},
		{
			name:    "linux: altroot present",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{zpoolLine: string(altRootPool)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{
					{
						Name:    "tank",
						Size:    "1.82T",
						Alloc:   "672G",
						Free:    "1.17T",
						Health:  "ONLINE",
						Altroot: "/alternate",
					},
				}, info.Pools)
			},
		},
		{
			name:    "linux: malformed line skipped",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{zpoolLine: string(malformedLine)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{}, info.Pools)
			},
		},
		{
			name:    "linux: empty output returns empty list",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{zpoolLine: ""}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{}, info.Pools)
			},
		},
		{
			name:    "linux: zpool not installed returns empty list",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{zpoolLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{}, info.Pools)
			},
		},
		{
			name:    "linux: a failed run is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						zpoolLine: errors.New("exit status 1: permission denied"),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: nil executor returns empty list",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{}, info.Pools)
			},
		},
		{
			name:    "darwin: single pool",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{zpoolLine: string(onePool)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{
					{Name: "tank", Size: "1.82T", Alloc: "672G", Free: "1.17T", Health: "ONLINE"},
				}, info.Pools)
			},
		},
		{
			name:    "darwin: zpool not installed returns empty list",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{zpoolLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*zpools.Info)
				s.Require().True(ok)
				s.Equal([]zpools.Pool{}, info.Pools)
			},
		},
		{
			name:    "darwin: a failed run is an error",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						zpoolLine: errors.New("exit status 1: permission denied"),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Exactly(nil, got)
			},
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
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
