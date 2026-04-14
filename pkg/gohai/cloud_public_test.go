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

package gohai_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
)

type CloudPublicTestSuite struct {
	suite.Suite
}

func TestCloudPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CloudPublicTestSuite))
}

func (s *CloudPublicTestSuite) TestCloud() {
	tests := []struct {
		name  string
		facts *gohai.Facts
		want  string
	}{
		{
			name:  "no provider returns nil",
			facts: &gohai.Facts{},
			want:  "",
		},
		{
			name:  "ec2 returns aws",
			facts: &gohai.Facts{Ec2: &ec2.Info{InstanceID: "i-abc"}},
			want:  gohai.CloudAWS,
		},
		{
			name:  "gce returns gce",
			facts: &gohai.Facts{Gce: &gce.Info{}},
			want:  gohai.CloudGCE,
		},
		{
			name:  "azure returns azure",
			facts: &gohai.Facts{Azure: &azure.Info{}},
			want:  gohai.CloudAzure,
		},
		{
			name:  "digital_ocean returns digital_ocean",
			facts: &gohai.Facts{DigitalOcean: &digitalocean.Info{}},
			want:  gohai.CloudDigitalOcean,
		},
		{
			name:  "oci returns oci",
			facts: &gohai.Facts{OCI: &oci.Info{}},
			want:  gohai.CloudOCI,
		},
		{
			name:  "alibaba returns alibaba",
			facts: &gohai.Facts{Alibaba: &alibaba.Info{}},
			want:  gohai.CloudAlibaba,
		},
		{
			name:  "linode returns linode",
			facts: &gohai.Facts{Linode: &linode.Info{}},
			want:  gohai.CloudLinode,
		},
		{
			name:  "openstack returns openstack",
			facts: &gohai.Facts{OpenStack: &openstack.Info{}},
			want:  gohai.CloudOpenStack,
		},
		{
			name:  "scaleway returns scaleway",
			facts: &gohai.Facts{Scaleway: &scaleway.Info{}},
			want:  gohai.CloudScaleway,
		},
		{
			name: "first-match wins (ec2 before gce)",
			facts: &gohai.Facts{
				Ec2: &ec2.Info{},
				Gce: &gce.Info{},
			},
			want: gohai.CloudAWS,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.facts.Cloud()
			if tt.want == "" {
				s.Nil(got)
				return
			}
			s.Require().NotNil(got)
			s.Equal(tt.want, got.Name)
		})
	}
}
