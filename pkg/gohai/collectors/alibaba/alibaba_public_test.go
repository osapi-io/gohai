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

// treeResponses mirrors Alibaba's metadata tree. Paths ending with "/"
// are directory listings (newline-separated children); others are
// leaves. Matches what Ohai's fetch_metadata would encounter.
var treeResponses = map[string]string{
	"/":                                     "meta-data/\nuser-data",
	"/meta-data/":                           "hostname\ninstance-id\nregion-id\nzone-id\nimage-id\ninstance/\nmac\nprivate-ipv4\neipv4\nvpc-id\nvpc-cidr-block\nvswitch-id\nvswitch-cidr-block\nserial-number\nnetwork-type\ndns-conf/\nntp-conf/\nram/",
	"/meta-data/hostname":                   "prod-1",
	"/meta-data/instance-id":                "i-abc",
	"/meta-data/region-id":                  "cn-hangzhou",
	"/meta-data/zone-id":                    "cn-hangzhou-b",
	"/meta-data/image-id":                   "img-1",
	"/meta-data/instance/":                  "instance-type\ninstance-name\nmax-netbw-ingress\nmax-netbw-egress",
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
	"/meta-data/dns-conf/":                  "nameservers",
	"/meta-data/dns-conf/nameservers":       "100.100.2.136 100.100.2.138",
	"/meta-data/ntp-conf/":                  "ntp-servers",
	"/meta-data/ntp-conf/ntp-servers":       "ntp1.aliyun.com ntp2.aliyun.com",
	"/meta-data/ram/":                       "role-name",
	"/meta-data/ram/role-name":              "ecs-default",
	// /user-data is served but should be excluded from the walk.
	"/user-data": "#!/bin/bash\\necho secret",
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

// serve wires the canned tree into an http.HandlerFunc. Paths absent
// from the map return 404 so tests exercise the tolerate-missing
// branches of walk.
func serve(
	tree map[string]string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := tree[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}
}

func (s *AlibabaPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		prior      collector.PriorResults
		tree       map[string]string
		handler    http.HandlerFunc // overrides tree when set
		closed     bool
		wantNil    bool
		wantNoHTTP bool
		verify     func(s *AlibabaPublicTestSuite, info *alibaba.Info, hitUserData bool)
	}{
		{
			name: "happy path populates typed fields + Raw tree",
			tree: treeResponses,
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, hitUserData bool) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
				s.Equal("prod-1", info.Hostname)
				s.Equal("cn-hangzhou", info.Region)
				s.Equal("cn-hangzhou-b", info.Zone)
				s.Equal("ecs.g6.large", info.InstanceType)
				s.Equal("prod-1", info.InstanceName)
				s.Equal("172.16.0.5", info.PrivateIPv4)
				s.Equal("47.1.2.3", info.PublicIPv4)
				s.Equal("vpc-1", info.VPCID)
				s.Equal("vsw-1", info.VSwitchID)
				s.Equal([]string{"100.100.2.136", "100.100.2.138"}, info.Nameservers)
				s.Equal([]string{"ntp1.aliyun.com", "ntp2.aliyun.com"}, info.NTPServers)
				s.Equal("ecs-default", info.RAMRoleName)
				s.Equal(int64(1048576), info.MaxBandwidthIngress)

				// Raw tree mirrors Ohai's node['alibaba']: sanitized
				// keys, nested sub-maps for directories.
				s.Require().NotNil(info.Raw)
				md, ok := info.Raw["meta_data"].(map[string]any)
				s.Require().True(ok)
				s.Equal("i-abc", md["instance_id"])
				// /user-data MUST NOT be present in Raw.
				_, hasUserData := info.Raw["user_data"]
				s.False(hasUserData)
				s.False(hitUserData)
			},
		},
		{
			name: "dmi says not Alibaba short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{Vendor: "Dell Inc."}},
			},
			tree:       treeResponses,
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:  "no dmi in prior fails open",
			prior: collector.PriorResults{},
			tree:  treeResponses,
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name:    "first-probe 404 drops silently",
			tree:    map[string]string{},
			wantNil: true,
		},
		{
			name:    "connection refused drops silently",
			closed:  true,
			wantNil: true,
		},
		{
			name: "JSON leaf parses as JSON",
			tree: map[string]string{
				"/":                    "meta-data/",
				"/meta-data/":          "json-leaf",
				"/meta-data/json-leaf": `{"foo": "bar"}`,
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				md, _ := info.Raw["meta_data"].(map[string]any)
				s.Require().NotNil(md)
				leaf, ok := md["json_leaf"].(map[string]any)
				s.Require().True(ok)
				s.Equal("bar", leaf["foo"])
			},
		},
		{
			name: "subdir fetch error is tolerated",
			tree: map[string]string{
				"/":                   "meta-data/",
				"/meta-data/":         "hostname\nbroken-subdir/",
				"/meta-data/hostname": "only-me",
				// /meta-data/broken-subdir/ intentionally absent → 404
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Equal("only-me", info.Hostname)
				md, _ := info.Raw["meta_data"].(map[string]any)
				_, present := md["broken_subdir"]
				s.False(present)
			},
		},
		{
			name: "leaf fetch error is tolerated",
			tree: map[string]string{
				"/":                   "meta-data/",
				"/meta-data/":         "hostname\nmissing",
				"/meta-data/hostname": "prod-1",
				// /meta-data/missing intentionally absent → 404
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Equal("prod-1", info.Hostname)
				md, _ := info.Raw["meta_data"].(map[string]any)
				_, present := md["missing"]
				s.False(present)
			},
		},
		{
			name: "empty dns-conf produces nil nameservers",
			tree: map[string]string{
				"/":                               "meta-data/",
				"/meta-data/":                     "hostname\ndns-conf/",
				"/meta-data/hostname":             "h",
				"/meta-data/dns-conf/":            "nameservers",
				"/meta-data/dns-conf/nameservers": "   ",
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Nil(info.Nameservers)
			},
		},
		{
			name: "empty ntp-servers produces nil NTPServers",
			tree: map[string]string{
				"/":                               "meta-data/",
				"/meta-data/":                     "hostname\nntp-conf/",
				"/meta-data/hostname":             "h",
				"/meta-data/ntp-conf/":            "ntp-servers",
				"/meta-data/ntp-conf/ntp-servers": "",
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Nil(info.NTPServers)
			},
		},
		{
			name: "tree without meta-data directory returns empty typed fields",
			tree: map[string]string{
				"/":                 "some-other-dir/",
				"/some-other-dir/":  "x",
				"/some-other-dir/x": "y",
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Require().NotNil(info)
				s.Empty(info.InstanceID)
				s.Empty(info.Hostname)
				// Raw still has the alternate tree — forward-compat
				// guarantee.
				s.Contains(info.Raw, "some_other_dir")
			},
		},
		{
			name: "wrong-typed raw tree for canonical sub-objects is tolerated",
			tree: map[string]string{
				"/":                   "meta-data/",
				"/meta-data/":         "hostname\ninstance\ndns-conf\nntp-conf\nram",
				"/meta-data/hostname": "h",
				"/meta-data/instance": `"not-a-map"`,
				"/meta-data/dns-conf": `"neither"`,
				"/meta-data/ntp-conf": `"nor"`,
				"/meta-data/ram":      `"nope"`,
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Equal("h", info.Hostname)
				s.Empty(info.InstanceType)
				s.Nil(info.Nameservers)
				s.Nil(info.NTPServers)
				s.Empty(info.RAMRoleName)
			},
		},
		{
			name: "wrong-typed leaf value in map is tolerated by strVal",
			tree: map[string]string{
				"/":                      "meta-data/",
				"/meta-data/":            "instance-id",
				"/meta-data/instance-id": `123`, // parses as JSON number, not string
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Empty(info.InstanceID) // strVal returns "" when type isn't string
			},
		},
		{
			name: "empty listing lines are skipped",
			tree: map[string]string{
				"/":                   "\n\nmeta-data/\n",
				"/meta-data/":         "\nhostname\n\n",
				"/meta-data/hostname": "h",
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Equal("h", info.Hostname)
			},
		},
		{
			name: "instance sub-map without bandwidth fields hits intVal miss branch",
			tree: map[string]string{
				"/":                                 "meta-data/",
				"/meta-data/":                       "instance/",
				"/meta-data/instance/":              "instance-type",
				"/meta-data/instance/instance-type": "ecs.g6.large",
			},
			verify: func(s *AlibabaPublicTestSuite, info *alibaba.Info, _ bool) {
				s.Equal("ecs.g6.large", info.InstanceType)
				s.Equal(int64(0), info.MaxBandwidthIngress)
				s.Equal(int64(0), info.MaxBandwidthEgress)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var httpCalled bool
			var hitUserData bool
			h := tt.handler
			if h == nil {
				h = serve(tt.tree)
			}
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled = true
					if r.URL.Path == "/user-data" {
						hitUserData = true
					}
					h(w, r)
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
				tt.verify(s, info, hitUserData)
			}
		})
	}
}
