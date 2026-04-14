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

package gce_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cloud/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware/dmi"
)

// gcePrior builds a PriorResults containing a dmi.Info whose product
// name matches GCE — the gate the collector's Collect runs before
// hitting the metadata endpoint.
func gcePrior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{Product: &dmi.Product{Name: "Google Compute Engine"}},
	}
}

// cannedResponse is a realistic slice of GCE's recursive metadata
// response — same field names and shape as the real service, including
// the fields (attributes, licenses, scheduling detail, IAM scopes,
// advanced network fields, disk encryption) that earlier versions of
// this collector dropped.
const cannedResponse = `{
  "instance": {
    "id": 1234567890123,
    "name": "my-vm",
    "hostname": "my-vm.c.my-project.internal",
    "description": "primary app server",
    "zone": "projects/987654321/zones/us-central1-a",
    "machineType": "projects/987654321/machineTypes/n1-standard-1",
    "cpuPlatform": "Intel Broadwell",
    "image": "projects/debian-cloud/global/images/debian-12",
    "tags": ["http-server", "https-server"],
    "disks": [
      {
        "deviceName": "boot",
        "index": 0,
        "mode": "READ_WRITE",
        "type": "PERSISTENT",
        "interface": "SCSI",
        "encrypted": true
      }
    ],
    "networkInterfaces": [
      {
        "ip": "10.128.0.5",
        "mac": "42:01:0a:80:00:05",
        "network": "projects/987654321/networks/default",
        "subnetmask": "255.255.240.0",
        "gateway": "10.128.0.1",
        "dnsServers": ["169.254.169.254"],
        "ipAliases": ["10.132.0.0/20"],
        "forwardedIps": ["34.102.0.1"],
        "targetInstanceIps": [],
        "mtu": 1460,
        "accessConfigs": [
          {"externalIp": "34.123.45.67", "type": "ONE_TO_ONE_NAT"},
          {"externalIp": "34.123.45.68", "type": "ONE_TO_ONE_NAT"}
        ]
      }
    ],
    "serviceAccounts": {
      "default": {
        "email": "default@my-project.iam.gserviceaccount.com",
        "aliases": ["alt"],
        "scopes": [
          "https://www.googleapis.com/auth/cloud-platform",
          "https://www.googleapis.com/auth/logging.write"
        ]
      }
    },
    "scheduling": {
      "preemptible": "FALSE",
      "automaticRestart": "TRUE",
      "onHostMaintenance": "MIGRATE"
    },
    "attributes": {
      "ssh-keys": "user:ssh-rsa AAAA...",
      "startup-script": "#!/bin/bash\\necho hi"
    },
    "licenses": [{"id": "8045211539491955793"}],
    "maintenanceEvent": "NONE"
  },
  "project": {
    "projectId": "my-project",
    "numericProjectId": 987654321,
    "attributes": {
      "ssh-keys": "admin:ssh-rsa BBBB..."
    }
  }
}`

type GcePublicTestSuite struct {
	suite.Suite
}

func TestGcePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GcePublicTestSuite))
}

func (s *GcePublicTestSuite) TestNew() {
	s.NotNil(gce.New())
}

func (s *GcePublicTestSuite) TestMetadata() {
	tests := []struct {
		name       string
		handler    func(w http.ResponseWriter, r *http.Request)
		closed     bool                   // if true, close server before calling Collect
		prior      collector.PriorResults // defaults to gcePrior() when nil
		wantNil    bool
		wantErr    bool
		wantNoHTTP bool // if true, the gate should have short-circuited before the HTTP call
		verify     func(s *GcePublicTestSuite, info *gce.Info, hdrGot string)
	}{
		{
			name: "happy path transforms raw response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			verify: func(s *GcePublicTestSuite, info *gce.Info, hdrGot string) {
				s.Equal("Google", hdrGot)
				s.Require().NotNil(info)
				s.Equal(int64(1234567890123), info.InstanceID)
				s.Equal("my-vm", info.Name)
				s.Equal("my-vm.c.my-project.internal", info.Hostname)
				s.Equal("primary app server", info.Description)
				s.Equal("us-central1-a", info.Zone)
				s.Equal("us-central1", info.Region)
				s.Equal("n1-standard-1", info.MachineType)
				s.Equal("debian-12", info.Image)
				s.Equal("Intel Broadwell", info.CPUPlatform)
				s.Equal([]string{"http-server", "https-server"}, info.Tags)
				s.False(info.Preemptible)
				s.Equal("TRUE", info.AutomaticRestart)
				s.Equal("MIGRATE", info.OnHostMaintenance)
				s.Equal("NONE", info.MaintenanceEvent)
				s.Equal("my-project", info.ProjectID)
				s.Equal(int64(987654321), info.NumericProjectID)

				s.Equal([]string{"8045211539491955793"}, info.Licenses)
				s.Contains(info.Attributes, "ssh-keys")
				s.Contains(info.Attributes, "startup-script")
				s.Contains(info.ProjectAttributes, "ssh-keys")

				s.Require().Len(info.Disks, 1)
				d := info.Disks[0]
				s.Equal("boot", d.DeviceName)
				s.Equal("PERSISTENT", d.Type)
				s.Equal("SCSI", d.Interface)
				s.True(d.Encrypted)

				s.Require().Len(info.NetworkInterfaces, 1)
				iface := info.NetworkInterfaces[0]
				s.Equal("10.128.0.5", iface.IP)
				s.Equal("default", iface.Network)
				s.Equal([]string{"10.132.0.0/20"}, iface.IPAliases)
				s.Equal([]string{"34.102.0.1"}, iface.ForwardedIPs)
				s.Equal(1460, iface.MTU)
				s.Require().Len(iface.AccessConfigs, 2)
				s.Equal("34.123.45.67", iface.AccessConfigs[0].ExternalIP)
				s.Equal("34.123.45.68", iface.AccessConfigs[1].ExternalIP)

				s.Require().Len(info.ServiceAccounts, 1)
				sa := info.ServiceAccounts[0]
				s.Equal("default", sa.Key)
				s.Equal("default@my-project.iam.gserviceaccount.com", sa.Email)
				s.Equal([]string{"alt"}, sa.Aliases)
				s.Len(sa.Scopes, 2)
			},
		},
		{
			name: "preemptible TRUE parses as bool",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(
					[]byte(`{"instance":{"scheduling":{"preemptible":"TRUE"}},"project":{}}`),
				)
			},
			verify: func(s *GcePublicTestSuite, info *gce.Info, _ string) {
				s.Require().NotNil(info)
				s.True(info.Preemptible)
			},
		},
		{
			name: "multiple service accounts collected",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"instance":{"serviceAccounts":{
					"default":{"email":"a@x.iam.gserviceaccount.com"},
					"extra":{"email":"b@x.iam.gserviceaccount.com"}
				}},"project":{}}`))
			},
			verify: func(s *GcePublicTestSuite, info *gce.Info, _ string) {
				s.Require().NotNil(info)
				emails := make([]string, 0, len(info.ServiceAccounts))
				for _, sa := range info.ServiceAccounts {
					emails = append(emails, sa.Email)
				}
				sort.Strings(emails)
				s.Equal([]string{
					"a@x.iam.gserviceaccount.com",
					"b@x.iam.gserviceaccount.com",
				}, emails)
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
			name: "500 drops silently",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
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
		{
			name:    "dmi says not GCE short-circuits without HTTP call",
			handler: func(http.ResponseWriter, *http.Request) {},
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{Name: "OptiPlex 3070"}},
			},
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:    "empty dmi product fails open and tries HTTP",
			handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(cannedResponse)) },
			prior:   collector.PriorResults{"dmi": &dmi.Info{}},
			verify: func(s *GcePublicTestSuite, info *gce.Info, _ string) {
				s.Require().NotNil(info)
				s.Equal("my-vm", info.Name)
			},
		},
		{
			name:    "no dmi in prior fails open and tries HTTP",
			handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(cannedResponse)) },
			prior:   collector.PriorResults{},
			verify: func(s *GcePublicTestSuite, info *gce.Info, _ string) {
				s.Require().NotNil(info)
				s.Equal("my-vm", info.Name)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var hdrGot string
			var httpCalled bool
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled = true
					hdrGot = r.Header.Get("Metadata-Flavor")
					tt.handler(w, r)
				},
			))
			if tt.closed {
				srv.Close()
			} else {
				defer srv.Close()
			}

			client := cloudmetadata.New(srv.URL,
				cloudmetadata.WithHeader("Metadata-Flavor", "Google"))
			c := gce.NewWithClient(client)

			prior := tt.prior
			if prior == nil {
				prior = gcePrior()
			}
			out, err := c.Collect(context.Background(), prior)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			if tt.wantNoHTTP {
				s.False(httpCalled, "collector should have skipped the HTTP call")
			}

			if tt.wantNil {
				s.Nil(out)
				return
			}

			info, ok := out.(*gce.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info, hdrGot)
			}
		})
	}
}

func (s *GcePublicTestSuite) TestMetadataInterface() {
	c := gce.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "gce"},
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
