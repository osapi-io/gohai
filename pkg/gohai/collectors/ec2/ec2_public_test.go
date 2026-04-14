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

package ec2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
)

func ec2Prior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{BIOS: &dmi.BIOS{Vendor: "Amazon EC2"}},
	}
}

var ec2Responses = map[string]string{
	"/latest/meta-data/ami-id":                      "ami-0123",
	"/latest/meta-data/ami-launch-index":            "0",
	"/latest/meta-data/ami-manifest-path":           "(unknown)",
	"/latest/meta-data/hostname":                    "ip-10-0-0-5.ec2.internal",
	"/latest/meta-data/instance-id":                 "i-abc",
	"/latest/meta-data/instance-type":               "t3.micro",
	"/latest/meta-data/instance-life-cycle":         "on-demand",
	"/latest/meta-data/local-hostname":              "ip-10-0-0-5.ec2.internal",
	"/latest/meta-data/local-ipv4":                  "10.0.0.5",
	"/latest/meta-data/mac":                         "0a:b0:c0:d0:e0:f0",
	"/latest/meta-data/placement/availability-zone": "us-east-1a",
	"/latest/meta-data/placement/region":            "us-east-1",
	"/latest/meta-data/public-hostname":             "ec2-1-2-3-4.compute-1.amazonaws.com",
	"/latest/meta-data/public-ipv4":                 "1.2.3.4",
	"/latest/meta-data/reservation-id":              "r-abc",
	"/latest/meta-data/security-groups":             "default\nssh",
	"/latest/meta-data/profile":                     "default-hvm",
}

const iamInfo = `{
  "Code": "Success",
  "LastUpdated": "2026-04-01T00:00:00Z",
  "InstanceProfileArn": "arn:aws:iam::123456789012:instance-profile/web",
  "InstanceProfileId": "AIPAXXXXXXXXXXXXXXXXX"
}`

const identityDoc = `{
  "accountId": "123456789012",
  "region": "us-east-1",
  "availabilityZone": "us-east-1a",
  "instanceId": "i-abc"
}`

type EC2PublicTestSuite struct {
	suite.Suite
}

func TestEC2PublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EC2PublicTestSuite))
}

// handler is the canned IMDS server: PUT on /latest/api/token
// returns a token; every GET requires X-aws-ec2-metadata-token.
func handler(
	tokenRequired bool,
	extraPaths map[string]string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/latest/api/token" {
			_, _ = w.Write([]byte("TOKEN-XYZ"))
			return
		}
		if tokenRequired && r.Header.Get("X-aws-ec2-metadata-token") == "" {
			http.Error(w, "token required", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/latest/meta-data/iam/info" {
			_, _ = w.Write([]byte(iamInfo))
			return
		}
		if r.URL.Path == "/latest/dynamic/instance-identity/document" {
			_, _ = w.Write([]byte(identityDoc))
			return
		}
		if body, ok := extraPaths[r.URL.Path]; ok {
			_, _ = w.Write([]byte(body))
			return
		}
		if body, ok := ec2Responses[r.URL.Path]; ok {
			_, _ = w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}
}

func (s *EC2PublicTestSuite) TestInterface() {
	c := ec2.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "ec2"},
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

func (s *EC2PublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		prior      collector.PriorResults
		handler    http.HandlerFunc
		closed     bool
		wantNil    bool
		wantNoHTTP bool
		verify     func(s *EC2PublicTestSuite, info *ec2.Info)
	}{
		{
			name:    "IMDSv2 happy path",
			handler: handler(true, nil),
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
				s.Equal("t3.micro", info.InstanceType)
				s.Equal("10.0.0.5", info.LocalIPv4)
				s.Equal("1.2.3.4", info.PublicIPv4)
				s.Equal("us-east-1", info.Region)
				s.Equal("us-east-1a", info.AvailabilityZone)
				s.Equal([]string{"default", "ssh"}, info.SecurityGroups)
				s.Equal("123456789012", info.AccountID)
				s.Require().NotNil(info.IAMInfo)
				s.Equal(
					"arn:aws:iam::123456789012:instance-profile/web",
					info.IAMInfo.InstanceProfileArn,
				)
			},
		},
		{
			name: "IMDSv1 fallback when token PUT 404s",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					http.NotFound(w, r)
					return
				}
				handler(false, nil)(w, r)
			},
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name: "dmi says not EC2 short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{BIOS: &dmi.BIOS{Vendor: "Dell Inc."}},
			},
			handler:    handler(true, nil),
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:    "no dmi in prior fails open",
			prior:   collector.PriorResults{},
			handler: handler(true, nil),
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name: "first-path 404 drops silently",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					_, _ = w.Write([]byte("T"))
					return
				}
				http.NotFound(w, r)
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
			name: "iam info missing tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/meta-data/iam/info" {
					http.NotFound(w, r)
					return
				}
				handler(true, nil)(w, r)
			},
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Nil(info.IAMInfo)
			},
		},
		{
			name: "malformed iam JSON tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/meta-data/iam/info" {
					_, _ = w.Write([]byte("not json"))
					return
				}
				handler(true, nil)(w, r)
			},
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Nil(info.IAMInfo)
			},
		},
		{
			name: "identity doc missing tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/dynamic/instance-identity/document" {
					http.NotFound(w, r)
					return
				}
				handler(true, nil)(w, r)
			},
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Equal("us-east-1", info.Region) // from meta-data/placement/region
			},
		},
		{
			name: "malformed identity doc JSON tolerated",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/dynamic/instance-identity/document" {
					_, _ = w.Write([]byte("not json"))
					return
				}
				handler(true, nil)(w, r)
			},
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Equal("us-east-1", info.Region)
				s.Empty(info.AccountID)
			},
		},
		{
			name: "identity doc fills missing fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if _, ok := ec2Responses[r.URL.Path]; ok &&
					r.URL.Path != "/latest/meta-data/ami-id" {
					http.NotFound(w, r)
					return
				}
				handler(true, nil)(w, r)
			},
			verify: func(s *EC2PublicTestSuite, info *ec2.Info) {
				s.Require().NotNil(info)
				s.Equal("i-abc", info.InstanceID)            // from identity doc
				s.Equal("us-east-1", info.Region)            // from identity doc
				s.Equal("us-east-1a", info.AvailabilityZone) // from identity doc
				s.Equal("123456789012", info.AccountID)
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
			c := ec2.NewWithClient(client)

			prior := tt.prior
			if prior == nil {
				prior = ec2Prior()
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
			info, ok := out.(*ec2.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info)
			}
		})
	}
}
