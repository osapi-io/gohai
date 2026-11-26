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

package linode_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

type LinodePublicTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestLinodePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LinodePublicTestSuite))
}

func (s *LinodePublicTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// writeApt writes a sources.list replacement and swaps the package
// var.
func (s *LinodePublicTestSuite) writeApt(
	content string,
) func() {
	path := filepath.Join(s.tmpDir, "sources.list")
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
	return linode.SetAptSourcesPath(path)
}

func (s *LinodePublicTestSuite) TestInterface() {
	c := linode.New()
	tests := []struct {
		name         string
		got          any
		validateFunc func(any)
	}{
		{
			name: "Name",
			got:  c.Name(),
			validateFunc: func(got any) {
				s.Equal("linode", got)
			},
		},
		{
			name: "Category",
			got:  string(c.Category()),
			validateFunc: func(got any) {
				s.Equal("cloud", got)
			},
		},
		{
			name: "DefaultEnabled",
			got:  c.DefaultEnabled(),
			validateFunc: func(got any) {
				s.Equal(false, got)
			},
		},
		{
			name: "Dependencies",
			got:  c.Dependencies(),
			validateFunc: func(got any) {
				s.Equal([]string{"hostname"}, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.got)
		})
	}
}

func (s *LinodePublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		apt          string
		noApt        bool
		hostname     *hostname.Info // when set, added to prior under "hostname"
		ip           string         // the output of ip -o -4 addr show
		ipErr        error
		validateFunc func(any)
	}{
		{
			name:     "FQDN contains 'linode' triggers detection without apt",
			apt:      "deb http://archive.ubuntu.com/ubuntu focal main",
			hostname: &hostname.Info{FQDN: "myhost.members.linode.com"},
			ip:       ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:     "Domain contains 'linode' triggers detection",
			apt:      "deb http://archive.ubuntu.com/ubuntu focal main",
			hostname: &hostname.Info{Domain: "members.linode.com"},
			ip:       ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:     "hostname without linode + no apt → nil",
			apt:      "deb http://archive.ubuntu.com/ubuntu focal main",
			hostname: &hostname.Info{FQDN: "host.example.com", Domain: "example.com"},
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name:     "nil hostname Info is tolerated",
			apt:      "linode",
			hostname: nil,
			ip:       ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name: "apt says linode, eth0 and the eth0:1 alias populated (link-local skipped)",
			apt:  "deb http://mirrors.linode.com/ focal main",
			// Link-local comes first: it must be skipped for the routable
			// address that follows. The alias is a label of eth0, not an
			// interface of its own.
			ip: ipLine("eth0", "eth0", "169.254.0.1/16") +
				ipLine("eth0", "eth0", "50.1.2.3/24") +
				ipLine("eth0", "eth0:1", "10.0.0.5/17") +
				ipLine("eth0", "eth0", "50.1.2.4/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("50.1.2.3", info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name: "no linode in apt → nil",
			apt:  "deb http://archive.ubuntu.com/ubuntu focal main",
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name:  "missing apt file → nil",
			noApt: true,
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name: "eth0 missing produces empty public_ip",
			apt:  "linode",
			ip:   ipLine("eth0", "eth0:1", "10.0.0.5/17"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Empty(info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name: "lines that are not an IPv4 address are skipped",
			apt:  "linode",
			ip: "\n" +
				"1: lo    inet\n" +
				"2: eth0    inet not-a-cidr scope global eth0\\ valid_lft forever\n" +
				"2: eth0    inet6 fe80::1/64 scope link\\ valid_lft forever\n",
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Empty(info.PublicIP)
				s.Empty(info.PrivateIP)
			},
		},
		{
			name:  "an ip error returns empty IPs",
			apt:   "linode",
			ipErr: errors.New("ip: not found"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Empty(info.PublicIP)
				s.Empty(info.PrivateIP)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.noApt {
				defer linode.SetAptSourcesPath(filepath.Join(s.tmpDir, "missing"))()
			} else {
				defer s.writeApt(tt.apt)()
			}

			m := sourcemocks.NewMockRunner(gomock.NewController(s.T()))
			m.EXPECT().
				Execute(gomock.Any(), "ip", "-o", "-4", "addr", "show").
				Return([]byte(tt.ip), tt.ipErr).
				AnyTimes()

			var prior collector.PriorResults
			if tt.hostname != nil {
				prior = collector.PriorResults{"hostname": tt.hostname}
			}
			c := linode.NewWith(source.Sources{Exec: m, Family: family.Debian})
			out, err := c.Collect(context.Background(), prior)
			s.Require().NoError(err)
			tt.validateFunc(out)
		})
	}
}

func (s *LinodePublicTestSuite) TestNewWith() {
	fs := memfs.New()
	s.Require().NoError(fs.MkdirAll("/etc/apt", 0o755))
	s.Require().NoError(fs.WriteFile(
		"/etc/apt/sources.list",
		[]byte("deb http://mirrors.linode.com/debian bookworm main\n"),
		0o644,
	))
	m := sourcemocks.NewMockRunner(gomock.NewController(s.T()))
	m.EXPECT().
		Execute(gomock.Any(), "ip", "-o", "-4", "addr", "show").
		Return([]byte(ipLine("eth0", "eth0", "50.1.2.3/24")), nil)

	c := linode.NewWith(source.Sources{FS: fs, Exec: m, Family: family.Debian})
	s.Same(fs, c.FS)
	s.Same(m, c.Exec)
	out, err := c.Collect(context.Background(), nil)
	s.Require().NoError(err)
	info, ok := out.(*linode.Info)
	s.Require().True(ok, "the apt sources of the given filesystem name linode")
	s.Equal("50.1.2.3", info.PublicIP, "the address comes from the given runner")
}

// ipLine is one line of `ip -o -4 addr show`: the address cidr on dev,
// with label, which is dev or an alias such as eth0:1.
func ipLine(
	dev, label, cidr string,
) string {
	return "2: " + dev + "    inet " + cidr + " scope global " + label +
		"\\       valid_lft forever preferred_lft forever\n"
}
