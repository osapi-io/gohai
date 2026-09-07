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

func TestPlatformPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

// TestDetect covers the Detect function across every family-mapping
// branch and every fallback path. Also includes a row exercising the
// swappable-var contract that collector tests rely on.
func (s *PlatformPublicTestSuite) TestDetect() {
	tests := []struct {
		name         string
		infoFn       func() (*host.InfoStat, error)
		override     func() string // non-nil swaps platform.Detect directly
		validateFunc func(string)
	}{
		{
			name:   "ubuntu → debian",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "ubuntu"}, nil },
			validateFunc: func(got string) {
				s.Equal("debian", got)
			},
		},
		{
			name:   "debian → debian",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "debian"}, nil },
			validateFunc: func(got string) {
				s.Equal("debian", got)
			},
		},
		{
			name:   "raspbian → debian",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "raspbian"}, nil },
			validateFunc: func(got string) {
				s.Equal("debian", got)
			},
		},
		{
			name:   "rhel → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "rhel"}, nil },
			validateFunc: func(got string) {
				s.Equal("rhel", got)
			},
		},
		{
			name:   "centos → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "centos"}, nil },
			validateFunc: func(got string) {
				s.Equal("rhel", got)
			},
		},
		{
			name:   "fedora → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "fedora"}, nil },
			validateFunc: func(got string) {
				s.Equal("rhel", got)
			},
		},
		{
			name:   "amazon linux → rhel",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "amzn"}, nil },
			validateFunc: func(got string) {
				s.Equal("rhel", got)
			},
		},
		{
			name:   "darwin platform empty, OS=darwin",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "", OS: "darwin"}, nil },
			validateFunc: func(got string) {
				s.Equal("darwin", got)
			},
		},
		{
			name:   "arch linux passes through",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "arch"}, nil },
			validateFunc: func(got string) {
				s.Equal("arch", got)
			},
		},
		{
			name:   "alpine passes through",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "alpine"}, nil },
			validateFunc: func(got string) {
				s.Equal("alpine", got)
			},
		},
		{
			name:   "suse passes through (not rhel family)",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "suse"}, nil },
			validateFunc: func(got string) {
				s.Equal("suse", got)
			},
		},
		{
			name:   "gopsutil error returns empty",
			infoFn: func() (*host.InfoStat, error) { return nil, errors.New("boom") },
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
		{
			name:   "nil info returns empty",
			infoFn: func() (*host.InfoStat, error) { return nil, nil },
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
		{
			name:   "mixed-case platform is normalized",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "Ubuntu"}, nil },
			validateFunc: func(got string) {
				s.Equal("debian", got)
			},
		},
		{
			name:     "Detect var is swappable by callers without gopsutil",
			override: func() string { return "custom" },
			validateFunc: func(got string) {
				s.Equal("custom", got)
			},
		},
	}

	origDetect := platform.Detect
	defer func() { platform.Detect = origDetect }()

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.override != nil {
				platform.Detect = tt.override

				tt.validateFunc(platform.Detect())

				platform.Detect = origDetect

				return
			}
			restore := platform.SetHostInfoFn(tt.infoFn)
			defer restore()

			tt.validateFunc(platform.Detect())
		})
	}
}

func (s *PlatformPublicTestSuite) TestIsLinux() {
	tests := []struct {
		name         string
		infoFn       func() (*host.InfoStat, error)
		validateFunc func(bool)
	}{
		{
			name:   "ubuntu is linux",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "ubuntu"}, nil },
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:   "arch is linux",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "arch"}, nil },
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:   "darwin is not linux",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{OS: "darwin"}, nil },
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:   "empty not linux",
			infoFn: func() (*host.InfoStat, error) { return nil, nil },
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := platform.SetHostInfoFn(tt.infoFn)
			defer restore()
			tt.validateFunc(platform.IsLinux())
		})
	}
}

func (s *PlatformPublicTestSuite) TestIsDarwin() {
	tests := []struct {
		name         string
		infoFn       func() (*host.InfoStat, error)
		validateFunc func(bool)
	}{
		{
			name:   "darwin is darwin",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{OS: "darwin"}, nil },
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:   "ubuntu is not darwin",
			infoFn: func() (*host.InfoStat, error) { return &host.InfoStat{Platform: "ubuntu"}, nil },
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:   "empty not darwin",
			infoFn: func() (*host.InfoStat, error) { return nil, nil },
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := platform.SetHostInfoFn(tt.infoFn)
			defer restore()
			tt.validateFunc(platform.IsDarwin())
		})
	}
}
