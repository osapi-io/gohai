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

package languages_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
)

var (
	_ collector.Collector = (*languages.Linux)(nil)
	_ collector.Collector = (*languages.Darwin)(nil)
)

type LanguagesPublicTestSuite struct {
	suite.Suite
}

func TestLanguagesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LanguagesPublicTestSuite))
}

func strPtr(
	s string,
) *string {
	return &s
}

// buildFullMock creates a mock that returns canned output for all six
// language probes.
func buildFullMock(
	t *testing.T,
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "go", "version").
		Return([]byte("go version go1.21.0 linux/amd64\n"), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "python3", "--version").
		Return([]byte("Python 3.11.4\n"), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "ruby", "--version").
		Return([]byte("ruby 3.2.2 (2023-03-30)\n"), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "node", "--version").
		Return([]byte("v20.1.0\n"), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "java", "-version").
		Return([]byte("openjdk version \"21.0.1\" 2023-10-17\n"), nil).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "perl", "--version").
		Return([]byte("This is perl 5, version 36, subversion 0 (v5.36.0)\n"), nil).
		AnyTimes()
	return m
}

// buildAbsentMock creates a mock where every probe fails.
func buildAbsentMock(
	t *testing.T,
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	absent := errors.New("not found")
	m.EXPECT().Execute(gomock.Any(), "go", "version").Return(nil, absent).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "python3", "--version").Return(nil, absent).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "ruby", "--version").Return(nil, absent).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "node", "--version").Return(nil, absent).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "java", "-version").Return(nil, absent).AnyTimes()
	m.EXPECT().Execute(gomock.Any(), "perl", "--version").Return(nil, absent).AnyTimes()
	return m
}

type langOverride struct {
	cmd    string
	arg    string
	output string
}

// buildSingleMock creates a mock where one language returns custom output
// and the rest are absent.
func buildSingleMock(
	t *testing.T,
	override langOverride,
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	absent := errors.New("not found")

	probes := []struct{ cmd, arg string }{
		{"go", "version"},
		{"python3", "--version"},
		{"ruby", "--version"},
		{"node", "--version"},
		{"java", "-version"},
		{"perl", "--version"},
	}
	for _, p := range probes {
		if p.cmd == override.cmd && p.arg == override.arg {
			m.EXPECT().
				Execute(gomock.Any(), p.cmd, p.arg).
				Return([]byte(override.output), nil).
				AnyTimes()
		} else {
			m.EXPECT().
				Execute(gomock.Any(), p.cmd, p.arg).
				Return(nil, absent).
				AnyTimes()
		}
	}

	return m
}

func (s *LanguagesPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(languages.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c languages.Collector) {
				_, ok := c.(*languages.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c languages.Collector) {
				_, ok := c.(*languages.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c languages.Collector) {
				_, ok := c.(*languages.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c languages.Collector) {
				_, ok := c.(*languages.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c languages.Collector) {
				_, ok := c.(*languages.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := languages.New()
			s.Equal("languages", c.Name())
			s.Equal("software", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *LanguagesPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		exec         executor.Executor
		validateFunc func(any, error)
	}{
		{
			name:    "linux: all runtimes present",
			variant: "linux",
			exec:    buildFullMock(s.T()),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{
					Go:     strPtr("1.21.0"),
					Python: strPtr("3.11.4"),
					Ruby:   strPtr("3.2.2"),
					Node:   strPtr("20.1.0"),
					Java:   strPtr("21.0.1"),
					Perl:   strPtr("v5.36.0"),
				}, *info)
			},
		},
		{
			name:    "linux: no runtimes present, all nil",
			variant: "linux",
			exec:    buildAbsentMock(s.T()),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{}, *info)
			},
		},
		{
			name:    "linux: nil Exec returns empty Info",
			variant: "linux",
			exec:    nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{}, *info)
			},
		},
		{
			name:    "darwin: all runtimes present",
			variant: "darwin",
			exec:    buildFullMock(s.T()),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{
					Go:     strPtr("1.21.0"),
					Python: strPtr("3.11.4"),
					Ruby:   strPtr("3.2.2"),
					Node:   strPtr("20.1.0"),
					Java:   strPtr("21.0.1"),
					Perl:   strPtr("v5.36.0"),
				}, *info)
			},
		},
		{
			name:    "darwin: no runtimes present, all nil",
			variant: "darwin",
			exec:    buildAbsentMock(s.T()),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{}, *info)
			},
		},
		{
			name:    "darwin: nil Exec returns empty Info",
			variant: "darwin",
			exec:    nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{}, *info)
			},
		},
		{
			name:    "linux: go version without go prefix falls back to trimmed output",
			variant: "linux",
			exec:    buildSingleMock(s.T(), langOverride{"go", "version", "something 1.2.3"}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{Go: strPtr("something 1.2.3")}, *info)
			},
		},
		{
			name:    "linux: python single token fallback",
			variant: "linux",
			exec:    buildSingleMock(s.T(), langOverride{"python3", "--version", "3.11.4"}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{Python: strPtr("3.11.4")}, *info)
			},
		},
		{
			name:    "linux: ruby single token fallback",
			variant: "linux",
			exec:    buildSingleMock(s.T(), langOverride{"ruby", "--version", "3.2.2"}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{Ruby: strPtr("3.2.2")}, *info)
			},
		},
		{
			name:    "linux: java version without quotes uses first line",
			variant: "linux",
			exec: buildSingleMock(
				s.T(),
				langOverride{"java", "-version", "java version abc\nother line"},
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{Java: strPtr("java version abc")}, *info)
			},
		},
		{
			name:    "linux: perl version without parens falls back to first line",
			variant: "linux",
			exec:    buildSingleMock(s.T(), langOverride{"perl", "--version", "perl 5.36"}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*languages.Info)
				s.Require().True(ok)
				s.Equal(languages.Info{Perl: strPtr("perl 5.36")}, *info)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c languages.Collector
			switch tt.variant {
			case "linux":
				c = &languages.Linux{Exec: tt.exec}
			case "darwin":
				c = &languages.Darwin{Exec: tt.exec}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
