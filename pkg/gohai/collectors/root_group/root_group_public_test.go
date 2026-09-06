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

package rootgroup_test

import (
	"context"
	"errors"
	"os/user"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
)

var (
	_ collector.Collector = (*rootgroup.Linux)(nil)
	_ collector.Collector = (*rootgroup.Darwin)(nil)
)

type RootGroupPublicTestSuite struct {
	suite.Suite
}

func TestRootGroupPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RootGroupPublicTestSuite))
}

func (s *RootGroupPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(rootgroup.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c rootgroup.Collector) {
				_, ok := c.(*rootgroup.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := rootgroup.New()
			s.Equal("root_group", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *RootGroupPublicTestSuite) TestCollect() {
	rootUser := &user.User{Username: "root", Uid: "0", Gid: "0"}
	rootGroup := &user.Group{Gid: "0", Name: "root"}
	customRootUser := &user.User{Username: "root", Uid: "0", Gid: "1000"}
	customGroup := &user.Group{Gid: "1000", Name: "wheel"}
	wheelGroup := &user.Group{Gid: "0", Name: "wheel"}

	tests := []struct {
		name          string
		variant       string
		lookupUserFn  func(string) (*user.User, error)
		lookupGroupFn func(string) (*user.Group, error)
		validateFunc  func(any, error)
	}{
		{
			name:          "linux: root user → root group (standard Linux)",
			variant:       "linux",
			lookupUserFn:  func(string) (*user.User, error) { return rootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return rootGroup, nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*rootgroup.Info)
				s.Require().True(ok)
				s.Equal("root", info.Name)
			},
		},
		{
			name:          "linux: root primary gid customized to non-zero",
			variant:       "linux",
			lookupUserFn:  func(string) (*user.User, error) { return customRootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return customGroup, nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*rootgroup.Info)
				s.Require().True(ok)
				s.Equal("wheel", info.Name)
			},
		},
		{
			name:          "linux: user lookup error propagated",
			variant:       "linux",
			lookupUserFn:  func(string) (*user.User, error) { return nil, errors.New("no root user") },
			lookupGroupFn: func(string) (*user.Group, error) { return rootGroup, nil },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:          "linux: group lookup error propagated",
			variant:       "linux",
			lookupUserFn:  func(string) (*user.User, error) { return rootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return nil, errors.New("no such group") },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:          "darwin: root user → wheel group on macOS",
			variant:       "darwin",
			lookupUserFn:  func(string) (*user.User, error) { return rootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return wheelGroup, nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*rootgroup.Info)
				s.Require().True(ok)
				s.Equal("wheel", info.Name)
			},
		},
		{
			name:          "darwin: user lookup error propagated",
			variant:       "darwin",
			lookupUserFn:  func(string) (*user.User, error) { return nil, errors.New("no root user") },
			lookupGroupFn: func(string) (*user.Group, error) { return wheelGroup, nil },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:          "darwin: group lookup error propagated",
			variant:       "darwin",
			lookupUserFn:  func(string) (*user.User, error) { return rootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return nil, errors.New("no such group") },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer rootgroup.SetLookupUserFn(tt.lookupUserFn)()
			defer rootgroup.SetLookupGroupFn(tt.lookupGroupFn)()
			var c rootgroup.Collector
			switch tt.variant {
			case "linux":
				c = &rootgroup.Linux{}
			case "darwin":
				c = &rootgroup.Darwin{}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
