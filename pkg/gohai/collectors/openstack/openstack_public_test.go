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

package openstack_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
)

func osPrior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{Product: &dmi.Product{Name: "OpenStack Nova"}},
	}
}

var ec2Responses = map[string]string{
	"/latest/meta-data/ami-id":                      "ami-xxx",
	"/latest/meta-data/instance-id":                 "i-abc",
	"/latest/meta-data/instance-type":               "m1.small",
	"/latest/meta-data/hostname":                    "prod-1",
	"/latest/meta-data/local-hostname":              "prod-1.local",
	"/latest/meta-data/local-ipv4":                  "10.0.0.5",
	"/latest/meta-data/public-hostname":             "prod-1.public",
	"/latest/meta-data/public-ipv4":                 "10.0.0.6",
	"/latest/meta-data/placement/availability-zone": "nova",
	"/latest/meta-data/reservation-id":              "r-abc",
	"/latest/meta-data/kernel-id":                   "aki-xxx",
	"/latest/meta-data/ramdisk-id":                  "ari-xxx",
	"/latest/meta-data/security-groups":             "default\nssh",
}

const novaDoc = `{
  "uuid": "uuid-xxx",
  "name": "prod-1",
  "project_id": "proj-1",
  "hostname": "prod-1",
  "availability_zone": "nova",
  "meta": {"owner": "sre"},
  "public_keys": {"default": "ssh-rsa AAA..."}
}`

type OpenStackPublicTestSuite struct {
	suite.Suite
}

func TestOpenStackPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OpenStackPublicTestSuite))
}

func (s *OpenStackPublicTestSuite) TestInterface() {
	c := openstack.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "openstack"},
		{"Category", c.Category(), "cloud"},
		{"DefaultEnabled", c.DefaultEnabled(), false},
		{"Dependencies", c.Dependencies(), []string{"dmi"}},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.got)
		})
	}
}

func canned(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/openstack/latest/meta_data.json" {
		_, _ = w.Write([]byte(novaDoc))
		return
	}
	body, ok := ec2Responses[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(body))
}

func (s *OpenStackPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		prior      collector.PriorResults
		handler    func(w http.ResponseWriter, r *http.Request)
		closed     bool
		wantNil    bool
		wantNoHTTP bool
		verify     func(s *OpenStackPublicTestSuite, info *openstack.Info)
	}{
		{
			name:    "happy path combines ec2 mirror + nova doc",
			handler: canned,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
				s.Equal("m1.small", info.InstanceType)
				s.Equal("10.0.0.5", info.LocalIPv4)
				s.Equal("nova", info.AvailabilityZone)
				s.Equal([]string{"default", "ssh"}, info.SecurityGroups)
				s.Equal("uuid-xxx", info.UUID)
				s.Equal("proj-1", info.ProjectID)
				s.Equal("sre", info.MetaData["owner"])
			},
		},
		{
			name: "dmi says not OpenStack short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{Name: "VMware"}},
			},
			handler:    canned,
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:    "no dmi in prior fails open",
			prior:   collector.PriorResults{},
			handler: canned,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name:    "first-path 404 drops silently",
			handler: func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
			wantNil: true,
		},
		{
			name: "ec2 paths partial 404 tolerated, nova doc fills uuid",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/latest/meta-data/ami-id":
					_, _ = w.Write([]byte("ami-yyy"))
				case "/openstack/latest/meta_data.json":
					_, _ = w.Write([]byte(`{"uuid": "uuid-solo", "hostname": "h"}`))
				default:
					http.NotFound(w, r)
				}
			},
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Equal("uuid-solo", info.UUID)
				s.Equal("h", info.Hostname)
			},
		},
		{
			name: "nova doc missing is tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/openstack/latest/meta_data.json" {
					http.NotFound(w, r)
					return
				}
				canned(w, r)
			},
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Empty(info.UUID)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name: "nova doc returns malformed JSON and is tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/openstack/latest/meta_data.json" {
					_, _ = w.Write([]byte("not json"))
					return
				}
				canned(w, r)
			},
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Empty(info.UUID)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name:    "connection refused drops silently",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var httpCalled bool
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled = true
					tt.handler(w, r)
				},
			))
			if tt.closed {
				srv.Close()
			} else {
				defer srv.Close()
			}

			client := cloudmetadata.New(srv.URL)
			c := openstack.NewWithClient(client)

			prior := tt.prior
			if prior == nil {
				prior = osPrior()
			}
			out, err := c.Collect(context.Background(), prior)
			s.Require().NoError(err)

			if tt.wantNoHTTP {
				s.False(httpCalled)
			}
			if tt.wantNil {
				s.Nil(out)
				return
			}
			info, ok := out.(*openstack.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info)
			}
		})
	}
}
