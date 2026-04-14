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
	"os"
	"path/filepath"
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

// metadataTree mirrors OpenStack's EC2-compatible meta-data tree.
// Paths ending with "/" are directory listings; others are leaves.
var metadataTree = map[string]string{
	"/latest/meta-data/":                            "ami-id\ninstance-id\ninstance-type\nhostname\nlocal-hostname\nlocal-ipv4\npublic-hostname\npublic-ipv4\nplacement/\nreservation-id\nkernel-id\nramdisk-id\nsecurity-groups",
	"/latest/meta-data/ami-id":                      "ami-xxx",
	"/latest/meta-data/instance-id":                 "i-abc",
	"/latest/meta-data/instance-type":               "m1.small",
	"/latest/meta-data/hostname":                    "prod-1",
	"/latest/meta-data/local-hostname":              "prod-1.local",
	"/latest/meta-data/local-ipv4":                  "10.0.0.5",
	"/latest/meta-data/public-hostname":             "prod-1.public",
	"/latest/meta-data/public-ipv4":                 "10.0.0.6",
	"/latest/meta-data/placement/":                  "availability-zone",
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
	tmpDir string
}

func TestOpenStackPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OpenStackPublicTestSuite))
}

func (s *OpenStackPublicTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
	// Default: point passwd path at a nonexistent file so provider
	// resolves to "openstack" unless a test overrides.
	openstack.SetPasswdPath(filepath.Join(s.tmpDir, "no-passwd"))
}

// writePasswd writes a passwd-format file with the given lines and
// swaps the package var.
func (s *OpenStackPublicTestSuite) writePasswd(
	content string,
) func() {
	path := filepath.Join(s.tmpDir, "passwd")
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
	return openstack.SetPasswdPath(path)
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

func canned(
	tree map[string]string,
	novaBody string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openstack/latest/meta_data.json" {
			if novaBody == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(novaBody))
			return
		}
		body, ok := tree[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}
}

func (s *OpenStackPublicTestSuite) TestCollect() {
	tests := []struct {
		name        string
		prior       collector.PriorResults
		tree        map[string]string
		novaBody    string
		passwdLines string // when non-empty, written before Collect
		handler     http.HandlerFunc
		closed      bool
		wantNil     bool
		wantNoHTTP  bool
		verify      func(s *OpenStackPublicTestSuite, info *openstack.Info)
	}{
		{
			name:     "happy path combines EC2-mirror walk + Nova doc + openstack provider",
			tree:     metadataTree,
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Equal("openstack", info.Provider)
				s.Equal("i-abc", info.InstanceID)
				s.Equal("m1.small", info.InstanceType)
				s.Equal("10.0.0.5", info.LocalIPv4)
				s.Equal("nova", info.AvailabilityZone)
				s.Equal([]string{"default", "ssh"}, info.SecurityGroups)
				s.Equal("uuid-xxx", info.UUID)
				s.Equal("proj-1", info.ProjectID)
				s.Equal("sre", info.MetaData["owner"])
				// Raw forward-compat
				s.Require().NotNil(info.Raw)
				s.Equal("i-abc", info.Raw["instance_id"])
			},
		},
		{
			name:        "passwd exists without dhc-user → provider = openstack",
			tree:        metadataTree,
			novaBody:    novaDoc,
			passwdLines: "root:x:0:0:root:/root:/bin/bash\nuser:x:1000:1000::/home/user:/bin/bash\n",
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("openstack", info.Provider)
			},
		},
		{
			name:        "passwd contains dhc-user → provider = dreamhost",
			tree:        metadataTree,
			novaBody:    novaDoc,
			passwdLines: "root:x:0:0:root:/root:/bin/bash\ndhc-user:x:1000:1000::/home/dhc-user:/bin/bash\n",
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("dreamhost", info.Provider)
			},
		},
		{
			name: "dmi says not OpenStack short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{Name: "VMware"}},
			},
			tree:       metadataTree,
			novaBody:   novaDoc,
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:     "no dmi in prior fails open",
			prior:    collector.PriorResults{},
			tree:     metadataTree,
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name:     "EC2-mirror missing but Nova doc present → still detected",
			tree:     map[string]string{},
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Require().NotNil(info)
				s.Equal("uuid-xxx", info.UUID)
				s.Empty(info.InstanceID) // no EC2-mirror data
			},
		},
		{
			name:     "both EC2-mirror and Nova doc missing → drop",
			tree:     map[string]string{},
			novaBody: "",
			wantNil:  true,
		},
		{
			name: "Nova doc malformed JSON tolerated",
			tree: metadataTree,
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/openstack/latest/meta_data.json" {
					_, _ = w.Write([]byte("not json"))
					return
				}
				canned(metadataTree, "")(w, r)
			},
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Empty(info.UUID)
				s.Equal("i-abc", info.InstanceID)
			},
		},
		{
			name: "EC2-mirror partial: only ami-id present",
			tree: map[string]string{
				"/latest/meta-data/":       "ami-id",
				"/latest/meta-data/ami-id": "ami-yyy",
			},
			novaBody: `{"uuid": "uuid-solo", "hostname": "h"}`,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("uuid-solo", info.UUID)
				s.Equal("h", info.Hostname)
				s.Equal("ami-yyy", info.AMIID)
			},
		},
		{
			name: "broken subdir during walk is tolerated",
			tree: map[string]string{
				"/latest/meta-data/":       "ami-id\nbroken/",
				"/latest/meta-data/ami-id": "ami-zzz",
				// /latest/meta-data/broken/ intentionally missing → 404
			},
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("ami-zzz", info.AMIID)
				_, present := info.Raw["broken"]
				s.False(present)
			},
		},
		{
			name: "leaf fetch error during walk is tolerated",
			tree: map[string]string{
				"/latest/meta-data/":       "ami-id\nmissing-leaf",
				"/latest/meta-data/ami-id": "ami-aaa",
				// /latest/meta-data/missing-leaf intentionally absent → 404
			},
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("ami-aaa", info.AMIID)
				_, present := info.Raw["missing_leaf"]
				s.False(present)
			},
		},
		{
			name:    "connection refused drops silently",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			wantNil: true,
		},
		{
			name: "empty listing lines are skipped",
			tree: map[string]string{
				"/latest/meta-data/":       "\n\nami-id\n\n",
				"/latest/meta-data/ami-id": "ami-skip",
			},
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("ami-skip", info.AMIID)
			},
		},
		{
			name: "tree without placement subdirectory leaves AZ empty (until Nova fills it)",
			tree: map[string]string{
				"/latest/meta-data/":       "ami-id",
				"/latest/meta-data/ami-id": "ami-no-placement",
			},
			novaBody: novaDoc,
			verify: func(s *OpenStackPublicTestSuite, info *openstack.Info) {
				s.Equal("nova", info.AvailabilityZone) // from Nova doc
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.passwdLines != "" {
				defer s.writePasswd(tt.passwdLines)()
			}

			var httpCalled bool
			h := tt.handler
			if h == nil {
				h = canned(tt.tree, tt.novaBody)
			}
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled = true
					h(w, r)
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
