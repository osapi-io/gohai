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

package gohai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
)

type GohaiTestSuite struct {
	suite.Suite
}

func TestGohaiTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GohaiTestSuite))
}

type cycleCollector struct {
	name string
	deps []string
}

func (c *cycleCollector) Name() string {
	return c.name
}

func (c *cycleCollector) Category() string {
	return "misc"
}

func (c *cycleCollector) DefaultEnabled() bool {
	return true
}

func (c *cycleCollector) Dependencies() []string {
	return c.deps
}

func (c *cycleCollector) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	return nil, nil
}

func (s *GohaiTestSuite) TestFactsSet() {
	tests := []struct {
		name   string
		key    string
		value  any
		verify func(s *GohaiTestSuite, f *Facts)
	}{
		{
			name:  "gce populates Facts.Gce",
			key:   "gce",
			value: &gce.Info{Name: "vm-1"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Gce)
				s.Equal("vm-1", f.Gce.Name)
			},
		},
		{
			name:  "ec2 populates Facts.Ec2",
			key:   "ec2",
			value: &ec2.Info{InstanceID: "i-abc"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Ec2)
				s.Equal("i-abc", f.Ec2.InstanceID)
			},
		},
		{
			name:   "ec2 wrong type ignored",
			key:    "ec2",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.Ec2) },
		},
		{
			name:  "azure populates Facts.Azure",
			key:   "azure",
			value: &azure.Info{VMID: "vm"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Azure)
				s.Equal("vm", f.Azure.VMID)
			},
		},
		{
			name:   "azure wrong type ignored",
			key:    "azure",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.Azure) },
		},
		{
			name:  "digital_ocean populates Facts.DigitalOcean",
			key:   "digital_ocean",
			value: &digitalocean.Info{DropletID: 1},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.DigitalOcean)
				s.Equal(int64(1), f.DigitalOcean.DropletID)
			},
		},
		{
			name:   "digital_ocean wrong type ignored",
			key:    "digital_ocean",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.DigitalOcean) },
		},
		{
			name:  "oci populates Facts.OCI",
			key:   "oci",
			value: &oci.Info{ID: "ocid1"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.OCI)
				s.Equal("ocid1", f.OCI.ID)
			},
		},
		{
			name:   "oci wrong type ignored",
			key:    "oci",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.OCI) },
		},
		{
			name:  "alibaba populates Facts.Alibaba",
			key:   "alibaba",
			value: &alibaba.Info{InstanceID: "i-ali"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Alibaba)
				s.Equal("i-ali", f.Alibaba.InstanceID)
			},
		},
		{
			name:   "alibaba wrong type ignored",
			key:    "alibaba",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.Alibaba) },
		},
		{
			name:  "linode populates Facts.Linode",
			key:   "linode",
			value: &linode.Info{PublicIP: "1.2.3.4"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Linode)
				s.Equal("1.2.3.4", f.Linode.PublicIP)
			},
		},
		{
			name:   "linode wrong type ignored",
			key:    "linode",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.Linode) },
		},
		{
			name:  "openstack populates Facts.OpenStack",
			key:   "openstack",
			value: &openstack.Info{InstanceID: "i-os"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.OpenStack)
				s.Equal("i-os", f.OpenStack.InstanceID)
			},
		},
		{
			name:   "openstack wrong type ignored",
			key:    "openstack",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.OpenStack) },
		},
		{
			name:  "scaleway populates Facts.Scaleway",
			key:   "scaleway",
			value: &scaleway.Info{ID: "sc-1"},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.Scaleway)
				s.Equal("sc-1", f.Scaleway.ID)
			},
		},
		{
			name:   "scaleway wrong type ignored",
			key:    "scaleway",
			value:  "x",
			verify: func(s *GohaiTestSuite, f *Facts) { s.Nil(f.Scaleway) },
		},
		{
			name:  "dmi populates Facts.DMI",
			key:   "dmi",
			value: &dmi.Info{Product: &dmi.Product{Name: "Google Compute Engine"}},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Require().NotNil(f.DMI)
				s.Equal("Google Compute Engine", f.DMI.Product.Name)
			},
		},
		{
			name:  "mismatched type for gce is silently ignored",
			key:   "gce",
			value: "not-gce-info",
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Nil(f.Gce)
			},
		},
		{
			name:  "mismatched type for dmi is silently ignored",
			key:   "dmi",
			value: 42,
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Nil(f.DMI)
			},
		},
		{
			name:  "unknown key is a no-op",
			key:   "nope",
			value: &dmi.Info{},
			verify: func(s *GohaiTestSuite, f *Facts) {
				s.Nil(f.DMI)
				s.Nil(f.Gce)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := &Facts{}
			f.set(tt.key, tt.value)
			tt.verify(s, f)
		})
	}
}

func (s *GohaiTestSuite) TestFactsCountPopulated() {
	tests := []struct {
		name  string
		facts *Facts
		want  int
	}{
		{"empty", &Facts{}, 0},
		{"gce only", &Facts{Gce: &gce.Info{}}, 1},
		{"dmi only", &Facts{DMI: &dmi.Info{}}, 1},
		{"gce + dmi", &Facts{Gce: &gce.Info{}, DMI: &dmi.Info{}}, 2},
		{"ec2", &Facts{Ec2: &ec2.Info{}}, 1},
		{"azure", &Facts{Azure: &azure.Info{}}, 1},
		{"digital_ocean", &Facts{DigitalOcean: &digitalocean.Info{}}, 1},
		{"oci", &Facts{OCI: &oci.Info{}}, 1},
		{"alibaba", &Facts{Alibaba: &alibaba.Info{}}, 1},
		{"linode", &Facts{Linode: &linode.Info{}}, 1},
		{"openstack", &Facts{OpenStack: &openstack.Info{}}, 1},
		{"scaleway", &Facts{Scaleway: &scaleway.Info{}}, 1},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.facts.countPopulated())
		})
	}
}

func (s *GohaiTestSuite) TestCollectPropagatesRunError() {
	reg := collector.NewRegistry()
	s.Require().NoError(reg.Register(&cycleCollector{name: "a", deps: []string{"b"}}))
	s.Require().NoError(reg.Register(&cycleCollector{name: "b", deps: []string{"a"}}))

	g := &Gohai{registry: reg}
	sel, err := reg.Selected(nil, nil)
	s.Require().NoError(err)
	g.selected = sel

	_, err = g.Collect(context.Background())
	s.Error(err)
}
