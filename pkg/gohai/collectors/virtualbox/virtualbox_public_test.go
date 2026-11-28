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
	"fmt"
	osexec "os/exec"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualbox"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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

// The command lines that the rows answer, as sourcetest.Commands keys them.
const (
	vboxLine = "VBoxControl guestproperty enumerate"
)

type VirtualBoxPublicTestSuite struct {
	suite.Suite
}

func TestVirtualBoxPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VirtualBoxPublicTestSuite))
}

func (s *VirtualBoxPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(virtualbox.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c virtualbox.Collector) {
				_, ok := c.(*virtualbox.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c virtualbox.Collector) {
				_, ok := c.(*virtualbox.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c virtualbox.Collector) {
				_, ok := c.(*virtualbox.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c virtualbox.Collector) {
				_, ok := c.(*virtualbox.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c virtualbox.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := virtualbox.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = virtualbox.New()
			}
			s.Equal("virtualbox", c.Name())
			s.Equal("virtualization", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *VirtualBoxPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		// ----- Linux -----
		{
			name:    "linux: canonical guestproperty output parsed",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{vboxLine: vboxControlOutput},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*virtualbox.Info)
				s.Require().True(ok)
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{vboxLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a failed run is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{vboxLine: errors.New("exit status 1: permission denied")},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: nil Exec returns nil",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: output with no matching lines yields empty Info",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						vboxLine: "Name: /VirtualBox/GuestInfo/OS/Product, value: Linux\n",
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*virtualbox.Info)
				s.Require().True(ok)
				s.Empty(i.HostVersion)
				s.Empty(i.GuestAdditionsVersion)
			},
		},
		{
			name:    "linux: partial output with only VBoxVer",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{
					vboxLine: "Name: /VirtualBox/HostInfo/VBoxVer, value: 6.1.0, timestamp: 0, flags:\n",
				}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*virtualbox.Info)
				s.Require().True(ok)
				s.Equal("6.1.0", i.HostVersion)
				s.Empty(i.HostRevision)
				s.Empty(i.GuestAdditionsVersion)
			},
		},
		// ----- Darwin -----
		{
			name:    "darwin: canonical guestproperty output parsed",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{vboxLine: vboxControlOutput},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*virtualbox.Info)
				s.Require().True(ok)
				s.Equal("7.0.14", i.HostVersion)
				s.Equal("7.0.14", i.GuestAdditionsVersion)
				s.Equal("en_US", i.LanguageID)
			},
		},
		{
			name:    "darwin: VBoxControl not found returns nil",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{vboxLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: a failed run is an error",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{vboxLine: errors.New("exit status 1: permission denied")},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: nil Exec returns nil",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
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
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
