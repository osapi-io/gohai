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
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
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

func mustCIDR(
	s *LinodePublicTestSuite,
	c string,
) *net.IPNet {
	ip, n, err := net.ParseCIDR(c)
	s.Require().NoError(err)
	n.IP = ip
	return n
}

func (s *LinodePublicTestSuite) TestDefaultInterfaceAddrs() {
	// Find a real loopback interface (name varies: "lo" on Linux,
	// "lo0" on Darwin/BSD). Every Unix host has one.
	all, err := net.Interfaces()
	s.Require().NoError(err)
	var loName string
	for _, i := range all {
		if i.Flags&net.FlagLoopback != 0 {
			loName = i.Name
			break
		}
	}
	s.Require().NotEmpty(loName)

	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{
			// Loopback exists on every Unix — exercises the happy
			// path through net.InterfaceByName + Addrs().
			name:  "loopback returns addresses",
			iface: loName,
		},
		{
			name:    "missing interface returns error",
			iface:   "gohai-nonexistent-iface-xyz",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			addrs, err := linode.DefaultInterfaceAddrs(tt.iface)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotNil(addrs)
		})
	}
}

func (s *LinodePublicTestSuite) TestInterface() {
	c := linode.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "linode"},
		{"Category", c.Category(), "cloud"},
		{"DefaultEnabled", c.DefaultEnabled(), false},
		{"Dependencies", c.Dependencies(), []string{"hostname"}},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.got)
		})
	}
}

func (s *LinodePublicTestSuite) TestCollect() {
	tests := []struct {
		name      string
		apt       string
		noApt     bool
		hostname  *hostname.Info // when set, added to prior under "hostname"
		lookups   map[string][]net.Addr
		lookupErr error
		wantNil   bool
		verify    func(s *LinodePublicTestSuite, info *linode.Info)
	}{
		{
			name:     "FQDN contains 'linode' triggers detection without apt",
			apt:      "deb http://archive.ubuntu.com/ubuntu focal main",
			hostname: &hostname.Info{FQDN: "myhost.members.linode.com"},
			lookups: map[string][]net.Addr{
				"eth0": {mustCIDR(s, "50.1.2.3/24")},
			},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:     "Domain contains 'linode' triggers detection",
			apt:      "deb http://archive.ubuntu.com/ubuntu focal main",
			hostname: &hostname.Info{Domain: "members.linode.com"},
			lookups:  map[string][]net.Addr{"eth0": {mustCIDR(s, "50.1.2.3/24")}},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:     "hostname without linode + no apt → nil",
			apt:      "deb http://archive.ubuntu.com/ubuntu focal main",
			hostname: &hostname.Info{FQDN: "host.example.com", Domain: "example.com"},
			wantNil:  true,
		},
		{
			name:     "nil hostname Info is tolerated",
			apt:      "linode",
			hostname: nil,
			lookups:  map[string][]net.Addr{"eth0": {mustCIDR(s, "50.1.2.3/24")}},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name: "apt says linode, eth0 and eth0:1 populated (link-local skipped)",
			apt:  "deb http://mirrors.linode.com/ focal main",
			lookups: map[string][]net.Addr{
				// Link-local comes first — firstIPv4 must skip it
				// and pick the routable address that follows.
				"eth0":   {mustCIDR(s, "169.254.0.1/16"), mustCIDR(s, "50.1.2.3/24")},
				"eth0:1": {mustCIDR(s, "10.0.0.5/16")},
			},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
				s.Equal("50.1.2.3", info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name: "default interface lookup runs on production code path",
			apt:  "linode",
			// No lookups → SetInterfaceAddrs is skipped below so
			// defaultInterfaceAddrs executes. eth0/eth0:1 almost
			// certainly don't exist in the test environment, so the
			// result is just an empty Info — we only care that the
			// code path is exercised.
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
			},
		},
		{
			name:    "no linode in apt → nil",
			apt:     "deb http://archive.ubuntu.com/ubuntu focal main",
			wantNil: true,
		},
		{
			name:    "missing apt file → nil",
			noApt:   true,
			wantNil: true,
		},
		{
			name: "eth0 missing produces empty public_ip",
			apt:  "linode",
			lookups: map[string][]net.Addr{
				"eth0:1": {mustCIDR(s, "10.0.0.5/16")},
			},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
				s.Empty(info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name: "IPv6-only interface returns empty",
			apt:  "linode",
			lookups: map[string][]net.Addr{
				"eth0": {mustCIDR(s, "fe80::1/64")},
			},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
				s.Empty(info.PublicIP)
			},
		},
		{
			name: "non-IPNet address type skipped",
			apt:  "linode",
			lookups: map[string][]net.Addr{
				// Use a net.Addr that isn't *net.IPNet — e.g., a
				// net.IPAddr — to hit the type-assert-miss branch.
				"eth0": {&net.IPAddr{IP: net.ParseIP("50.1.2.3")}},
			},
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
				s.Require().NotNil(info)
				s.Empty(info.PublicIP)
			},
		},
		{
			name:      "interface lookup error returns empty IP",
			apt:       "linode",
			lookupErr: errors.New("no such interface"),
			verify: func(s *LinodePublicTestSuite, info *linode.Info) {
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

			if tt.lookups != nil || tt.lookupErr != nil {
				defer linode.SetInterfaceAddrs(func(name string) ([]net.Addr, error) {
					if tt.lookupErr != nil {
						return nil, tt.lookupErr
					}
					addrs, ok := tt.lookups[name]
					if !ok {
						return nil, errors.New("not found")
					}
					return addrs, nil
				})()
			}

			var prior collector.PriorResults
			if tt.hostname != nil {
				prior = collector.PriorResults{"hostname": tt.hostname}
			}
			c := linode.New()
			out, err := c.Collect(context.Background(), prior)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(out)
				return
			}
			info, ok := out.(*linode.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info)
			}
		})
	}
}
