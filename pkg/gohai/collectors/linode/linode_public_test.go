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
	"os/exec"
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
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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
		collector    func() linode.Collector
		validateFunc func(linode.Collector)
	}{
		{
			name:      "New gives the linode cloud collector, opt-in, after dmi and hostname",
			collector: linode.New,
			validateFunc: func(c linode.Collector) {
				s.Equal("linode", c.Name())
				s.Equal(collector.CategoryCloud, c.Category())
				s.False(c.DefaultEnabled())
				s.Equal([]string{"dmi", "hostname"}, c.Dependencies())
				s.NotNil(c.(*linode.Host).FS)
				s.NotNil(c.(*linode.Host).Exec)
			},
		},
		{
			name: "NewWith reads through the given sources",
			collector: func() linode.Collector {
				return linode.NewWith(source.Sources{
					FS:     memfs.New(),
					Exec:   mocks.NewMockRunner(gomock.NewController(s.T())),
					Family: family.Debian,
				})
			},
			validateFunc: func(c linode.Collector) {
				s.IsType(&memfs.MemFS{}, c.(*linode.Host).FS)
				s.IsType(&mocks.MockRunner{}, c.(*linode.Host).Exec)
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
		nilFS        bool
		validateFunc func(any, error)
	}{
		{
			name:  "with no Exec, a Linode has no addresses",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "myhost.members.linode.com"},
			},
			nilExec: true,
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Equal(&linode.Info{}, out)
			},
		},
		{
			name:  "an FQDN in the linode.com domain triggers detection without apt",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "myhost.members.linode.com"},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:  "a Domain in linode.com triggers detection",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{
				"hostname": &hostname.Info{Domain: "members.linode.com"},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
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
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name:  "a nil hostname Info is no signal",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			prior: collector.PriorResults{"hostname": (*hostname.Info)(nil)},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name: "the DMI vendor Linode triggers detection with no apt file",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{VendorName: "Linode"}},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
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
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name:  "a host that DMI says is not a Linode is not one, though it uses mirrors.linode.com",
			files: linodeApt,
			prior: collector.PriorResults{
				"dmi":      &dmi.Info{Product: &dmi.Product{VendorName: "QEMU"}},
				"hostname": &hostname.Info{FQDN: "box.members.linode.com"},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name: "the DMI vendor Akamai triggers detection",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{VendorName: "Akamai"}},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:  "dmi with no product table is not a Linode",
			files: linodeApt,
			prior: collector.PriorResults{"dmi": &dmi.Info{}},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name: "a name that only contains linode is not in the linode.com domain",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{
					FQDN:   "linode-builder.example.com",
					Domain: "example.com",
				},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name: "a FQDN of notlinode.com is not in the linode.com domain",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "notlinode.com"},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, out)
			},
		},
		{
			name: "a FQDN of linode.com.evil is not in the linode.com domain",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "linode.com.evil"},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, out)
			},
		},
		{
			name: "a Domain of notlinode.com is not in the linode.com domain",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{Domain: "notlinode.com"},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, out)
			},
		},
		{
			name: "a Domain of linode.com.evil is not in the linode.com domain",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{Domain: "linode.com.evil"},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, out)
			},
		},
		{
			name: "a fully qualified name with a trailing dot is in the domain",
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "li1-2.members.linode.com."},
			},
			ip: ipLine("eth0", "eth0", "50.1.2.3/24"),
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
			},
		},
		{
			name:  "with no FS, apt is no signal",
			nilFS: true,
			prior: collector.PriorResults{
				"hostname": &hostname.Info{FQDN: "host.example.com"},
			},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
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
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
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
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
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
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Equal("50.1.2.3", info.PublicIP)
				s.Equal("10.0.0.5", info.PrivateIP)
			},
		},
		{
			name:  "no linode in apt → nil",
			files: map[string]string{"/etc/apt/sources.list": ubuntuApt},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name: "missing apt file → nil",
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				s.Nil(out)
			},
		},
		{
			name:  "eth0 missing produces empty public_ip",
			files: linodeApt,
			ip:    ipLine("eth0", "eth0:1", "10.0.0.5/17"),
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
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
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Empty(info.PublicIP)
				s.Empty(info.PrivateIP)
			},
		},
		{
			name:  "an absent ip gives empty IPs",
			files: linodeApt,
			ipErr: &exec.Error{Name: "ip", Err: exec.ErrNotFound},
			validateFunc: func(out any, err error) {
				s.Require().NoError(err)
				info, ok := out.(*linode.Info)
				s.Require().True(ok)
				s.Empty(info.PublicIP)
				s.Empty(info.PrivateIP)
			},
		},
		{
			name:  "an ip that runs and fails is an error",
			files: linodeApt,
			ipErr: errors.New("exit status 1"),
			validateFunc: func(out any, err error) {
				s.EqualError(err, "linode: ip addr: exit status 1")
				s.Nil(out)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			m := mocks.NewMockRunner(gomock.NewController(s.T()))
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
				c = &linode.Host{FS: sourcetest.FS(s.T(), tt.files)}
			}
			if tt.nilFS {
				c = &linode.Host{Exec: m}
			}
			out, err := c.Collect(context.Background(), tt.prior)
			tt.validateFunc(out, err)
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
