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

package digital_ocean_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

// doPrior returns a prior-results map whose dmi bios vendor matches
// DigitalOcean, so Collect will proceed past the gate.
func doPrior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{BIOS: &dmi.BIOS{Vendor: "DigitalOcean"}},
	}
}

const cannedResponse = `{
  "droplet_id": 123456,
  "hostname": "web-1",
  "region": "nyc3",
  "public_keys": ["ssh-rsa AAAA..."],
  "tags": ["prod"],
  "features": ["dhcp_enabled"],
  "vendor_data": "#cloud-config\nsecret: dropped",
  "floating_ip": {"ipv4": {"ip_address": "138.1.2.3"}},
  "dns": {"nameservers": ["67.207.67.2", "67.207.67.3"]},
  "interfaces": {
    "public": [{
      "mac": "f6:a0:11:22:33:44",
      "type": "public",
      "ipv4": {"ip_address": "138.4.5.6", "netmask": "255.255.240.0", "gateway": "138.4.0.1"},
      "ipv6": {"ip_address": "2001:db8::1", "cidr": 64, "gateway": "fe80::1"},
      "anchor_ipv4": {"ip_address": "10.15.0.2"}
    }],
    "private": [{
      "mac": "f6:a0:aa:bb:cc:dd",
      "type": "private",
      "ipv4": {"ip_address": "10.132.0.2", "netmask": "255.255.0.0", "gateway": "10.132.0.1"}
    }]
  }
}`

type DigitalOceanPublicTestSuite struct {
	suite.Suite
}

func TestDigitalOceanPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DigitalOceanPublicTestSuite))
}

func (s *DigitalOceanPublicTestSuite) TestInterface() {
	c := digitalocean.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "digital_ocean"},
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

func (s *DigitalOceanPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		prior      collector.PriorResults
		handler    func(w http.ResponseWriter, r *http.Request)
		closed     bool
		wantNil    bool
		wantErr    bool
		wantNoHTTP bool
		verify     func(s *DigitalOceanPublicTestSuite, info *digitalocean.Info)
	}{
		{
			name: "happy path transforms canned response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			verify: func(s *DigitalOceanPublicTestSuite, info *digitalocean.Info) {
				s.Require().NotNil(info)
				s.Equal(int64(123456), info.DropletID)
				s.Equal("web-1", info.Hostname)
				s.Equal("nyc3", info.Region)
				s.Equal([]string{"prod"}, info.Tags)
				s.Equal([]string{"dhcp_enabled"}, info.Features)
				s.Equal([]string{"ssh-rsa AAAA..."}, info.PublicKeys)
				s.Equal("138.1.2.3", info.FloatingIP)
				s.Equal([]string{"67.207.67.2", "67.207.67.3"}, info.IPv4NS)

				s.Require().Len(info.Interfaces, 2)
				pub := info.Interfaces[0]
				s.Equal("public", pub.Scope)
				s.Equal("138.4.5.6", pub.IPv4)
				s.Equal("255.255.240.0", pub.IPv4Mask)
				s.Equal("2001:db8::1", pub.IPv6)
				s.Equal(64, pub.IPv6Mask)
				s.Equal("10.15.0.2", pub.Anchor)

				priv := info.Interfaces[1]
				s.Equal("private", priv.Scope)
				s.Equal("10.132.0.2", priv.IPv4)
				s.Empty(priv.IPv6)
			},
		},
		{
			name: "dmi says not DO short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{BIOS: &dmi.BIOS{Vendor: "Dell Inc."}},
			},
			handler:    func(http.ResponseWriter, *http.Request) {},
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:  "dmi without BIOS fails open and tries HTTP",
			prior: collector.PriorResults{"dmi": &dmi.Info{}},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"droplet_id": 42}`))
			},
			verify: func(s *DigitalOceanPublicTestSuite, info *digitalocean.Info) {
				s.Require().NotNil(info)
				s.Equal(int64(42), info.DropletID)
			},
		},
		{
			name:  "no dmi in prior fails open",
			prior: collector.PriorResults{},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"droplet_id": 43}`))
			},
			verify: func(s *DigitalOceanPublicTestSuite, info *digitalocean.Info) {
				s.Require().NotNil(info)
				s.Equal(int64(43), info.DropletID)
			},
		},
		{
			name: "404 drops silently",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.NotFound(w, nil)
			},
			wantNil: true,
		},
		{
			name:    "connection refused drops silently",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			wantNil: true,
		},
		{
			name: "malformed JSON surfaces as error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("not json"))
			},
			wantErr: true,
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
			c := digitalocean.NewWithClient(client)

			prior := tt.prior
			if prior == nil {
				prior = doPrior()
			}
			out, err := c.Collect(context.Background(), prior)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			if tt.wantNoHTTP {
				s.False(httpCalled, "should have short-circuited before HTTP")
			}
			if tt.wantNil {
				s.Nil(out)
				return
			}
			info, ok := out.(*digitalocean.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info)
			}
		})
	}
}
