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

func (s *AzurePublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		waagent    bool
		handler    func(w http.ResponseWriter, r *http.Request)
		closed     bool
		wantNil    bool
		wantErr    bool
		wantNoHTTP bool
		verify     func(s *AzurePublicTestSuite, info *azure.Info, gotHdr string, gotAPI string)
	}{
		{
			name:    "happy path",
			waagent: true,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(cannedResponse))
			},
			verify: func(s *AzurePublicTestSuite, info *azure.Info, gotHdr, gotAPI string) {
				s.Equal("true", gotHdr)
				s.Equal("2023-07-01", gotAPI)
				s.Require().NotNil(info)
				s.Equal("abcd-1234", info.VMID)
				s.Equal("Standard_D2s_v3", info.VMSize)
				s.Equal("eastus", info.Location)
				s.Equal("1", info.Zone)
				s.Equal("sub-uuid", info.SubscriptionID)
				s.Require().NotNil(info.SecurityProfile)
				s.Equal("true", info.SecurityProfile.SecureBootEnabled)
				s.Require().Len(info.PublicKeys, 1)
				s.Require().NotNil(info.StorageProfile)
				s.Require().NotNil(info.StorageProfile.OSDisk)
				s.Equal("osdisk", info.StorageProfile.OSDisk.Name)
				s.Require().Len(info.StorageProfile.DataDisks, 1)
				s.Require().Len(info.Interfaces, 1)
				s.Equal("000D3A1122AA", info.Interfaces[0].MACAddress)
				s.Equal([]string{"10.0.0.4"}, info.LocalIPv4)
				s.Equal([]string{"20.1.2.3"}, info.PublicIPv4)
				s.Equal([]string{"fd00::4"}, info.LocalIPv6)
				s.Equal([]string{"2603::1"}, info.PublicIPv6)
			},
		},
		{
			name:       "no waagent short-circuits",
			waagent:    false,
			handler:    func(http.ResponseWriter, *http.Request) {},
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:    "404 drops silently",
			waagent: true,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.NotFound(w, nil)
			},
			wantNil: true,
		},
		{
			name:    "connection refused drops silently",
			waagent: true,
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			wantNil: true,
		},
		{
			name:    "malformed JSON surfaces as error",
			waagent: true,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("not json"))
			},
			wantErr: true,
		},
		{
			name:    "empty compute and network skip all transform branches",
			waagent: true,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			},
			verify: func(s *AzurePublicTestSuite, info *azure.Info, _, _ string) {
				s.Require().NotNil(info)
				s.Empty(info.VMID)
				s.Empty(info.Interfaces)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.waagent {
				defer s.installWaagent()()
			} else {
				defer azure.SetWaagentPath(filepath.Join(s.tmpDir, "missing"))()
			}

			var httpCalled bool
			var gotHdr, gotAPI string
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled = true
					gotHdr = r.Header.Get("Metadata")
					gotAPI = r.URL.Query().Get("api-version")
					tt.handler(w, r)
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
				tt.verify(s, info, gotHdr, gotAPI)
			}
		})
	}
}
