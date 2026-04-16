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

package oci_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
)

func ociPrior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{Chassis: &dmi.Chassis{AssetTag: "OracleCloud.com"}},
	}
}

const instanceResponse = `{
  "id": "ocid1.instance.oc1.phx.xxx",
  "displayName": "prod-1",
  "hostname": "prod-1",
  "shape": "VM.Standard.E4.Flex",
  "shapeConfig": {"ocpus": 2.0, "memoryInGBs": 16.0, "maxVnicAttachments": 2, "gpus": 0},
  "image": "ocid1.image.oc1.phx.yyy",
  "region": "phx",
  "canonicalRegionName": "us-phoenix-1",
  "availabilityDomain": "pPrU:PHX-AD-1",
  "faultDomain": "FAULT-DOMAIN-1",
  "compartmentId": "ocid1.compartment.oc1..zzz",
  "tenantId": "ocid1.tenancy.oc1..www",
  "state": "RUNNING",
  "timeCreated": 1713000000000,
  "metadata": {"ssh_authorized_keys": "ssh-rsa AAA..."},
  "freeformTags": {"env": "prod"},
  "regionInfo": {"realmKey": "oc1", "regionKey": "PHX", "regionIdentifier": "us-phoenix-1"},
  "agentConfig": {"isManagementDisabled": false, "isMonitoringDisabled": false, "areAllPluginsDisabled": false},
  "availabilityConfig": {"isLiveMigrationPreferred": true, "recoveryAction": "RESTORE_INSTANCE"},
  "instancePoolId": "ocid1.instancepool.oc1.phx.pool1",
  "dedicatedVmHostId": "ocid1.dedicatedvmhost.oc1.phx.host1",
  "launchOptions": {
    "bootVolumeType": "PARAVIRTUALIZED",
    "firmware": "UEFI_64",
    "networkType": "PARAVIRTUALIZED",
    "remoteDataVolumeType": "PARAVIRTUALIZED",
    "isConsistentVolumeNamingEnabled": true,
    "isPvEncryptionInTransitEnabled": true
  },
  "sourceDetails": {
    "sourceType": "image",
    "imageId": "ocid1.image.oc1.phx.src",
    "bootVolumeSizeInGBs": 50,
    "kmsKeyId": "ocid1.key.oc1.phx.k1"
  },
  "platformConfig": {"type": "AMD_VM", "isSecureBootEnabled": true}
}`

const vnicsResponse = `[
  {"vnicId": "ocid1.vnic.oc1.phx.aaa", "privateIp": "10.0.1.5", "macAddr": "02:00:17:01:02:03", "subnetCidrBlock": "10.0.1.0/24", "nicIndex": 0, "vlanTag": 2000, "virtualRouterIp": "10.0.1.1"}
]`

const volumesResponse = `[
  {"id": "ocid1.volumeattachment.oc1.phx.bbb", "attachmentType": "paravirtualized", "displayName": "boot", "volumeId": "ocid1.volume.oc1.phx.ccc", "lifecycleState": "ATTACHED", "device": "/dev/oracleoci/oraclevdb"}
]`

type OCIPublicTestSuite struct {
	suite.Suite
}

func TestOCIPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OCIPublicTestSuite))
}

func (s *OCIPublicTestSuite) TestInterface() {
	c := oci.New()
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Name", c.Name(), "oci"},
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

// handlerFunc wires an httptest router that matches OCI's three
// endpoints. Missing paths return 404 so tests exercise the
// tolerate-missing branches.
func handler(
	instance, vnics, volumes string,
	gotAuth *string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*gotAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/instance":
			if instance == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(instance))
		case "/vnics":
			if vnics == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(vnics))
		case "/allVolumeAttachments":
			if volumes == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(volumes))
		default:
			http.NotFound(w, r)
		}
	}
}

func (s *OCIPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		prior      collector.PriorResults
		instance   string
		vnics      string
		volumes    string
		handler    http.HandlerFunc // overrides the default when set
		closed     bool
		wantNil    bool
		wantErr    bool
		wantNoHTTP bool
		verify     func(s *OCIPublicTestSuite, info *oci.Info, gotAuth string)
	}{
		{
			name:     "happy path populates all three sections",
			instance: instanceResponse,
			vnics:    vnicsResponse,
			volumes:  volumesResponse,
			verify: func(s *OCIPublicTestSuite, info *oci.Info, gotAuth string) {
				s.Equal("Bearer Oracle", gotAuth)
				s.Require().NotNil(info)
				s.Equal("ocid1.instance.oc1.phx.xxx", info.ID)
				s.Equal("VM.Standard.E4.Flex", info.Shape)
				s.Require().NotNil(info.ShapeConfig)
				s.InDelta(2.0, info.ShapeConfig.OCPUs, 0.01)
				s.Equal("us-phoenix-1", info.CanonicalRegionName)
				s.Require().NotNil(info.RegionInfo)
				s.Equal("PHX", info.RegionInfo.RegionKey)
				s.Require().Len(info.VNICs, 1)
				s.Equal("10.0.1.5", info.VNICs[0].PrivateIP)
				s.Require().Len(info.VolumeAttachments, 1)
				va, ok := info.VolumeAttachments["ocid1.volumeattachment.oc1.phx.bbb"]
				s.Require().True(ok)
				s.Equal("ATTACHED", va.LifecycleState)

				// New typed fields from Ohai blind compute dump.
				s.Require().NotNil(info.AgentConfig)
				s.False(info.AgentConfig.IsManagementDisabled)
				s.Require().NotNil(info.AvailabilityConfig)
				s.True(info.AvailabilityConfig.IsLiveMigrationPreferred)
				s.Equal("RESTORE_INSTANCE", info.AvailabilityConfig.RecoveryAction)
				s.Equal("ocid1.instancepool.oc1.phx.pool1", info.InstancePoolID)
				s.Equal("ocid1.dedicatedvmhost.oc1.phx.host1", info.DedicatedVMHostID)
				s.Require().NotNil(info.LaunchOptions)
				s.Equal("PARAVIRTUALIZED", info.LaunchOptions.BootVolumeType)
				s.Equal("UEFI_64", info.LaunchOptions.Firmware)
				s.True(info.LaunchOptions.IsPVEncryptionInTransitEnabled)
				s.Require().NotNil(info.SourceDetails)
				s.Equal("image", info.SourceDetails.SourceType)
				s.Equal(50, info.SourceDetails.BootVolumeSizeInGBs)
				s.Equal("AMD_VM", info.PlatformConfig["type"])
			},
		},
		{
			name:     "missing vnics and volumes tolerated",
			instance: instanceResponse,
			verify: func(s *OCIPublicTestSuite, info *oci.Info, _ string) {
				s.Require().NotNil(info)
				s.Empty(info.VNICs)
				s.Empty(info.VolumeAttachments)
			},
		},
		{
			name: "dmi says not OCI short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{Chassis: &dmi.Chassis{AssetTag: "Something Else"}},
			},
			wantNil:    true,
			wantNoHTTP: true,
		},
		{
			name:     "no dmi in prior fails open",
			prior:    collector.PriorResults{},
			instance: instanceResponse,
			verify: func(s *OCIPublicTestSuite, info *oci.Info, _ string) {
				s.Require().NotNil(info)
				s.Equal("VM.Standard.E4.Flex", info.Shape)
			},
		},
		{
			name:    "404 on instance drops silently",
			wantNil: true,
		},
		{
			name:     "volumes with empty id are skipped",
			instance: instanceResponse,
			volumes:  `[{"id": "", "lifecycleState": "ATTACHED"}, {"id": "ocid1.va.oc1.bbb", "lifecycleState": "ATTACHED"}]`,
			verify: func(s *OCIPublicTestSuite, info *oci.Info, _ string) {
				s.Require().Len(info.VolumeAttachments, 1)
				_, ok := info.VolumeAttachments["ocid1.va.oc1.bbb"]
				s.True(ok)
			},
		},
		{
			name:    "connection refused drops silently",
			closed:  true,
			wantNil: true,
		},
		{
			name:     "malformed instance JSON surfaces error",
			instance: "not json",
			wantErr:  true,
		},
		{
			name:     "malformed vnics JSON surfaces error",
			instance: instanceResponse,
			vnics:    "not json",
			wantErr:  true,
		},
		{
			name:     "malformed volumes JSON surfaces error",
			instance: instanceResponse,
			volumes:  "not json",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var gotAuth string
			h := tt.handler
			if h == nil {
				h = handler(tt.instance, tt.vnics, tt.volumes, &gotAuth)
			}
			var httpCalled bool
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

			client := cloudmetadata.New(srv.URL,
				cloudmetadata.WithHeader("Authorization", "Bearer Oracle"))
			c := oci.NewWithClient(client)

			prior := tt.prior
			if prior == nil {
				prior = ociPrior()
			}
			out, err := c.Collect(context.Background(), prior)
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
			info, ok := out.(*oci.Info)
			s.Require().True(ok)
			if tt.verify != nil {
				tt.verify(s, info, gotAuth)
			}
		})
	}
}
