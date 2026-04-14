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

package scaleway_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
)

const cannedResponse = `{
  "id": "sc-abc",
  "name": "prod-1",
  "hostname": "prod-1",
  "organization": "org-uuid",
  "project": "proj-uuid",
  "commercial_type": "DEV1-S",
  "tags": ["web", "prod"],
  "state_detail": "booted",
  "public_ip": {"id": "pip-1", "address": "51.0.0.1", "dynamic": false},
  "private_ip": "10.64.0.1",
  "ipv6": {"address": "2001:db8::1", "netmask": "/64", "gateway": "2001:db8::"},
  "location": {"zone_id": "fr-par-1", "platform_id": "4"},
  "ssh_public_keys": [{"key": "ssh-rsa AAA..."}, {"key": "  "}],
  "volumes": {
    "0": {"id": "vol-a", "name": "root", "volume_type": "b_ssd", "size": 50000000000, "export_uri": "nbd://..."}
  }
}`

type ScalewayPublicTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestScalewayPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ScalewayPublicTestSuite))
}

func (s *ScalewayPublicTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// writeCmdline writes a /proc/cmdline replacement containing the
// given content and swaps the package var. Returns restore func.
func (s *ScalewayPublicTestSuite) writeCmdline(
	content string,
) func() {
	path := filepath.Join(s.tmpDir, "cmdline")
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
	return scaleway.SetProcCmdlinePath(path)
}

func (s *ScalewayPublicTestSuite) TestInterface() {
	c := scaleway.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "scaleway"},
		{"Category", c.Category(), "cloud"},
		{"DefaultEnabled", c.DefaultEnabled(), false},
		{"Dependencies", c.Dependencies(), []string(nil)},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.got)
		})
	}
}

func (s *ScalewayPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		cmdline    string
		noCmdline  bool // if true, point at a nonexistent file
		handler    func(w http.ResponseWriter, r *http.Request)
		closed     bool
		wantNil    bool
		wantErr    bool
		wantNoHTTP bool
		verify     func(s *ScalewayPublicTestSuite, info *scaleway.Info)
	}{
		{
			name:    "happy path transforms canned response",
			cmdline: "LABEL=cloudinit scaleway boot=local",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			verify: func(s *ScalewayPublicTestSuite, info *scaleway.Info) {
				s.Require().NotNil(info)
				s.Equal("sc-abc", info.ID)
				s.Equal("prod-1", info.Name)
				s.Equal("DEV1-S", info.CommercialType)
				s.Equal("51.0.0.1", info.PublicIP)
				s.Equal("pip-1", info.PublicIPID)
				s.False(info.PublicIPDynamic)
				s.Equal("10.64.0.1", info.PrivateIP)
				s.Equal("2001:db8::1", info.IPv6Address)
				s.Equal("/64", info.IPv6Netmask)
				s.Equal("fr-par-1", info.Zone)
				s.Equal("4", info.PlatformID)
				s.Equal([]string{"ssh-rsa AAA..."}, info.SSHPublicKeys)
				s.Require().Len(info.Volumes, 1)
				s.Equal("vol-a", info.Volumes[0].ID)
				s.Equal("b_ssd", info.Volumes[0].VolumeType)
				s.Equal(int64(50000000000), info.Volumes[0].Size)
			},
		},
		{
			name:       "cmdline without scaleway signature short-circuits",
			cmdline:    "LABEL=cloudinit boot=local",
			handler:    func(http.ResponseWriter, *http.Request) {},
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:       "missing /proc/cmdline short-circuits",
			noCmdline:  true,
			handler:    func(http.ResponseWriter, *http.Request) {},
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:    "404 drops silently when cmdline says scaleway",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.NotFound(w, nil)
			},
			wantNil: true,
		},
		{
			name:    "connection refused drops silently",
			cmdline: "scaleway",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			wantNil: true,
		},
		{
			name:    "malformed JSON surfaces as error",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("not json"))
			},
			wantErr: true,
		},
		{
			name:    "empty optional nested objects produce empty fields",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"id":"sc-x","tags":[]}`))
			},
			verify: func(s *ScalewayPublicTestSuite, info *scaleway.Info) {
				s.Require().NotNil(info)
				s.Equal("sc-x", info.ID)
				s.Empty(info.PublicIP)
				s.Empty(info.IPv6Address)
				s.Empty(info.Zone)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.noCmdline {
				defer scaleway.SetProcCmdlinePath(filepath.Join(s.tmpDir, "nonexistent"))()
			} else {
				defer s.writeCmdline(tt.cmdline)()
			}

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
			c := scaleway.NewWithClient(client)

			out, err := c.Collect(context.Background(), nil)
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
			info, ok := out.(*scaleway.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info)
			}
		})
	}
}
