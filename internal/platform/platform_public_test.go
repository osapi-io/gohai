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

package platform_test

import (
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/platform"
)

type PlatformPublicTestSuite struct {
	suite.Suite
}

func TestPlatformPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

func (s *PlatformPublicTestSuite) TestDetect() {
	tests := []struct {
		name   string
		infoFn func() (*host.InfoStat, error)
		want   string
	}{
		{
			name:   "ubuntu → debian",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "ubuntu"}, nil },
			want:   "debian",
		},
		{
			name:   "debian → debian",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "debian"}, nil },
			want:   "debian",
		},
		{
			name:   "raspbian → debian",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "raspbian"}, nil },
			want:   "debian",
		},
		{
			name:   "rhel → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "rhel"}, nil },
			want:   "rhel",
		},
		{
			name:   "centos → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "centos"}, nil },
			want:   "rhel",
		},
		{
			name:   "fedora → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "fedora"}, nil },
			want:   "rhel",
		},
		{
			name:   "amazon linux → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "amzn"}, nil },
			want:   "rhel",
		},
		{
			name:   "darwin platform empty, OS=darwin",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "", OS: "darwin"}, nil },
			want:   "darwin",
		},
		{
			name:   "arch linux passes through",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "arch"}, nil },
			want:   "arch",
		},
		{
			name:   "alpine passes through",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "alpine"}, nil },
			want:   "alpine",
		},
		{
			name:   "suse passes through (not in rhel family despite similarity)",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "suse"}, nil },
			want:   "suse",
		},
		{
			name:   "gopsutil error returns empty",
			infoFn: func() (*host.InfoStat, error) { return nil, errors.New("boom") },
			want:   "",
		},
		{
			name:   "nil info returns empty",
			infoFn: func() (*host.InfoStat, error) { return nil, nil },
			want:   "",
		},
		{
			name:   "mixed-case platform is normalized",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "Ubuntu"}, nil },
			want:   "debian",
		},
	}

	orig := platform.HostInfoFn
	defer func() { platform.HostInfoFn = orig }()

	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.HostInfoFn = tt.infoFn
			s.Equal(tt.want, platform.Detect())
		})
	}
}

func (s *PlatformPublicTestSuite) TestIsLinux() {
	orig := platform.HostInfoFn
	defer func() { platform.HostInfoFn = orig }()

	tests := []struct {
		name   string
		infoFn func() (*host.InfoStat, error)
		want   bool
	}{
		{"ubuntu is linux", func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "ubuntu"}, nil }, true},
		{"arch is linux", func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "arch"}, nil }, true},
		{"darwin is not linux", func() (*host.InfoStat, error) { return &host.InfoStat{OS: "darwin"}, nil }, false},
		{"empty not linux", func() (*host.InfoStat, error) { return nil, nil }, false},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.HostInfoFn = tt.infoFn
			s.Equal(tt.want, platform.IsLinux())
		})
	}
}

func (s *PlatformPublicTestSuite) TestIsDarwin() {
	orig := platform.HostInfoFn
	defer func() { platform.HostInfoFn = orig }()

	platform.HostInfoFn = func() (*host.InfoStat, error) { return &host.InfoStat{OS: "darwin"}, nil }
	s.True(platform.IsDarwin())
	platform.HostInfoFn = func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "ubuntu"}, nil }
	s.False(platform.IsDarwin())
}
