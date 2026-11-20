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

package family_test

import (
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/family"
)

type FamilyPublicTestSuite struct {
	suite.Suite
}

func (s *FamilyPublicTestSuite) TestPlatform() {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "when the ID is remapped", id: "amzn", want: "amazon"},
		{name: "when rhel is remapped to redhat", id: "rhel", want: "redhat"},
		{name: "when the ID is in upper case", id: "OL", want: "oracle"},
		{name: "when the ID is its own platform", id: "ubuntu", want: "ubuntu"},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Equal(tc.want, family.Platform(tc.id))
		})
	}
}

// TestOf holds the table to Ohai's platform_family_from_platform: one
// platform per match, and the cases where Ohai's order decides.
func (s *FamilyPublicTestSuite) TestOf() {
	tests := map[string]string{
		"ubuntu": family.Debian, "debian": family.Debian, "linuxmint": family.Debian,
		"raspbian": family.Debian, "cumulus": family.Debian, "kali": family.Debian,
		"pop":    family.Debian,
		"centos": family.RHEL, "redhat": family.RHEL, "oracle": family.RHEL,
		"almalinux": family.RHEL, "rocky": family.RHEL, "scientific": family.RHEL,
		"enterpriseenterprise": family.RHEL, "xenserver": family.RHEL,
		"xcp-ng": family.RHEL, "cloudlinux": family.RHEL, "alibabalinux": family.RHEL,
		"sangoma": family.RHEL, "clearos": family.RHEL, "parallels": family.RHEL,
		"ibm_powerkvm": family.RHEL, "bigip": family.RHEL, "virtuozzo": family.RHEL,
		// centos comes before nexus in Ohai's order.
		"nexus_centos": family.RHEL,
		"amazon":       family.Amazon,
		"suse":         family.SUSE, "opensuseleap": family.SUSE, "sles": family.SUSE,
		"sled": family.SUSE, "sle-micro": family.SUSE, "sle_hpc": family.SUSE,
		"fedora": family.Fedora, "arista_eos": family.Fedora,
		"nexus": family.WRLinux, "ios_xr": family.WRLinux,
		"gentoo": family.Gentoo,
		"arch":   family.Arch, "manjaro": family.Arch,
		"exherbo":    family.Exherbo,
		"alpine":     family.Alpine,
		"clearlinux": family.ClearLinux,
		"mangeia":    family.Mandriva, "mageia": family.Mandriva,
		"slackware": family.Slackware,
		"darwin":    family.Darwin,
		"Ubuntu":    family.Debian,
		"nixos":     "",
	}
	for platform, want := range tests {
		s.Run(platform, func() {
			s.Equal(want, family.Of(platform))
		})
	}
}

func (s *FamilyPublicTestSuite) TestDetect() {
	tests := []struct {
		name string
		goos string
		info *host.InfoStat
		err  error
		want string
	}{
		{name: "when the host is macOS", goos: "darwin", want: family.Darwin},
		{
			name: "when the platform has a family", goos: "linux",
			info: &host.InfoStat{Platform: "amzn"}, want: family.Amazon,
		},
		{
			name: "when the platform has no family", goos: "linux",
			info: &host.InfoStat{Platform: "NixOS"}, want: "nixos",
		},
		{
			name: "when the platform is empty", goos: "linux",
			info: &host.InfoStat{}, want: "",
		},
		{
			name: "when the host cannot be read", goos: "linux",
			err: errors.New("boom"), want: "",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			defer family.SetGOOS(tc.goos)()
			defer family.SetHostInfoFn(func() (*host.InfoStat, error) {
				return tc.info, tc.err
			})()
			s.Equal(tc.want, family.Detect())
		})
	}
}

func TestFamilyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FamilyPublicTestSuite))
}
