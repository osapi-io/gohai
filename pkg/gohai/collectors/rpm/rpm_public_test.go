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

package rpm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs/vfs/osfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rpm"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*rpm.Linux)(nil)
	_ collector.Collector = (*rpm.Darwin)(nil)
)

// showrcNoMarker has no `=====` marker lines — parseShowrc returns empty.
const showrcNoMarker = `ARCHITECTURE AND OS:
build arch : x86_64
- %_topdir /root/rpmbuild
`

type RPMPublicTestSuite struct {
	suite.Suite
}

func TestRPMPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RPMPublicTestSuite))
}

func (s *RPMPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(rpm.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c rpm.Collector) {
				_, ok := c.(*rpm.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c rpm.Collector) {
				_, ok := c.(*rpm.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c rpm.Collector) {
				_, ok := c.(*rpm.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c rpm.Collector) {
				_, ok := c.(*rpm.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c rpm.Collector) {
				_, ok := c.(*rpm.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := rpm.New()
			s.Equal("rpm", c.Name())
			s.Equal("linux", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *RPMPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		setupExec    func(ctrl *gomock.Controller) *sourcemocks.MockRunner
		validateFunc func(any, error)
	}{
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Nil(got)
			},
		},
		{
			name:    "linux: rpm not installed — empty macros, no error",
			variant: "linux",
			setupExec: func(ctrl *gomock.Controller) *sourcemocks.MockRunner {
				m := sourcemocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "rpm", "--showrc").
					Return(nil, errors.New("rpm: command not found"))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)

				info, ok := got.(*rpm.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Macros)
			},
		},
		{
			name:    "linux: rpm --showrc, as Fedora prints it, parsed correctly",
			variant: "linux",
			setupExec: func(ctrl *gomock.Controller) *sourcemocks.MockRunner {
				out, err := osfs.NewWithNoIdm().ReadFile("testdata/showrc-fedora.txt")
				s.Require().NoError(err)
				m := sourcemocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "rpm", "--showrc").
					Return(out, nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)

				info, ok := got.(*rpm.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"P":             "<builtin>",
					"S":             "<builtin>",
					"__7zip":        "/usr/bin/7za",
					"___build_args": "-e",
					"___build_cmd": "%{?_sudo:%{_sudo} }%{?_remsh:%{_remsh} %{_remhost} }" +
						"%{?_remsudo:%{_remsudo} }%{?_remchroot:%{_remchroot} %{_remroot} }" +
						"%{___build_shell} %{___build_args}",
					"___build_post": "",
					"__spec_install_pre": "%{___build_pre}\n" +
						"    %{__rm} -rf \"%{buildroot}\"\n" +
						"    %{__mkdir_p} \"%{dirname:%{buildroot}}\"\n" +
						"    %{__mkdir} \"%{buildroot}\"\n" +
						"%{nil}",
					"_target": "aarch64-linux",
					"with":    "%{expand:%%{?with_%{1}:1}%%{!?with_%{1}:0}}",
					"without": "%{expand:%%{?with_%{1}:0}%%{!?with_%{1}:1}}",
					"x86_64":  "x86_64 x86_64_v2 x86_64_v3 x86_64_v4 amd64 em64t",
				}, info.Macros)
			},
		},
		{
			name:    "linux: no marker lines — empty macros",
			variant: "linux",
			setupExec: func(ctrl *gomock.Controller) *sourcemocks.MockRunner {
				m := sourcemocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "rpm", "--showrc").
					Return([]byte(showrcNoMarker), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)

				info, ok := got.(*rpm.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Macros)
			},
		},
		{
			name:      "linux: nil executor — empty macros",
			variant:   "linux",
			setupExec: nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)

				info, ok := got.(*rpm.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Macros)
			},
		},
		{
			// A macro line that is just "-" (no name) exercises the
			// len(parts) < 2 branch; a macro line with only "-  name"
			// (no value) exercises the len(parts) != 3 branch.
			name:    "linux: edge-case macro lines",
			variant: "linux",
			setupExec: func(ctrl *gomock.Controller) *sourcemocks.MockRunner {
				m := sourcemocks.NewMockRunner(ctrl)
				// A continuation before any macro is dropped. A macro with no
				// tab has an empty body. A run-time macro has level 0 or more.
				out := "=====\nstray\n-13: novalue\n  0: runtime\tset\n 12: nested\tdeep\n=====\n"
				m.EXPECT().
					Execute(gomock.Any(), "rpm", "--showrc").
					Return([]byte(out), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)

				info, ok := got.(*rpm.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"novalue": "",
					"runtime": "set",
					"nested":  "deep",
				}, info.Macros)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())

			var c rpm.Collector
			switch tt.variant {
			case "linux":
				l := &rpm.Linux{}
				if tt.setupExec != nil {
					l.Exec = tt.setupExec(ctrl)
				}
				c = l
			case "darwin":
				c = &rpm.Darwin{}
			}

			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
