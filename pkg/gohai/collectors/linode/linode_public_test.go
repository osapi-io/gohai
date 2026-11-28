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
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

// ubuntuApt is a sources.list that does not name linode.
const ubuntuApt = "deb http://archive.ubuntu.com/ubuntu focal main"

// linodeApt is a sources.list with Linode's mirror.
var linodeApt = map[string]string{
	"/etc/apt/sources.list": "deb http://mirrors.linode.com/ focal main",
}

type LinodePublicTestSuite struct {
	suite.Suite
}

func TestLinodePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LinodePublicTestSuite))
}

func (s *LinodePublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		collector    func() *linode.Collector
		validateFunc func(*linode.Collector)
	}{
		{
			name:      "New gives the linode cloud collector, opt-in, after dmi and hostname",
			collector: linode.New,
			validateFunc: func(c *linode.Collector) {
				s.Equal("linode", c.Name())
				s.Equal(collector.CategoryCloud, c.Category())
				s.False(c.DefaultEnabled())
				s.Equal([]string{"dmi", "hostname"}, c.Dependencies())
				s.NotNil(c.FS)
				s.NotNil(c.Exec)
			},
		},
		{
			name: "NewWith reads through the given sources",
			collector: func() *linode.Collector {
				return linode.NewWith(source.Sources{
					FS:     memfs.New(),
					Exec:   sourcemocks.NewMockRunner(gomock.NewController(s.T())),
					Family: family.Debian,
				})
			},
			validateFunc: func(c *linode.Collector) {
				s.IsType(&memfs.MemFS{}, c.FS)
				s.IsType(&sourcemocks.MockRunner{}, c.Exec)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.collector())
		})
	}
}

func (s *LinodePublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		files        map[string]string
		prior        collector.PriorResults
		ip           string // the output of ip -o -4 addr show
		ipErr        error
		nilExec      bool
		validateFunc func(any)
	}{
		{
			name:  "with no Exec, a Linode has no addresses",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "myhost.members.linode.com"},
			},
			nilExec: true,
			validateFunc: func(out any) {
				s.Equal(&linode.Info{}, out)
			},
		},
		{
			name:  "FQDN contains 'linode' triggers detection without apt",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "myhost.members.linode.com"},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:  "Domain contains 'linode' triggers detection",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{Domain: "members.linode.com"},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:  "hostname without linode + no apt → nil",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "host.example.com", Domain: "example.com"},
			},
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name:  "a nil hostname Info is no signal",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{"hostname": (*hostname.Info)(nil)},
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name: "the DMI vendor Linode triggers detection with no apt file",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{VendorName: "Linode"}},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name: "another DMI vendor is no signal",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{VendorName: "QEMU"}},
			},
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name: "a deb822 file in sources.list.d that names linode triggers detection",
			files: map[string]string{
				"/etc/apt/sources.list.d/ubuntu.sources": "Types: deb\nURIs: http://archive.ubuntu.com/ubuntu\n",
				"/etc/apt/sources.list.d/linode.sources": "Types: deb\nURIs: http://mirrors.linode.com/ubuntu\n",
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name: "sources.list.d without linode is no signal",
			files: map[string]string{
				"/etc/apt/sources.list.d/ubuntu.sources": "Types: deb\nURIs: http://archive.ubuntu.com/ubuntu\n",
				"/etc/apt/sources.list.d/sub/x.list":     "deb http://mirrors.linode.com/ focal main",
			},
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name:  "apt says linode, eth0 and the eth0:1 alias populated (link-local skipped)",
			files: linodeApt,
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
				s.Equal("50.1.2.3", info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name:  "no linode in apt → nil",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name: "missing apt file → nil",
			validateFunc: func(out any) {
				s.Nil(out)
			},
		},
		{
			name:  "eth0 missing produces empty public_ip",
			files: linodeApt,
			ip:    ipLine("eth0", "eth0:1", "10.0.0.5/17"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Empty(info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name:  "lines that are not an IPv4 address are skipped",
			files: linodeApt,
			ip: "\n" +
				"1: lo    inet\n" +
				"2: eth0    inet not-a-cidr scope global eth0\\ valid_lft forever\n" +
				"2: eth0    inet6 fe80::1/64 scope link\\ valid_lft forever\n",
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Empty(info.PublicIP)
				s.Empty(info.PrivateIP)
			},
		},
		{
			name:  "an ip error returns empty IPs",
			files: linodeApt,
			ipErr: errors.New("ip: not found"),
			validateFunc: func(out any) {
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Empty(info.PublicIP)
				s.Empty(info.PrivateIP)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			m := sourcemocks.NewMockRunner(gomock.NewController(s.T()))
			m.EXPECT().
				Execute(gomock.Any(), "ip", "-o", "-4", "addr", "show").
				Return([]byte(tt.ip), tt.ipErr).
				AnyTimes()

			var c collector.Collector = linode.NewWith(source.Sources{
				FS:     sourcetest.FS(s.T(), tt.files),
				Exec:   m,
				Family: family.Debian,
			})
			if tt.nilExec {
				c = &linode.Collector{FS: sourcetest.FS(s.T(), tt.files)}
			}
			out, err := c.Collect(context.Background(), tt.prior)
			s.Require().NoError(err)
			tt.validateFunc(out)
		})
	}
}

// ipLine is one line of `ip -o -4 addr show`: the address cidr on dev,
// with label, which is dev or an alias such as eth0:1.
func ipLine(
	dev string,
	label string,
	cidr string,
) string {
	return "2: " + dev + "    inet " + cidr + " scope global " + label +
		"\\       valid_lft forever preferred_lft forever\n"
}
