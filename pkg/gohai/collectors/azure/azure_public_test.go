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

package azure_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
)

const cannedResponse = `{
  "compute": {
    "vmId": "abcd-1234",
    "name": "web-1",
    "vmSize": "Standard_D2s_v3",
    "resourceId": "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/web-1",
    "resourceGroupName": "rg",
    "location": "eastus",
    "zone": "1",
    "subscriptionId": "sub-uuid",
    "azEnvironment": "AzurePublicCloud",
    "offer": "UbuntuServer",
    "publisher": "Canonical",
    "sku": "22_04-lts-gen2",
    "version": "22.04.202311010",
    "osType": "Linux",
    "provider": "Microsoft.Compute",
    "licenseType": "Windows_Server",
    "priority": "Spot",
    "evictionPolicy": "Deallocate",
    "placementGroupId": "pg-xxx",
    "platformFaultDomain": "0",
    "platformUpdateDomain": "1",
    "vmScaleSetName": "vmss",
    "tags": "env:prod;owner:sre",
    "tagsList": [{"name": "env", "value": "prod"}],
    "userData": "dXNlcg==",
    "customData": "",
    "isHostCompatibilityLayerVm": false,
    "plan": {"name": "p", "publisher": "pub", "product": "prod"},
    "securityProfile": {
      "secureBootEnabled": "true",
      "virtualTpmEnabled": "true",
      "encryptionAtHost": "false"
    },
    "publicKeys": [{"keyData": "ssh-rsa AAAA", "path": "/home/azureuser/.ssh/authorized_keys"}],
    "storageProfile": {
      "osDisk": {
        "name": "osdisk",
        "diskSizeGB": "30",
        "caching": "ReadWrite",
        "createOption": "FromImage",
        "writeAcceleratorEnabled": "false",
        "managedDisk": {"id": "/disks/osdisk", "storageAccountType": "Premium_LRS"}
      },
      "dataDisks": [{
        "name": "data0",
        "diskSizeGB": "100",
        "caching": "None",
        "createOption": "Empty",
        "writeAcceleratorEnabled": "false",
        "lun": 0,
        "managedDisk": {"id": "/disks/data0", "storageAccountType": "Standard_LRS"}
      }]
    }
  },
  "network": {
    "interface": [{
      "macAddress": "000D3A1122AA",
      "ipv4": {
        "ipAddress": [{"privateIpAddress": "10.0.0.4", "publicIpAddress": "20.1.2.3"}],
        "subnet": [{"address": "10.0.0.0", "prefix": "24"}]
      },
      "ipv6": {
        "ipAddress": [{"privateIpAddress": "fd00::4", "publicIpAddress": "2603::1"}],
        "subnet": [{"address": "fd00::", "prefix": "64"}]
      }
    }]
  }
}`

const negotiationBody = `{"newest-versions": ["2023-07-01", "9999-99-99"]}`

type AzurePublicTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestAzurePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AzurePublicTestSuite))
}

func (s *AzurePublicTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// installWaagent creates a fake waagent binary at a tmp path and
// swaps the package var. Returns restore func.
func (s *AzurePublicTestSuite) installWaagent() func() {
	path := filepath.Join(s.tmpDir, "waagent")
	s.Require().NoError(os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	return azure.SetWaagentPath(path)
}

// writeLeases writes a dhclient leases file containing the given
// content and swaps the package var.
func (s *AzurePublicTestSuite) writeLeases(
	content string,
) func() {
	path := filepath.Join(s.tmpDir, "leases")
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
	return azure.SetDhclientLeasesPath(path)
}

// pointAwayFromWaagent + pointAwayFromLeases point the package vars
// at nonexistent paths so detection short-circuits.
func (s *AzurePublicTestSuite) pointAwayFromWaagent() func() {
	return azure.SetWaagentPath(filepath.Join(s.tmpDir, "no-waagent"))
}

func (s *AzurePublicTestSuite) pointAwayFromLeases() func() {
	return azure.SetDhclientLeasesPath(filepath.Join(s.tmpDir, "no-leases"))
}

func (s *AzurePublicTestSuite) TestInterface() {
	c := azure.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "azure"},
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

// handler simulates Azure's metadata service. GET /metadata/instance
// without api-version returns 400 with newest-versions; with a known
// api-version returns the canned response. `respOverride` lets tests
// force a specific handler.
func handler(
	respOverride func(w http.ResponseWriter, r *http.Request),
	negotiationJSON string,
	negotiationStatus int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if respOverride != nil {
			respOverride(w, r)
			return
		}
		if r.URL.Path != "/metadata/instance" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("api-version") == "" {
			w.WriteHeader(negotiationStatus)
			_, _ = w.Write([]byte(negotiationJSON))
			return
		}
		_, _ = w.Write([]byte(cannedResponse))
	}
}

func (s *AzurePublicTestSuite) TestCollect() {
	tests := []struct {
		name              string
		waagent           bool
		leasesContent     string // when non-empty, writes a dhclient leases file
		noDetection       bool   // when true, skip waagent + leases setup
		negotiationJSON   string
		negotiationStatus int
		overrideHandler   func(w http.ResponseWriter, r *http.Request)
		closed            bool
		wantNil           bool
		wantErr           bool
		wantNoHTTP        bool
		verify            func(s *AzurePublicTestSuite, info *azure.Info, gotAPI string)
	}{
		{
			name:              "happy path with waagent + successful version negotiation",
			waagent:           true,
			negotiationJSON:   negotiationBody,
			negotiationStatus: http.StatusBadRequest,
			verify: func(s *AzurePublicTestSuite, info *azure.Info, gotAPI string) {
				s.Equal("2023-07-01", gotAPI)
				s.Require().NotNil(info)
				s.Equal("abcd-1234", info.VMID)
				s.Equal("eastus", info.Location)
				s.Require().Len(info.Interfaces, 1)
				iface, ok := info.Interfaces["000D3A1122AA"]
				s.Require().True(ok)
				s.Require().NotNil(iface.IPv4)
				s.Equal([]string{"10.0.0.4"}, info.LocalIPv4)
				s.Equal([]string{"20.1.2.3"}, info.PublicIPv4)
			},
		},
		{
			name:              "DHCP option 245 detects Azure without waagent",
			leasesContent:     "lease {\n  option unknown-245 12:34;\n}\n",
			negotiationJSON:   negotiationBody,
			negotiationStatus: http.StatusBadRequest,
			verify: func(s *AzurePublicTestSuite, info *azure.Info, _ string) {
				s.Require().NotNil(info)
				s.Equal("abcd-1234", info.VMID)
			},
		},
		{
			name:          "DHCP leases without signature does not detect",
			leasesContent: "lease { option routers 10.0.0.1; }\n",
			wantNil:       true,
			wantNoHTTP:    true,
		},
		{
			name:        "no waagent + no leases file short-circuits",
			noDetection: true,
			wantNil:     true,
			wantNoHTTP:  true,
		},
		{
			name:              "version negotiation: 404 falls back to latest",
			waagent:           true,
			negotiationJSON:   "",
			negotiationStatus: http.StatusNotFound,
			verify: func(s *AzurePublicTestSuite, info *azure.Info, gotAPI string) {
				s.Equal("2023-07-01", gotAPI)
				s.Require().NotNil(info)
			},
		},
		{
			name:              "version negotiation: malformed JSON falls back to latest",
			waagent:           true,
			negotiationJSON:   "not json",
			negotiationStatus: http.StatusBadRequest,
			verify: func(s *AzurePublicTestSuite, _ *azure.Info, gotAPI string) {
				s.Equal("2023-07-01", gotAPI)
			},
		},
		{
			name:              "version negotiation: empty newest-versions falls back",
			waagent:           true,
			negotiationJSON:   `{"newest-versions":[]}`,
			negotiationStatus: http.StatusBadRequest,
			verify: func(s *AzurePublicTestSuite, _ *azure.Info, gotAPI string) {
				s.Equal("2023-07-01", gotAPI)
			},
		},
		{
			name:              "version negotiation: no intersection falls back",
			waagent:           true,
			negotiationJSON:   `{"newest-versions":["2099-01-01", "2099-02-01"]}`,
			negotiationStatus: http.StatusBadRequest,
			verify: func(s *AzurePublicTestSuite, _ *azure.Info, gotAPI string) {
				s.Equal("2023-07-01", gotAPI)
			},
		},
		{
			name:              "version negotiation: intersection picks latest",
			waagent:           true,
			negotiationJSON:   `{"newest-versions":["2021-02-01","2023-07-01","2019-11-01"]}`,
			negotiationStatus: http.StatusBadRequest,
			verify: func(s *AzurePublicTestSuite, _ *azure.Info, gotAPI string) {
				s.Equal("2023-07-01", gotAPI)
			},
		},
		{
			name:    "404 on main fetch drops silently",
			waagent: true,
			overrideHandler: func(w http.ResponseWriter, _ *http.Request) {
				http.NotFound(w, nil)
			},
			wantNil: true,
		},
		{
			name:    "connection refused drops silently",
			waagent: true,
			closed:  true,
			wantNil: true,
		},
		{
			name:    "malformed main JSON surfaces as error",
			waagent: true,
			overrideHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("api-version") == "" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(negotiationBody))
					return
				}
				_, _ = w.Write([]byte("not json"))
			},
			wantErr: true,
		},
		{
			name:    "empty compute and network skip transform branches",
			waagent: true,
			overrideHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("api-version") == "" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(negotiationBody))
					return
				}
				_, _ = w.Write([]byte(`{}`))
			},
			verify: func(s *AzurePublicTestSuite, info *azure.Info, _ string) {
				s.Require().NotNil(info)
				s.Empty(info.VMID)
				s.Empty(info.Interfaces)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.noDetection {
				defer s.pointAwayFromWaagent()()
				defer s.pointAwayFromLeases()()
			} else if tt.waagent {
				defer s.installWaagent()()
				defer s.pointAwayFromLeases()()
			} else if tt.leasesContent != "" {
				defer s.pointAwayFromWaagent()()
				defer s.writeLeases(tt.leasesContent)()
			}

			var httpCalled bool
			var gotAPI string
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled = true
					if r.URL.Query().Get("api-version") != "" {
						gotAPI = r.URL.Query().Get("api-version")
					}
					handler(tt.overrideHandler, tt.negotiationJSON, tt.negotiationStatus)(w, r)
				},
			))
			if tt.closed {
				srv.Close()
			} else {
				defer srv.Close()
			}

			client := cloudmetadata.New(srv.URL,
				cloudmetadata.WithHeader("Metadata", "true"))
			c := azure.NewWithClient(client)

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
			info, ok := out.(*azure.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info, gotAPI)
			}
		})
	}
}
