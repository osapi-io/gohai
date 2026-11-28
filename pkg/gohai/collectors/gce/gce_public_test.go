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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// gcePrior builds a PriorResults containing a dmi.Info whose product
// name matches GCE. That is the gate the collector's Collect runs before
// hitting the metadata endpoint.
func gcePrior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{Product: &dmi.Product{Name: "Google Compute Engine"}},
	}
}

// cannedResponse is a realistic slice of GCE's recursive metadata
// response, with the same field names and shape as the real service, including
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
    "maintenanceEvent": "NONE",
    "virtualClock": {"driftToken": "abc-123"},
    "remainingCpuTime": 3600,
    "partnerAttributes": {"marketplace_product": "some-appliance"}
  },
  "project": {
    "projectId": "my-project",
    "numericProjectId": 987654321,
    "attributes": {
      "ssh-keys": "admin:ssh-rsa BBBB..."
    }
  }
}`

// metadata answers each request with status and body, or with err, and
// keeps the requests that it saw. http.RoundTripper is the standard
// library's interface, so it has no generated mock. The http.Client
// calls RoundTrip on the caller's goroutine, so the test reads requests
// with no lock.
type metadata struct {
	status   int
	body     string
	err      error
	requests []*http.Request
}

// RoundTrip records req and gives the canned answer.
func (m *metadata) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return nil, m.err
	}

	return &http.Response{
		StatusCode: m.status,
		Status:     http.StatusText(m.status),
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Request:    req,
	}, nil
}

type GcePublicTestSuite struct {
	suite.Suite
}

func TestGcePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GcePublicTestSuite))
}

func (s *GcePublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		validateFunc func(gce.Collector)
	}{
		{
			name: "New gives the gce cloud collector, opt-in, after dmi",
			validateFunc: func(c gce.Collector) {
				s.Equal("gce", c.Name())
				s.Equal(collector.CategoryCloud, c.Category())
				s.False(c.DefaultEnabled())
				s.Equal([]string{"dmi"}, c.Dependencies())
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(gce.New())
		})
	}
}

func (s *GcePublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		meta         *metadata
		prior        collector.PriorResults // defaults to gcePrior() when nil
		validateFunc func(any, error, *metadata)
	}{
		{
			name: "happy path transforms raw response",
			meta: &metadata{status: http.StatusOK, body: cannedResponse},
			validateFunc: func(out any, err error, m *metadata) {
				s.Require().NoError(err)
				s.Require().Len(m.requests, 1)
				s.Equal("Google", m.requests[0].Header.Get("Metadata-Flavor"))
				s.Equal(
					"http://metadata.google.internal./computeMetadata/v1/?recursive=true",
					m.requests[0].URL.String(),
				)
				info, ok := out.(*gce.Info)
				s.Require().True(ok)
				s.Equal(int64(1234567890123), info.ID)
				s.Equal("my-vm", info.Name)
				s.Equal("my-vm.c.my-project.internal", info.Hostname)
				s.Equal("primary app server", info.Description)
				s.Equal("us-central1-a", info.Zone)
				s.Equal("us-central1", info.Region)
				s.Equal("n1-standard-1", info.Type)
				s.Equal("debian-12", info.ImageID)
				s.Equal("Intel Broadwell", info.CPUPlatform)
				s.Equal([]string{"http-server", "https-server"}, info.Tags)
				s.False(info.Preemptible)
				s.Equal("TRUE", info.AutomaticRestart)
				s.Equal("MIGRATE", info.OnHostMaintenance)
				s.Equal("NONE", info.MaintenanceEvent)
				s.Equal("abc-123", info.VirtualClockDriftToken)
				s.Equal(int64(3600), info.RemainingCPUTime)
				s.Equal("my-project", info.ProjectUID)
				s.Equal(int64(987654321), info.NumericProjectID)

				s.Equal([]string{"8045211539491955793"}, info.Licenses)
				// The metadata and partner attributes hold ssh-keys,
				// startup scripts and partner data. The facts leave
				// them out.
				facts, err := json.Marshal(info)
				s.Require().NoError(err)
				s.NotContains(string(facts), "ssh-keys")
				s.NotContains(string(facts), "startup-script")
				s.NotContains(string(facts), "some-appliance")

				s.Equal([]gce.Disk{{
					DeviceName: "boot",
					Type:       "PERSISTENT",
					Mode:       "READ_WRITE",
					Interface:  "SCSI",
					Encrypted:  true,
				}}, info.Disks)

				s.Require().Len(info.NetworkInterfaces, 1)
				iface := info.NetworkInterfaces[0]
				s.Equal("10.128.0.5", iface.IP)
				s.Equal("default", iface.Network)
				s.Equal([]string{"10.132.0.0/20"}, iface.IPAliases)
				s.Equal([]string{"34.102.0.1"}, iface.ForwardedIPs)
				s.Equal(1460, iface.MTU)
				s.Equal([]gce.AccessConfig{
					{ExternalIP: "34.123.45.67", Type: "ONE_TO_ONE_NAT"},
					{ExternalIP: "34.123.45.68", Type: "ONE_TO_ONE_NAT"},
				}, iface.AccessConfigs)

				s.Equal([]gce.ServiceAccount{{
					Key:     "default",
					Email:   "default@my-project.iam.gserviceaccount.com",
					Aliases: []string{"alt"},
					Scopes: []string{
						"https://www.googleapis.com/auth/cloud-platform",
						"https://www.googleapis.com/auth/logging.write",
					},
				}}, info.ServiceAccounts)
			},
		},
		{
			name: "preemptible TRUE parses as bool",
			meta: &metadata{
				status: http.StatusOK,
				body:   `{"instance":{"scheduling":{"preemptible":"TRUE"}},"project":{}}`,
			},
			validateFunc: func(out any, err error, _ *metadata) {
				s.Require().NoError(err)
				info, ok := out.(*gce.Info)
				s.Require().True(ok)
				s.True(info.Preemptible)
			},
		},
		{
			name: "service accounts are in key order",
			meta: &metadata{
				status: http.StatusOK,
				body: `{"instance":{"serviceAccounts":{
					"zeta":{"email":"z@x.iam.gserviceaccount.com"},
					"alpha":{"email":"a@x.iam.gserviceaccount.com"},
					"mid":{"email":"m@x.iam.gserviceaccount.com"}
				}},"project":{}}`,
			},
			validateFunc: func(out any, err error, _ *metadata) {
				s.Require().NoError(err)
				info, ok := out.(*gce.Info)
				s.Require().True(ok)
				s.Equal([]gce.ServiceAccount{
					{Key: "alpha", Email: "a@x.iam.gserviceaccount.com"},
					{Key: "mid", Email: "m@x.iam.gserviceaccount.com"},
					{Key: "zeta", Email: "z@x.iam.gserviceaccount.com"},
				}, info.ServiceAccounts)
			},
		},
		{
			name: "404 is not GCE",
			meta: &metadata{status: http.StatusNotFound},
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Len(m.requests, 1)
			},
		},
		{
			name: "500 is not GCE",
			meta: &metadata{status: http.StatusInternalServerError, body: "boom"},
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Len(m.requests, 1)
			},
		},
		{
			name: "connection refused is not GCE",
			meta: &metadata{err: errors.New("connection refused")},
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Len(m.requests, 1)
			},
		},
		{
			name: "a body over the limit is an error, not an absent fact",
			meta: &metadata{
				status: http.StatusOK,
				body:   strings.Repeat("x", cloudmetadata.MaxBody+1),
			},
			validateFunc: func(out any, err error, _ *metadata) {
				s.ErrorContains(err, "read gce metadata")
				s.NotErrorIs(err, cloudmetadata.ErrNotAvailable)
				s.Nil(out)
			},
		},
		{
			name: "malformed JSON is an error",
			meta: &metadata{status: http.StatusOK, body: "not json"},
			validateFunc: func(out any, err error, _ *metadata) {
				s.ErrorContains(err, "parse gce metadata: ")
				s.Nil(out)
			},
		},
		{
			name: "dmi says not GCE: no request",
			meta: &metadata{status: http.StatusOK, body: cannedResponse},
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Product: &dmi.Product{Name: "OptiPlex 3070"}},
			},
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Empty(m.requests)
			},
		},
		{
			name:  "dmi with no product table is not GCE: no request",
			meta:  &metadata{status: http.StatusOK, body: cannedResponse},
			prior: collector.PriorResults{"dmi": &dmi.Info{}},
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Empty(m.requests)
			},
		},
		{
			name:  "dmi that did not run: the metadata service decides",
			meta:  &metadata{status: http.StatusOK, body: cannedResponse},
			prior: collector.PriorResults{},
			validateFunc: func(out any, err error, m *metadata) {
				s.Require().NoError(err)
				s.Len(m.requests, 1)
				info, ok := out.(*gce.Info)
				s.Require().True(ok)
				s.Equal("my-vm", info.Name)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := gce.NewWith(source.Sources{HTTP: tt.meta})
			prior := tt.prior
			if prior == nil {
				prior = gcePrior()
			}
			out, err := c.Collect(context.Background(), prior)
			tt.validateFunc(out, err, tt.meta)
		})
	}
}
