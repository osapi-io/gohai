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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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
    "customData": "Y3VzdG9t",
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
    },
    "host": {"id": "/hosts/host-1"},
    "hostGroup": {"id": "/hostgroups/hg-1"},
    "osProfile": {
      "adminUsername": "azureuser",
      "computerName": "web-1",
      "disablePasswordAuthentication": "true"
    },
    "additionalCapabilities": {"hibernationEnabled": "true"},
    "extendedLocation": {"name": "losangeles", "type": "EdgeZone"}
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

// waagent is a host with the Azure Linux Agent at its canonical path.
var waagent = map[string]string{"/usr/sbin/waagent": "#!/bin/sh\n"}

// metadata is Azure's metadata service. A request with no api-version
// is the version negotiation, and gets negotiation; a request with one
// gets instance. err fails each request. http.RoundTripper is the
// standard library's interface, so it has no generated mock. The
// http.Client calls RoundTrip on the caller's goroutine, so the test
// reads requests with no lock.
type metadata struct {
	negotiation answer
	instance    answer
	err         error
	requests    []*http.Request
}

// answer is a status and a body.
type answer struct {
	status int
	body   string
}

// RoundTrip records req and gives the canned answer.
func (m *metadata) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return nil, m.err
	}
	a := m.instance
	if req.URL.Query().Get("api-version") == "" {
		a = m.negotiation
	}

	return &http.Response{
		StatusCode: a.status,
		Status:     http.StatusText(a.status),
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(a.body)),
		Request:    req,
	}, nil
}

// apiVersion is the api-version of the instance read, or "" when there
// was none.
func (m *metadata) apiVersion() string {
	for _, req := range m.requests {
		if v := req.URL.Query().Get("api-version"); v != "" {
			return v
		}
	}

	return ""
}

// azureService answers the negotiation with negotiationBody and the
// instance read with cannedResponse.
func azureService() *metadata {
	return &metadata{
		negotiation: answer{status: http.StatusBadRequest, body: negotiationBody},
		instance:    answer{status: http.StatusOK, body: cannedResponse},
	}
}

type AzurePublicTestSuite struct {
	suite.Suite
}

func TestAzurePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AzurePublicTestSuite))
}

func (s *AzurePublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		validateFunc func(azure.Collector)
	}{
		{
			name: "New gives the azure cloud collector, opt-in, after dmi",
			validateFunc: func(c azure.Collector) {
				s.Equal("azure", c.Name())
				s.Equal(collector.CategoryCloud, c.Category())
				s.False(c.DefaultEnabled())
				s.Equal([]string{"dmi"}, c.Dependencies())
				s.NotNil(c.(*azure.Metadata).FS)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(azure.New())
		})
	}
}

func (s *AzurePublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		files        map[string]string
		prior        collector.PriorResults
		meta         *metadata
		validateFunc func(any, error, *metadata)
	}{
		{
			name:  "happy path with waagent + successful version negotiation",
			files: waagent,
			meta:  azureService(),
			validateFunc: func(out any, err error, m *metadata) {
				s.Require().NoError(err)
				s.Equal("2023-07-01", m.apiVersion())
				s.Equal("true", m.requests[0].Header.Get("Metadata"))
				info, ok := out.(*azure.Info)
				s.Require().True(ok)
				s.Equal("abcd-1234", info.ID)
				s.Equal("eastus", info.Region)
				s.Require().Len(info.Interfaces, 1)
				iface, ok := info.Interfaces["000D3A1122AA"]
				s.Require().True(ok)
				s.Require().NotNil(iface.IPv4)
				s.Equal([]string{"10.0.0.4"}, info.LocalIPv4)
				s.Equal([]string{"20.1.2.3"}, info.PublicIPv4)
				s.Equal(&azure.Host{ID: "/hosts/host-1"}, info.Host)
				s.Equal(&azure.HostGroup{ID: "/hostgroups/hg-1"}, info.HostGroup)
				s.Require().NotNil(info.OSProfile)
				s.Equal("azureuser", info.OSProfile.AdminUsername)
				s.Equal("web-1", info.OSProfile.ComputerName)
				s.Equal("true", info.OSProfile.DisablePasswordAuthentication)
				s.Require().NotNil(info.AdditionalCapabilities)
				s.Equal("true", info.AdditionalCapabilities.HibernationEnabled)
				s.Require().NotNil(info.ExtendedLocation)
				s.Equal("losangeles", info.ExtendedLocation.Name)
				s.Equal("EdgeZone", info.ExtendedLocation.Type)
				// userData and customData often hold secrets. The facts
				// leave them out.
				facts, err := json.Marshal(info)
				s.Require().NoError(err)
				s.NotContains(string(facts), "dXNlcg==")
				s.NotContains(string(facts), "Y3VzdG9t")
			},
		},
		{
			name:  "waagent at /usr/bin detects Azure",
			files: map[string]string{"/usr/bin/waagent": "#!/bin/sh\n"},
			meta:  azureService(),
			validateFunc: func(out any, err error, _ *metadata) {
				s.Require().NoError(err)
				info, ok := out.(*azure.Info)
				s.Require().True(ok)
				s.Equal("abcd-1234", info.ID)
			},
		},
		{
			name: "the DMI chassis asset tag detects Azure with no agent and no lease",
			prior: collector.PriorResults{"dmi": &dmi.Info{
				Chassis: &dmi.Chassis{AssetTag: "7783-7084-3265-9085-8269-3286-77"},
			}},
			meta: azureService(),
			validateFunc: func(out any, err error, _ *metadata) {
				s.Require().NoError(err)
				info, ok := out.(*azure.Info)
				s.Require().True(ok)
				s.Equal("abcd-1234", info.ID)
			},
		},
		{
			name: "another chassis asset tag does not detect",
			prior: collector.PriorResults{"dmi": &dmi.Info{
				Chassis: &dmi.Chassis{AssetTag: "0000-0000"},
			}},
			meta: azureService(),
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Empty(m.requests)
			},
		},
		{
			name: "DHCP option 245 detects Azure without waagent",
			files: map[string]string{
				"/var/lib/dhcp/dhclient.eth0.leases": "lease {\n  option unknown-245 12:34;\n}\n",
			},
			meta: azureService(),
			validateFunc: func(out any, err error, _ *metadata) {
				s.Require().NoError(err)
				info, ok := out.(*azure.Info)
				s.Require().True(ok)
				s.Equal("abcd-1234", info.ID)
			},
		},
		{
			name: "DHCP leases without signature does not detect",
			files: map[string]string{
				"/var/lib/dhcp/dhclient.eth0.leases": "lease { option routers 10.0.0.1; }\n",
			},
			meta: azureService(),
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Empty(m.requests)
			},
		},
		{
			name: "no signal: no request",
			meta: azureService(),
			validateFunc: func(out any, err error, m *metadata) {
				s.NoError(err)
				s.Nil(out)
				s.Empty(m.requests)
			},
		},
		{
			name:  "version negotiation: 404 falls back to latest",
			files: waagent,
			meta: &metadata{
				negotiation: answer{status: http.StatusNotFound},
				instance:    answer{status: http.StatusOK, body: cannedResponse},
			},
			validateFunc: func(out any, err error, m *metadata) {
				s.Require().NoError(err)
				s.Equal("2023-07-01", m.apiVersion())
				s.NotNil(out)
			},
		},
		{
			name:  "version negotiation: malformed JSON falls back to latest",
			files: waagent,
			meta: &metadata{
				negotiation: answer{status: http.StatusBadRequest, body: "not json"},
				instance:    answer{status: http.StatusOK, body: cannedResponse},
			},
			validateFunc: func(_ any, err error, m *metadata) {
				s.NoError(err)
				s.Equal("2023-07-01", m.apiVersion())
			},
		},
		{
			name:  "version negotiation: empty newest-versions falls back",
			files: waagent,
			meta: &metadata{
				negotiation: answer{status: http.StatusBadRequest, body: `{"newest-versions":[]}`},
				instance:    answer{status: http.StatusOK, body: cannedResponse},
			},
			validateFunc: func(_ any, err error, m *metadata) {
				s.NoError(err)
				s.Equal("2023-07-01", m.apiVersion())
			},
		},
		{
			name:  "version negotiation: no intersection falls back",
			files: waagent,
			meta: &metadata{
				negotiation: answer{
					status: http.StatusBadRequest,
					body:   `{"newest-versions":["2099-01-01", "2099-02-01"]}`,
				},
				instance: answer{status: http.StatusOK, body: cannedResponse},
			},
			validateFunc: func(_ any, err error, m *metadata) {
				s.NoError(err)
				s.Equal("2023-07-01", m.apiVersion())
			},
		},
		{
			name:  "version negotiation: the newest supported version, not the fallback",
			files: waagent,
			meta: &metadata{
				negotiation: answer{
					status: http.StatusBadRequest,
					body:   `{"newest-versions":["2019-11-01","2021-02-01","2099-01-01"]}`,
				},
				instance: answer{status: http.StatusOK, body: cannedResponse},
			},
			validateFunc: func(_ any, err error, m *metadata) {
				s.NoError(err)
				s.Equal("2021-02-01", m.apiVersion())
			},
		},
		{
			name:  "404 on main fetch is not Azure",
			files: waagent,
			meta: &metadata{
				negotiation: answer{status: http.StatusBadRequest, body: negotiationBody},
				instance:    answer{status: http.StatusNotFound},
			},
			validateFunc: func(out any, err error, _ *metadata) {
				s.NoError(err)
				s.Nil(out)
			},
		},
		{
			name:  "connection refused is not Azure",
			files: waagent,
			meta:  &metadata{err: errors.New("connection refused")},
			validateFunc: func(out any, err error, _ *metadata) {
				s.NoError(err)
				s.Nil(out)
			},
		},
		{
			name:  "malformed main JSON is an error",
			files: waagent,
			meta: &metadata{
				negotiation: answer{status: http.StatusBadRequest, body: negotiationBody},
				instance:    answer{status: http.StatusOK, body: "not json"},
			},
			validateFunc: func(out any, err error, _ *metadata) {
				s.Error(err)
				s.Nil(out)
			},
		},
		{
			name:  "empty compute and network skip transform branches",
			files: waagent,
			meta: &metadata{
				negotiation: answer{status: http.StatusBadRequest, body: negotiationBody},
				instance:    answer{status: http.StatusOK, body: `{}`},
			},
			validateFunc: func(out any, err error, _ *metadata) {
				s.Require().NoError(err)
				info, ok := out.(*azure.Info)
				s.Require().True(ok)
				s.Empty(info.ID)
				s.Empty(info.Interfaces)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := azure.NewWith(source.Sources{
				FS:     sourcetest.FS(s.T(), tt.files),
				Family: family.Debian,
				HTTP:   tt.meta,
			})
			out, err := c.Collect(context.Background(), tt.prior)
			tt.validateFunc(out, err, tt.meta)
		})
	}
}
