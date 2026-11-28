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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/family"
)

type PlatformPublicTestSuite struct {
	suite.Suite
}

func TestPlatformPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

// TestDetect: Detect is the family package's, and a caller can swap it.
// The families themselves are tested in pkg/gohai/family.
func (s *PlatformPublicTestSuite) TestDetect() {
	tests := []struct {
		name         string
		detect       func() string
		validateFunc func(string)
	}{
		{
			name: "by default is the family that family.Detect finds",
			validateFunc: func(got string) {
				s.Equal(family.Detect(), got)
			},
		},
		{
			name:   "a test can replace it",
			detect: func() string { return "custom" },
			validateFunc: func(got string) {
				s.Equal("custom", got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.detect != nil {
				orig := platform.Detect
				defer func() { platform.Detect = orig }()
				platform.Detect = tt.detect
			}

			tt.validateFunc(platform.Detect())
		})
	}
}

func (s *PlatformPublicTestSuite) TestIsLinux() {
	tests := []struct {
		name   string
		detect string
		want   bool
	}{
		{name: "when the family is debian", detect: family.Debian, want: true},
		{name: "when a Linux host has no family", detect: "nixos", want: true},
		{name: "when the host is macOS", detect: family.Darwin, want: false},
		{name: "when the host could not be read", detect: "", want: false},
	}
	orig := platform.Detect
	defer func() { platform.Detect = orig }()
	for _, tc := range tests {
		s.Run(tc.name, func() {
			platform.Detect = func() string { return tc.detect }
			s.Equal(tc.want, platform.IsLinux())
		})
	}
}

func (s *PlatformPublicTestSuite) TestIsDarwin() {
	tests := []struct {
		name   string
		detect string
		want   bool
	}{
		{name: "when the host is macOS", detect: family.Darwin, want: true},
		{name: "when the host is Linux", detect: family.RHEL, want: false},
	}
	orig := platform.Detect
	defer func() { platform.Detect = orig }()
	for _, tc := range tests {
		s.Run(tc.name, func() {
			platform.Detect = func() string { return tc.detect }
			s.Equal(tc.want, platform.IsDarwin())
		})
	}
}
