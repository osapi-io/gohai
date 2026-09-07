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

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rpm"
)

var (
	_ collector.Collector = (*rpm.Linux)(nil)
	_ collector.Collector = (*rpm.Darwin)(nil)
)

// showrcOutput is a trimmed but structurally representative sample of
// `rpm --showrc` output. The two `=====` lines bracket the macros
// section. Macro definitions start with `-`, continuations have no
// prefix.
const showrcOutput = `ARCHITECTURE AND OS:
build arch            : x86_64
Compatible build archs: x86_64 noarch
build os              : Linux
Compatible build os's : Linux
install arch          : x86_64
install os            : Linux
platform string       : x86_64-redhat-linux-gnu
====================================
- %_topdir /root/rpmbuild
- %_builddir %{_topdir}/BUILD
- %_rpmdir %{_topdir}/RPMS
- %_sourcedir %{_topdir}/SOURCES
- %_specdir %{_topdir}/SPECS
- %_srcrpmdir %{_topdir}/SRPMS
- %__cc gcc
  -m64 -mtune=generic
- %buildroot %{_buildrootdir}/%{NAME}-%{VERSION}-%{RELEASE}.%{_arch}
====================================
`

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
			s.Equal("linux", c.Category())
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
		setupExec    func(ctrl *gomock.Controller) *execmocks.MockExecutor
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
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
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
			name:    "linux: rpm --showrc parsed correctly",
			variant: "linux",
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "rpm", "--showrc").
					Return([]byte(showrcOutput), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)

				info, ok := got.(*rpm.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"%_topdir":    "/root/rpmbuild",
					"%_builddir":  "%{_topdir}/BUILD",
					"%_rpmdir":    "%{_topdir}/RPMS",
					"%_sourcedir": "%{_topdir}/SOURCES",
					"%_specdir":   "%{_topdir}/SPECS",
					"%_srcrpmdir": "%{_topdir}/SRPMS",
					"%__cc":       "gcc\n  -m64 -mtune=generic",
					"%buildroot":  "%{_buildrootdir}/%{NAME}-%{VERSION}-%{RELEASE}.%{_arch}",
				}, info.Macros)
			},
		},
		{
			name:    "linux: no marker lines — empty macros",
			variant: "linux",
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
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
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				// "-" alone: len(SplitN("-", " ", 3)) == 1 → skip.
				// "- %novalue": len == 2 → name="%novalue", value="".
				out := `=====
-
- %novalue
=====
`
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
					"%novalue": "",
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
