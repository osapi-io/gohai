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
		name         string
		facts        *gohai.Facts
		validateFunc func(*gohai.Cloud)
	}{
		{
			name:  "no provider returns nil",
			facts: &gohai.Facts{},
			validateFunc: func(got *gohai.Cloud) {
				s.Nil(got)
			},
		},
		{
			name:  "ec2 returns aws",
			facts: &gohai.Facts{Ec2: &ec2.Info{ID: "i-abc"}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudAWS, got.Name)
			},
		},
		{
			name:  "gce returns gce",
			facts: &gohai.Facts{Gce: &gce.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudGCE, got.Name)
			},
		},
		{
			name:  "azure returns azure",
			facts: &gohai.Facts{Azure: &azure.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudAzure, got.Name)
			},
		},
		{
			name:  "digital_ocean returns digital_ocean",
			facts: &gohai.Facts{DigitalOcean: &digitalocean.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudDigitalOcean, got.Name)
			},
		},
		{
			name:  "oci returns oci",
			facts: &gohai.Facts{OCI: &oci.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudOCI, got.Name)
			},
		},
		{
			name:  "alibaba returns alibaba",
			facts: &gohai.Facts{Alibaba: &alibaba.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudAlibaba, got.Name)
			},
		},
		{
			name:  "linode returns linode",
			facts: &gohai.Facts{Linode: &linode.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudLinode, got.Name)
			},
		},
		{
			name:  "openstack returns openstack",
			facts: &gohai.Facts{OpenStack: &openstack.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudOpenStack, got.Name)
			},
		},
		{
			name:  "scaleway returns scaleway",
			facts: &gohai.Facts{Scaleway: &scaleway.Info{}},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudScaleway, got.Name)
			},
		},
		{
			name: "first-match wins (ec2 before gce)",
			facts: &gohai.Facts{
				Ec2: &ec2.Info{},
				Gce: &gce.Info{},
			},
			validateFunc: func(got *gohai.Cloud) {
				s.Require().NotNil(got)
				s.Equal(gohai.CloudAWS, got.Name)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.facts.Cloud())
		})
	}
}
