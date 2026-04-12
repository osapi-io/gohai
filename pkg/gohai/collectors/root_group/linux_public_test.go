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

	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
)

type RootGroupLinuxPublicTestSuite struct {
	suite.Suite
}

func TestRootGroupLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RootGroupLinuxPublicTestSuite))
}

func (s *RootGroupLinuxPublicTestSuite) TestCollect() {
	rootUser := &user.User{Username: "root", Uid: "0", Gid: "0"}
	rootGroup := &user.Group{Gid: "0", Name: "root"}
	customRootUser := &user.User{Username: "root", Uid: "0", Gid: "1000"}
	customGroup := &user.Group{Gid: "1000", Name: "wheel"}

	tests := []struct {
		name          string
		lookupUserFn  func(string) (*user.User, error)
		lookupGroupFn func(string) (*user.Group, error)
		wantErr       bool
		want          string
	}{
		{
			name:          "root user → root group (standard Linux)",
			lookupUserFn:  func(string) (*user.User, error) { return rootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return rootGroup, nil },
			want:          "root",
		},
		{
			name:          "root primary gid customized to non-zero",
			lookupUserFn:  func(string) (*user.User, error) { return customRootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return customGroup, nil },
			want:          "wheel",
		},
		{
			name:          "user lookup error propagated",
			lookupUserFn:  func(string) (*user.User, error) { return nil, errors.New("no root user") },
			lookupGroupFn: func(string) (*user.Group, error) { return rootGroup, nil },
			wantErr:       true,
		},
		{
			name:          "group lookup error propagated",
			lookupUserFn:  func(string) (*user.User, error) { return rootUser, nil },
			lookupGroupFn: func(string) (*user.Group, error) { return nil, errors.New("no such group") },
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &rootgroup.Linux{LookupUserFn: tt.lookupUserFn, LookupGroupFn: tt.lookupGroupFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*rootgroup.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Name)
		})
	}
}

// TestNewLinuxWiresUp confirms the factory wires os/user lookups into
// the struct. We don't exercise the real functions here — those are
// stdlib, and the collector Collect tests above use injected stubs.
func (s *RootGroupLinuxPublicTestSuite) TestNewLinuxWiresUp() {
	c := rootgroup.NewLinux()
	s.NotNil(c.LookupUserFn)
	s.NotNil(c.LookupGroupFn)
}
