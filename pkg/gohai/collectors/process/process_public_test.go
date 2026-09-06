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

package process_test

import (
	"context"
	"errors"
	"os"
	"testing"

	gpprocess "github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
)

var (
	_ collector.Collector = (*process.Linux)(nil)
	_ collector.Collector = (*process.Darwin)(nil)
)

type ProcessPublicTestSuite struct {
	suite.Suite
}

func TestProcessPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessPublicTestSuite))
}

func (s *ProcessPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(process.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c process.Collector) {
				_, ok := c.(*process.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := process.New()
			s.Equal("process", c.Name())
			s.Equal("misc", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *ProcessPublicTestSuite) TestCollect() {
	ownPID := int32(os.Getpid())
	tests := []struct {
		name         string
		variant      string
		fn           func(context.Context) ([]*gpprocess.Process, error)
		validateFunc func(any, error)
	}{
		{
			name:    "linux: snapshot wraps gopsutil processes",
			variant: "linux",
			fn: func(context.Context) ([]*gpprocess.Process, error) {
				p, _ := gpprocess.NewProcess(ownPID)
				return []*gpprocess.Process{p}, nil
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*process.Info)
				s.Require().True(ok)
				s.Equal(1, info.Count)
			},
		},
		{
			name:    "linux: empty process list",
			variant: "linux",
			fn: func(context.Context) ([]*gpprocess.Process, error) {
				return []*gpprocess.Process{}, nil
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*process.Info)
				s.Require().True(ok)
				s.Equal(0, info.Count)
			},
		},
		{
			name:    "linux: gopsutil error propagated",
			variant: "linux",
			fn:      func(context.Context) ([]*gpprocess.Process, error) { return nil, errors.New("boom") },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: macOS snapshot",
			variant: "darwin",
			fn: func(context.Context) ([]*gpprocess.Process, error) {
				return []*gpprocess.Process{}, nil
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*process.Info)
				s.Require().True(ok)
				s.Equal(0, info.Count)
			},
		},
		{
			name:    "darwin: processes error propagated",
			variant: "darwin",
			fn:      func(context.Context) ([]*gpprocess.Process, error) { return nil, errors.New("kauth denied") },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer process.SetProcessesFn(tt.fn)()
			var c process.Collector
			switch tt.variant {
			case "linux":
				c = &process.Linux{}
			case "darwin":
				c = &process.Darwin{}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
