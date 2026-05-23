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

package virtualbox_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualbox"
)

var (
	_ collector.Collector = (*virtualbox.Linux)(nil)
	_ collector.Collector = (*virtualbox.Darwin)(nil)
)

// canonical VBoxControl guestproperty enumerate output from a real VirtualBox
// guest. Mirrors the lines parsed by Ohai's virtualbox.rb guest branch.
const vboxControlOutput = `Name: /VirtualBox/GuestInfo/OS/Product, value: Linux, timestamp: 0, flags:
Name: /VirtualBox/HostInfo/VBoxVer, value: 7.0.14, timestamp: 0, flags:
Name: /VirtualBox/HostInfo/VBoxRev, value: 161095, timestamp: 0, flags:
Name: /VirtualBox/GuestAdd/VersionExt, value: 7.0.14, timestamp: 0, flags:
Name: /VirtualBox/GuestAdd/Revision, value: 161095, timestamp: 0, flags:
Name: /VirtualBox/HostInfo/LanguageID, value: en_US, timestamp: 0, flags:
`

type VirtualBoxPublicTestSuite struct {
	suite.Suite
}

func TestVirtualBoxPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VirtualBoxPublicTestSuite))
}

// vboxExec returns a MockExecutor that answers `VBoxControl guestproperty enumerate`
// with the supplied output/error pair.
func vboxExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "VBoxControl", "guestproperty", "enumerate").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *VirtualBoxPublicTestSuite) TestNew() {
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
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := virtualbox.New()
			s.Equal("virtualbox", c.Name())
			s.Equal("virtualization", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*virtualbox.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*virtualbox.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *VirtualBoxPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		exec     func(*testing.T) executor.Executor
		wantNil  bool
		validate func(*virtualbox.Info)
	}{
		// ----- Linux -----
		{
			name:    "linux: canonical guestproperty output parsed",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return vboxExec(t, []byte(vboxControlOutput), nil)
			},
			validate: func(i *virtualbox.Info) {
				s.Equal("7.0.14", i.HostVersion)
				s.Equal("161095", i.HostRevision)
				s.Equal("7.0.14", i.GuestAdditionsVersion)
				s.Equal("161095", i.GuestAdditionsRevision)
				s.Equal("en_US", i.LanguageID)
			},
		},
		{
			name:    "linux: VBoxControl not found returns nil",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return vboxExec(t, nil, errors.New("exec: not found"))
			},
			wantNil: true,
		},
		{
			name:    "linux: nil Exec returns nil",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			wantNil: true,
		},
		{
			name:    "linux: output with no matching lines yields empty Info",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return vboxExec(
					t,
					[]byte("Name: /VirtualBox/GuestInfo/OS/Product, value: Linux\n"),
					nil,
				)
			},
			validate: func(i *virtualbox.Info) {
				s.Empty(i.HostVersion)
				s.Empty(i.GuestAdditionsVersion)
			},
		},
		{
			name:    "linux: partial output — only VBoxVer present",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return vboxExec(
					t,
					[]byte(
						"Name: /VirtualBox/HostInfo/VBoxVer, value: 6.1.0, timestamp: 0, flags:\n",
					),
					nil,
				)
			},
			validate: func(i *virtualbox.Info) {
				s.Equal("6.1.0", i.HostVersion)
				s.Empty(i.HostRevision)
				s.Empty(i.GuestAdditionsVersion)
			},
		},
		// ----- Darwin -----
		{
			name:    "darwin: canonical guestproperty output parsed",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return vboxExec(t, []byte(vboxControlOutput), nil)
			},
			validate: func(i *virtualbox.Info) {
				s.Equal("7.0.14", i.HostVersion)
				s.Equal("7.0.14", i.GuestAdditionsVersion)
				s.Equal("en_US", i.LanguageID)
			},
		},
		{
			name:    "darwin: VBoxControl not found returns nil",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return vboxExec(t, nil, errors.New("exec: not found"))
			},
			wantNil: true,
		},
		{
			name:    "darwin: nil Exec returns nil",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c virtualbox.Collector
			switch tt.variant {
			case "linux":
				c = &virtualbox.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &virtualbox.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*virtualbox.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
