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
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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
    "0": {"id": "vol-a", "name": "root", "volume_type": "b_ssd", "size": 50000000000, "export_uri": "nbd://..."},
    "10": {"id": "vol-k", "name": "eleventh", "volume_type": "l_ssd", "size": 1, "export_uri": "nbd://..."},
    "2": {"id": "vol-c", "name": "third", "volume_type": "l_ssd", "size": 1, "export_uri": "nbd://..."},
    "1": {"id": "vol-b", "name": "data", "volume_type": "l_ssd", "size": 1, "export_uri": "nbd://..."}
  },
  "timezone": "Europe/Paris",
  "bootscript": {"id": "bs-1", "title": "x86_64 mainline 4.4.230 rev1", "architecture": "x86_64", "kernel": "https://s.scw.co/kernel", "initrd": "https://s.scw.co/initrd", "bootcmdargs": "LINUX_COMMON scaleway boot=local", "organization": "org-scw", "public": true}
}`

type ScalewayPublicTestSuite struct {
	suite.Suite
}

func TestScalewayPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ScalewayPublicTestSuite))
}

func (s *ScalewayPublicTestSuite) TestNew() {
	fs := memfs.New()
	tests := []struct {
		name         string
		c            scaleway.Collector
		validateFunc func(scaleway.Collector)
	}{
		{
			name: "New: its name, category, default and dependencies",
			c:    scaleway.New(),
			validateFunc: func(c scaleway.Collector) {
				s.Equal("scaleway", c.Name())
				s.Equal("cloud", string(c.Category()))
				s.False(c.DefaultEnabled())
				s.Equal([]string{"dmi"}, c.Dependencies())
			},
		},
		{
			name: "NewWith: reads the filesystem that it is given",
			c:    scaleway.NewWith(source.Sources{FS: fs, Family: family.Debian}),
			validateFunc: func(c scaleway.Collector) {
				s.Same(fs, c.(*scaleway.Metadata).FS)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.c)
		})
	}
}

func (s *ScalewayPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		cmdline      string
		noCmdline    bool // if true, point at a nonexistent file
		prior        collector.PriorResults
		handler      func(w http.ResponseWriter, r *http.Request)
		closed       bool
		validateFunc func(any, bool, error)
	}{
		{
			name:      "DMI sys_vendor Scaleway, with no cmdline signature, is detected",
			noCmdline: true,
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{VendorName: "Scaleway"}},
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			validateFunc: func(out any, httpCalled bool, err error) {
				s.Require().NoError(err)
				s.True(httpCalled)
				s.NotNil(out)
			},
		},
		{
			name:      "DMI of another vendor and no cmdline signature is not Scaleway",
			noCmdline: true,
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{VendorName: "QEMU"}},
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			validateFunc: func(out any, httpCalled bool, err error) {
				s.Require().NoError(err)
				s.False(httpCalled)
				s.Nil(out)
			},
		},
		{
			name:    "happy path transforms canned response",
			cmdline: "LABEL=cloudinit scaleway boot=local",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			validateFunc: func(out any, _ bool, _ error) {
				info, ok := out.(*scaleway.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("sc-abc", info.ID)
				s.Equal("prod-1", info.Name)
				s.Equal("DEV1-S", info.Type)
				s.Equal("51.0.0.1", info.PublicIP)
				s.Equal("pip-1", info.PublicIPID)
				s.False(info.PublicIPDynamic)
				s.Equal("10.64.0.1", info.PrivateIP)
				s.Equal("2001:db8::1", info.IPv6Address)
				s.Equal("/64", info.IPv6Netmask)
				s.Equal("fr-par-1", info.Zone)
				s.Equal("4", info.PlatformID)
				s.Equal([]string{"ssh-rsa AAA..."}, info.SSHPublicKeys)
				s.Require().Len(info.Volumes, 4)
				s.Equal("vol-a", info.Volumes[0].ID)
				s.Equal("vol-b", info.Volumes[1].ID)
				s.Equal("vol-c", info.Volumes[2].ID)
				s.Equal("vol-k", info.Volumes[3].ID)
				s.Equal("b_ssd", info.Volumes[0].VolumeType)
				s.Equal(int64(50000000000), info.Volumes[0].Size)
				s.Equal("Europe/Paris", info.Timezone)
				s.Require().NotNil(info.Bootscript)
				s.Equal("bs-1", info.Bootscript.ID)
				s.Equal("x86_64", info.Bootscript.Architecture)
				s.True(info.Bootscript.Public)
			},
		},
		{
			name:    "cmdline without scaleway signature short-circuits",
			cmdline: "LABEL=cloudinit boot=local",
			handler: func(http.ResponseWriter, *http.Request) {},
			validateFunc: func(out any, httpCalled bool, _ error) {
				s.False(httpCalled)
				s.Nil(out)
			},
		},
		{
			name:      "missing /proc/cmdline short-circuits",
			noCmdline: true,
			handler:   func(http.ResponseWriter, *http.Request) {},
			validateFunc: func(out any, httpCalled bool, _ error) {
				s.False(httpCalled)
				s.Nil(out)
			},
		},
		{
			name:    "404 drops silently when cmdline says scaleway",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.NotFound(w, nil)
			},
			validateFunc: func(out any, _ bool, err error) {
				s.NoError(err)
				s.Nil(out)
			},
		},
		{
			name:    "connection refused drops silently",
			cmdline: "scaleway",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			validateFunc: func(out any, _ bool, err error) {
				s.NoError(err)
				s.Nil(out)
			},
		},
		{
			name:    "a body that stops partway is an error",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte(`{"id":`))
			},
			validateFunc: func(out any, _ bool, err error) {
				s.ErrorContains(err, "scaleway metadata: read body")
				s.Nil(out)
			},
		},
		{
			name:    "malformed JSON surfaces as error",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("not json"))
			},
			validateFunc: func(_ any, _ bool, err error) {
				s.ErrorContains(err, "decode scaleway metadata")
			},
		},
		{
			name:    "empty optional nested objects produce empty fields",
			cmdline: "scaleway",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"id":"sc-x","tags":[]}`))
			},
			validateFunc: func(out any, _ bool, _ error) {
				info, ok := out.(*scaleway.Info)
				s.Require().True(ok)
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
			files := map[string]string{}
			if !tt.noCmdline {
				files["/proc/cmdline"] = tt.cmdline
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

			c := scaleway.NewWith(source.Sources{
				FS:   sourcetest.FS(s.T(), files),
				HTTP: sourcetest.Server(srv.URL),
			})

			out, err := c.Collect(context.Background(), tt.prior)
			tt.validateFunc(out, httpCalled, err)
		})
	}
}
