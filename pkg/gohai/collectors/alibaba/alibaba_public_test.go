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

package alibaba_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

func aliPrior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{Product: &dmi.Product{Vendor: "Alibaba Cloud"}},
	}
}

// metadataResponses is the per-path canned data the fake server emits.
var metadataResponses = map[string]string{
	"/meta-data/hostname":                   "prod-1",
	"/meta-data/instance-id":                "i-abc",
	"/meta-data/region-id":                  "cn-hangzhou",
	"/meta-data/zone-id":                    "cn-hangzhou-b",
	"/meta-data/image-id":                   "img-1",
	"/meta-data/instance/instance-type":     "ecs.g6.large",
	"/meta-data/instance/instance-name":     "prod-1",
	"/meta-data/instance/max-netbw-ingress": "1048576",
	"/meta-data/instance/max-netbw-egress":  "1048576",
	"/meta-data/mac":                        "00:16:3e:00:00:01",
	"/meta-data/private-ipv4":               "172.16.0.5",
	"/meta-data/eipv4":                      "47.1.2.3",
	"/meta-data/vpc-id":                     "vpc-1",
	"/meta-data/vpc-cidr-block":             "172.16.0.0/16",
	"/meta-data/vswitch-id":                 "vsw-1",
	"/meta-data/vswitch-cidr-block":         "172.16.0.0/24",
	"/meta-data/serial-number":              "sn-xyz",
	"/meta-data/network-type":               "vpc",
	"/meta-data/dns-conf/nameservers":       "100.100.2.136 100.100.2.138",
	"/meta-data/ntp-conf/ntp-servers":       "ntp1.aliyun.com ntp2.aliyun.com",
	"/meta-data/ram/role-name":              "ecs-default",
}

type AlibabaPublicTestSuite struct {
	suite.Suite
}

func TestAlibabaPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AlibabaPublicTestSuite))
}

func (s *AlibabaPublicTestSuite) TestInterface() {
	c := alibaba.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "alibaba"},
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

// canned serves every known path from metadataResponses; missing
// paths return 404.
func canned(w http.ResponseWriter, r *http.Request) {
	body, ok := metadataResponses[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(body))
}

func (s *AlibabaPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		prior      collector.PriorResults
		handler    func(w http.ResponseWriter, r *http.Request)
		closed     bool
		wantNil    bool
		wantErr    bool
		wantNoHTTP bool
		verify     func(s *AlibabaPublicTestSuite, info *alibaba.Info)
	}{
		{
			name:    "happy path populates all fields",
			handler: canned,
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
				s.Equal("prod-1", info.Hostname)
				s.Equal("cn-hangzhou", info.Region)
				s.Equal("cn-hangzhou-b", info.Zone)
				s.Equal("ecs.g6.large", info.InstanceType)
				s.Equal("172.16.0.5", info.PrivateIPv4)
				s.Equal("47.1.2.3", info.PublicIPv4)
				s.Equal("vpc-1", info.VPCID)
				s.Equal("vsw-1", info.VSwitchID)
				s.Equal([]string{"100.100.2.136", "100.100.2.138"}, info.Nameservers)
				s.Equal([]string{"ntp1.aliyun.com", "ntp2.aliyun.com"}, info.NTPServers)
				s.Equal("ecs-default", info.RAMRoleName)
			},
		},
		{
			name: "partial 404s tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/meta-data/hostname" {
					_, _ = w.Write([]byte("only-hostname"))
					return
				}
				http.NotFound(w, r)
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info) {
				s.Require().NotNil(info)
				s.Equal("only-hostname", info.Hostname)
				s.Empty(info.InstanceID)
			},
		},
		{
			name: "dmi says not Alibaba short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{Vendor: "Dell Inc."}},
			},
			handler:    canned,
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:    "no dmi in prior fails open",
			prior:   collector.PriorResults{},
			handler: canned,
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name:    "first probe 404 drops silently",
			handler: func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
			wantNil: true,
		},
		{
			name:    "connection refused drops silently",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			wantNil: true,
		},
		{
			name: "empty dns-conf produces nil nameservers",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/meta-data/hostname":
					_, _ = w.Write([]byte("h"))
				case "/meta-data/dns-conf/nameservers":
					_, _ = w.Write([]byte("   "))
				default:
					http.NotFound(w, r)
				}
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info) {
				s.Require().NotNil(info)
				s.Nil(info.Nameservers)
			},
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
			c := alibaba.NewWithClient(client)

			prior := tt.prior
			if prior == nil {
				prior = aliPrior()
			}
			out, err := c.Collect(context.Background(), prior)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			if tt.wantNoHTTP {
				s.False(httpCalled)
			}
			if tt.wantNil {
				s.Nil(out)
				return
			}
			info, ok := out.(*alibaba.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info)
			}
		})
	}
}
