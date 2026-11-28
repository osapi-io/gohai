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

package dns_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dns"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*dns.Linux)(nil)
	_ collector.Collector = (*dns.Darwin)(nil)
)

type DNSPublicTestSuite struct {
	suite.Suite
}

func TestDNSPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DNSPublicTestSuite))
}

func (s *DNSPublicTestSuite) TestNew() {
	tests := []struct {
		name   string
		family string
		darwin bool
	}{
		{name: "darwin dispatches to Darwin", family: "darwin", darwin: true},
		{name: "debian dispatches to Linux", family: "debian"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := dns.NewWith(source.Sources{Family: tt.family})
			_, isDarwin := c.(*dns.Darwin)
			s.Equal(tt.darwin, isDarwin)
			s.Equal("dns", c.Name())
			s.Equal(collector.CategoryNetwork, c.Category())
			s.False(c.DefaultEnabled())
			s.Nil(c.Dependencies())
		})
	}
	s.NotNil(dns.New())
}

func (s *DNSPublicTestSuite) TestCollect() {
	const resolvConf = "# generated\n; also a comment\nnameserver 10.0.0.2\nnameserver fd00::53\n" +
		"domain old.example\nsearch corp.example example.com\noptions edns0 trust-ad\noptions ndots:2\nbad\n"
	tests := []struct {
		name         string
		variant      string
		files        map[string]string
		runner       sourcetest.Commands
		nilExec      bool
		validateFunc func(any, error)
	}{
		{
			name:    "linux: resolv.conf and resolvectl links",
			variant: "linux",
			files: map[string]string{
				"/etc/resolv.conf":                 resolvConf,
				"/run/systemd/resolve/resolv.conf": "",
			},
			runner: sourcetest.Commands{Out: map[string]string{
				"resolvectl dns":    "Global: 1.1.1.1\nLink 2 (eth0): 10.0.0.1 fe80::1%eth0\nLink 3 (wg0):\n",
				"resolvectl domain": "Global:\nLink 2 (eth0): corp.example ~.\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{
					Nameservers: []string{"10.0.0.2", "fd00::53"},
					Search:      []string{"corp.example", "example.com"},
					Options:     []string{"edns0", "trust-ad", "ndots:2"},
					Interfaces: map[string]dns.Resolver{
						"eth0": {
							Nameservers: []string{"10.0.0.1", "fe80::1"},
							Domains:     []string{"corp.example", "~."},
						},
						"wg0": {},
					},
					Global: &dns.Resolver{Nameservers: []string{"1.1.1.1"}},
				}, got)
			},
		},
		{
			name:    "linux: DNS over TLS servers lose their port and name, and Global domains are read",
			variant: "linux",
			files:   map[string]string{"/run/systemd/resolve/resolv.conf": ""},
			runner: sourcetest.Commands{Out: map[string]string{
				"resolvectl dns": "Global: 1.1.1.1:853#cloudflare-dns.com [2606:4700::1111]:853#cloudflare-dns.com\n" +
					"Link 2 (eth0): 10.0.0.1:53 fe80::1%eth0\n",
				"resolvectl domain": "Global: ~. corp.example\nLink 2 (eth0):\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{
					Interfaces: map[string]dns.Resolver{
						"eth0": {Nameservers: []string{"10.0.0.1", "fe80::1"}},
					},
					Global: &dns.Resolver{
						Nameservers: []string{"1.1.1.1", "2606:4700::1111"},
						Domains:     []string{"~.", "corp.example"},
					},
				}, got)
			},
		},
		{
			name:    "linux: no resolv.conf, and systemd-resolved is not running",
			variant: "linux",
			runner: sourcetest.Commands{Err: map[string]error{
				"resolvectl dns":    errors.New("exit status 1"),
				"resolvectl domain": errors.New("exit status 1"),
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{}, got)
			},
		},
		{
			name:    "linux: systemd-resolved runs, but the host has no resolvectl",
			variant: "linux",
			files: map[string]string{
				"/etc/resolv.conf":                 "nameserver 127.0.0.53\n",
				"/run/systemd/resolve/resolv.conf": "",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{Nameservers: []string{"127.0.0.53"}}, got)
			},
		},
		{
			name:    "linux: systemd-resolved runs, and resolvectl fails, keeps resolv.conf",
			variant: "linux",
			files: map[string]string{
				"/run/systemd/resolve/resolv.conf": "",
				"/etc/resolv.conf":                 "nameserver 1.1.1.1\n",
			},
			runner: sourcetest.Commands{Err: map[string]error{
				"resolvectl dns": errors.New("exit status 1"),
			}},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read resolvectl dns")
				info, ok := got.(*dns.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal([]string{"1.1.1.1"}, info.Nameservers)
			},
		},
		{
			name:    "linux: a cancelled resolvectl gives no fact",
			variant: "linux",
			files: map[string]string{
				"/run/systemd/resolve/resolv.conf": "",
				"/etc/resolv.conf":                 "nameserver 1.1.1.1\n",
			},
			runner: sourcetest.Commands{Err: map[string]error{
				"resolvectl dns": context.Canceled,
			}},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a resolved runtime directory that cannot be checked keeps resolv.conf",
			variant: "linux",
			files: map[string]string{
				"/run/systemd":     "",
				"/etc/resolv.conf": "nameserver 1.1.1.1\n",
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "stat /run/systemd/resolve")
				info, ok := got.(*dns.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal([]string{"1.1.1.1"}, info.Nameservers)
			},
		},
		{
			name:    "linux: a resolv.conf that cannot be read is an error",
			variant: "linux",
			files:   map[string]string{"/etc/resolv.conf/x": ""},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/resolv.conf")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: nil Exec",
			variant: "linux",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{Nameservers: []string{"1.1.1.1"}}, got)
			},
		},
		{
			name:    "darwin: scoped resolvers, the first of each interface, and the default resolver as global",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			runner: sourcetest.Commands{
				Out: map[string]string{"scutil --dns": "DNS configuration\n\n" +
					"resolver #1\n  search domain[0] : ts.example\n  nameserver[0] : 100.100.100.100\n  if_index : 24 (utun6)\n" +
					"  flags    : Supplemental, Request A records\n\n" +
					"resolver #2\n  nameserver[0] : 1.1.1.1\n  if_index : 14 (en0)\n\n" +
					"resolver #3\n  domain   : local\n  nameserver[0] : 224.0.0.251\n\n" +
					"DNS configuration (for scoped queries)\n\n" +
					"resolver #1\n  nameserver[0] : 9.9.9.9\n  if_index : 14 (en0)\n"},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{
					Nameservers: []string{"1.1.1.1"},
					Interfaces: map[string]dns.Resolver{
						"utun6": {
							Nameservers: []string{"100.100.100.100"},
							Domains:     []string{"ts.example"},
						},
						"en0": {Nameservers: []string{"1.1.1.1"}},
					},
					Global: &dns.Resolver{Nameservers: []string{"1.1.1.1"}},
				}, got)
			},
		},
		{
			name:    "darwin: a per-domain first resolver is not the global one",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			runner: sourcetest.Commands{
				Out: map[string]string{"scutil --dns": "DNS configuration\n\n" +
					"resolver #1\n  domain   : local\n  nameserver[0] : 224.0.0.251\n\n" +
					"resolver #2\n  nameserver[0] : 1.1.1.1\n"},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{
					Nameservers: []string{"1.1.1.1"},
					Global:      &dns.Resolver{Nameservers: []string{"1.1.1.1"}},
				}, got)
			},
		},
		{
			name:    "darwin: a scoped resolver is never the global one",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			runner: sourcetest.Commands{
				Out: map[string]string{
					"scutil --dns": "DNS configuration (for scoped queries)\n\n" +
						"resolver #1\n  nameserver[0] : 9.9.9.9\n  if_index : 14 (en0)\n",
				},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{
					Nameservers: []string{"1.1.1.1"},
					Interfaces:  map[string]dns.Resolver{"en0": {Nameservers: []string{"9.9.9.9"}}},
				}, got)
			},
		},
		{
			name:    "darwin: scutil fails, keeps resolv.conf",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			runner:  sourcetest.Commands{Err: map[string]error{"scutil --dns": errors.New("boom")}},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read scutil --dns")
				info, ok := got.(*dns.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal([]string{"1.1.1.1"}, info.Nameservers)
			},
		},
		{
			name:    "darwin: a cancelled scutil gives no fact",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			runner:  sourcetest.Commands{Err: map[string]error{"scutil --dns": context.Canceled}},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: no scutil on the host",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf": "nameserver 1.1.1.1\n"},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{Nameservers: []string{"1.1.1.1"}}, got)
			},
		},
		{
			name:    "darwin: a resolv.conf that cannot be read is an error",
			variant: "darwin",
			files:   map[string]string{"/etc/resolv.conf/x": ""},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/resolv.conf")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: nil Exec",
			variant: "darwin",
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&dns.Info{}, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			fs := sourcetest.FS(s.T(), tt.files)
			var c dns.Collector
			switch tt.variant {
			case "linux":
				l := &dns.Linux{FS: fs, Exec: tt.runner.Mock(s.T())}
				if tt.nilExec {
					l.Exec = nil
				}
				c = l
			case "darwin":
				d := &dns.Darwin{FS: fs, Exec: tt.runner.Mock(s.T())}
				if tt.nilExec {
					d.Exec = nil
				}
				c = d
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
