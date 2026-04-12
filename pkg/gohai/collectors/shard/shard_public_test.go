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

package shard_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shard"
)

var (
	_ collector.Collector = (*shard.Linux)(nil)
	_ collector.Collector = (*shard.Darwin)(nil)
)

type ShardPublicTestSuite struct {
	suite.Suite
}

func TestShardPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShardPublicTestSuite))
}

func (s *ShardPublicTestSuite) TestNew() {
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
			c := shard.New()
			s.Equal("shard", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*shard.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*shard.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *ShardPublicTestSuite) TestCollectLinux() {
	tests := []struct {
		name  string
		files map[string]string
		host  string
	}{
		{
			"stable inputs produce stable seed",
			map[string]string{"/etc/machine-id": "abc123\n"},
			"web01",
		},
		{
			"dbus fallback when /etc/machine-id missing",
			map[string]string{"/var/lib/dbus/machine-id": "dbus-id\n"},
			"web01",
		},
		{"empty machine-id still produces a seed", map[string]string{}, "laptop"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &shard.Linux{
				ReadFileFn: func(p string) ([]byte, error) {
					if v, ok := tt.files[p]; ok {
						return []byte(v), nil
					}
					return nil, errors.New("not found")
				},
				HostnameFn: func() (string, error) { return tt.host, nil },
			}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*shard.Info)
			s.Require().True(ok)
			s.Len(info.Seed, 64)
		})
	}
}

func (s *ShardPublicTestSuite) TestCollectLinuxDeterministic() {
	c := &shard.Linux{
		ReadFileFn: func(string) ([]byte, error) { return []byte("stable-id"), nil },
		HostnameFn: func() (string, error) { return "stable-host", nil },
	}
	first, err := c.Collect(context.Background())
	s.Require().NoError(err)
	second, err := c.Collect(context.Background())
	s.Require().NoError(err)
	s.Equal(first, second)
}

func (s *ShardPublicTestSuite) TestCollectDarwin() {
	tests := []struct {
		name     string
		midFn    func(context.Context) (string, error)
		hostname string
	}{
		{
			"IOPlatformUUID + hostname",
			func(context.Context) (string, error) { return "uuid-1234", nil },
			"johns-mac",
		},
		{
			"MachineIDFn error → empty machine_id, seed still computes",
			func(context.Context) (string, error) { return "", errors.New("boom") },
			"johns-mac",
		},
		{
			"empty machine_id (nil info upstream) → seed still computes",
			func(context.Context) (string, error) { return "", nil },
			"johns-mac",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &shard.Darwin{
				MachineIDFn: tt.midFn,
				HostnameFn:  func() (string, error) { return tt.hostname, nil },
			}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*shard.Info)
			s.Require().True(ok)
			s.Len(info.Seed, 64)
		})
	}
}

func (s *ShardPublicTestSuite) TestReadMachineUUID() {
	tests := []struct {
		name    string
		fn      func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    string
	}{
		{
			name: "success returns HostID",
			fn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{HostID: "uuid-abcd"}, nil
			},
			want: "uuid-abcd",
		},
		{
			name: "nil info returns empty",
			fn:   func(context.Context) (*host.InfoStat, error) { return nil, nil },
			want: "",
		},
		{
			name:    "gopsutil error returned",
			fn:      func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := shard.SetHostInfoFn(tt.fn)
			defer restore()
			got, err := shard.ReadMachineUUID(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
