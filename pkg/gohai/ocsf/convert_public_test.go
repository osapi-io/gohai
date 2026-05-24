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

package ocsf_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostnamectl"
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
	"github.com/osapi-io/gohai/pkg/gohai/ocsf"
)

type ConvertPublicTestSuite struct {
	suite.Suite
}

func TestConvertPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConvertPublicTestSuite))
}

func (s *ConvertPublicTestSuite) TestFromFacts() {
	tests := []struct {
		name   string
		facts  *gohai.Facts
		verify func(*ocsf.InventoryInfo)
	}{
		{
			name: "empty facts produces valid event envelope",
			facts: &gohai.Facts{
				CollectTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal(ocsf.ClassUIDInventoryInfo, event.ClassUID)
				s.Equal(ocsf.CategoryUIDDiscovery, event.CategoryUID)
				s.Equal(ocsf.TypeUIDCollect, event.TypeUID)
				s.Equal(ocsf.ActivityIDCollect, event.ActivityID)
				s.Equal(ocsf.SeverityIDInfo, event.SeverityID)
				s.Equal("Device Inventory Info", event.ClassName)
				s.NotZero(event.Time)
				s.NotNil(event.Metadata)
				s.Equal("1.8.0", event.Metadata.Version)
				s.Equal("gohai", event.Metadata.Product.Name)
				s.Equal("osapi-io", event.Metadata.Product.VendorName)
				s.NotNil(event.Device)
				s.Nil(event.Cloud)
			},
		},
		{
			name: "hostname maps to device",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Hostname: &hostname.Info{
					Name:   "web-01",
					Domain: "example.com",
					FQDN:   "web-01.example.com",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("web-01", event.Device.Hostname)
				s.Equal("example.com", event.Device.Domain)
				s.Equal("web-01.example.com", event.Device.FQDN)
			},
		},
		{
			name: "platform maps to device.os",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Platform: &platform.Info{
					OS:              "linux",
					Name:            "Ubuntu",
					Version:         "22.04",
					Family:          "debian",
					CPUArchitecture: "amd64",
					Build:           "SMP",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.NotNil(event.Device.OS)
				s.Equal("Ubuntu", event.Device.OS.Name)
				s.Equal("22.04", event.Device.OS.Version)
				s.Equal("linux", event.Device.OS.Type)
				s.Equal(200, event.Device.OS.TypeID)
				s.Equal("debian", event.Device.OS.Family)
				s.Equal("amd64", event.Device.OS.CPUArchitecture)
				s.Equal("SMP", event.Device.OS.Build)
			},
		},
		{
			name: "darwin os type_id",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Platform:    &platform.Info{OS: "darwin"},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal(300, event.Device.OS.TypeID)
			},
		},
		{
			name: "cpu and memory map to device.hw_info",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				CPU: &cpu.Info{
					Count:     8,
					Cores:     4,
					ModelName: "Intel Xeon",
					Speed:     3200,
					Sockets:   1,
					VendorID:  "GenuineIntel",
					Family:    "6",
					ModelID:   "85",
					Stepping:  7,
					Flags:     []string{"aes", "avx2"},
				},
				Memory: &memory.Info{
					Total: 16384000000,
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				hw := event.Device.HWInfo
				s.NotNil(hw)
				s.Equal(8, hw.CPUCount)
				s.Equal(4, hw.CPUCores)
				s.Equal("Intel Xeon", hw.CPUType)
				s.InDelta(3200, hw.CPUSpeed, 0.01)
				s.Equal(1, hw.CPUSockets)
				s.Equal("GenuineIntel", hw.CPUVendorID)
				s.Equal("6", hw.CPUFamily)
				s.Equal("85", hw.CPUModelID)
				s.Equal(int32(7), hw.CPUStepping)
				s.Equal([]string{"aes", "avx2"}, hw.CPUFlags)
				s.Equal(uint64(16384000000), hw.RAMSize)
			},
		},
		{
			name: "kernel maps to device.os kernel fields",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Platform:    &platform.Info{OS: "linux"},
				Kernel: &kernel.Info{
					Name:    "Linux",
					Release: "5.15.0-76-generic",
					Version: "SMP PREEMPT_DYNAMIC",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("5.15.0-76-generic", event.Device.OS.KernelRelease)
				s.Equal("Linux", event.Device.OS.KernelName)
				s.Equal("SMP PREEMPT_DYNAMIC", event.Device.OS.KernelVersion)
			},
		},
		{
			name: "network interfaces map to device.network_interfaces",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Network: &network.Info{
					DefaultInterface: "eth0",
					Interfaces: []network.Interface{
						{
							Name:  "eth0",
							MAC:   "00:11:22:33:44:55",
							MTU:   1500,
							Flags: []string{"up", "broadcast"},
							Addresses: []network.Address{
								{Addr: "10.0.0.5", Family: "inet"},
							},
						},
					},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Len(event.Device.NetworkInterfaces, 1)
				ni := event.Device.NetworkInterfaces[0]
				s.Equal("eth0", ni.Name)
				s.Equal("00:11:22:33:44:55", ni.MAC)
				s.Equal(1500, ni.MTU)
				s.Equal("10.0.0.5", ni.IP)
				s.Equal("10.0.0.5", event.Device.IP)
			},
		},
		{
			name: "extension fields populated",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				MachineID:   &machineid.Info{ID: "abc-123"},
				Uptime:      &uptime.Info{BootTime: 1700000000, Seconds: 86400},
				Timezone:    &timezone.Info{Name: "America/New_York", Offset: -18000},
				Virtualization: &virtualization.Info{
					System:  "docker",
					Role:    "host",
					Systems: map[string]string{"docker": "host"},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				d := event.Device
				s.Equal("abc-123", d.MachineID)
				s.Equal(int64(1700000000), d.BootTime)
				s.Equal(uint64(86400), d.UptimeSeconds)
				s.Equal("America/New_York", d.TimezoneName)
				s.Equal(-18000, d.TimezoneOffset)
				s.Equal("docker", d.Hypervisor)
				s.Equal("host", d.VirtRole)
				s.Equal(map[string]string{"docker": "host"}, d.VirtSystems)
			},
		},
		{
			name: "no cloud when no provider detected",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Nil(event.Cloud)
			},
		},
		{
			name: "ec2 maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Ec2: &ec2.Info{
					Region:         "us-east-1",
					Zone:           "us-east-1a",
					AccountUID:     "123456789012",
					CloudPartition: "aws",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.NotNil(event.Cloud)
				s.Equal("AWS", event.Cloud.Provider)
				s.Equal("us-east-1", event.Cloud.Region)
				s.Equal("us-east-1a", event.Cloud.Zone)
				s.Equal("123456789012", event.Cloud.Account.UID)
				s.Equal("aws", event.Cloud.CloudPartition)
			},
		},
		{
			name: "gce maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Gce: &gce.Info{
					Region:     "us-central1",
					Zone:       "us-central1-a",
					ProjectUID: "my-project",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("GCP", event.Cloud.Provider)
				s.Equal("my-project", event.Cloud.ProjectUID)
			},
		},
		{
			name: "azure maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Azure: &azure.Info{
					Region:         "eastus",
					Zone:           "1",
					AccountUID:     "sub-123",
					CloudPartition: "AzureCloud",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("Azure", event.Cloud.Provider)
				s.Equal("eastus", event.Cloud.Region)
				s.Equal("sub-123", event.Cloud.Account.UID)
				s.Equal("AzureCloud", event.Cloud.CloudPartition)
			},
		},
		{
			name: "oci maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				OCI: &oci.Info{
					Region:     "us-ashburn-1",
					Zone:       "AD-1",
					AccountUID: "ocid1.tenancy",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("OCI", event.Cloud.Provider)
				s.Equal("ocid1.tenancy", event.Cloud.Account.UID)
			},
		},
		{
			name: "alibaba maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Alibaba: &alibaba.Info{
					Region:     "cn-hangzhou",
					Zone:       "cn-hangzhou-a",
					AccountUID: "ali-123",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("Alibaba Cloud", event.Cloud.Provider)
				s.Equal("ali-123", event.Cloud.Account.UID)
			},
		},
		{
			name: "digital_ocean maps to cloud",
			facts: &gohai.Facts{
				CollectTime:  time.Now(),
				DigitalOcean: &digitalocean.Info{Region: "nyc3"},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("DigitalOcean", event.Cloud.Provider)
				s.Equal("nyc3", event.Cloud.Region)
			},
		},
		{
			name: "openstack maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				OpenStack:   &openstack.Info{Zone: "nova", ProjectUID: "proj-123"},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("OpenStack", event.Cloud.Provider)
				s.Equal("proj-123", event.Cloud.ProjectUID)
			},
		},
		{
			name: "scaleway maps to cloud",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Scaleway: &scaleway.Info{
					Zone:       "fr-par-1",
					AccountUID: "org-123",
					ProjectUID: "proj-456",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("Scaleway", event.Cloud.Provider)
				s.Equal("org-123", event.Cloud.Account.UID)
				s.Equal("proj-456", event.Cloud.ProjectUID)
			},
		},
		{
			name: "dmi maps to hw_info",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				DMI: &dmi.Info{
					BIOS: &dmi.BIOS{Manufacturer: "Dell", Ver: "2.18", Date: "2023-01-15"},
					Product: &dmi.Product{
						SerialNumber: "SN123",
						UUID:         "uuid-456",
						VendorName:   "Dell Inc.",
					},
					Chassis: &dmi.Chassis{Type: "Rack Mount"},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				hw := event.Device.HWInfo
				s.Equal("Dell", hw.BIOSManufacturer)
				s.Equal("2.18", hw.BIOSVer)
				s.Equal("2023-01-15", hw.BIOSDate)
				s.Equal("SN123", hw.SerialNumber)
				s.Equal("uuid-456", hw.UUID)
				s.Equal("Dell Inc.", hw.VendorName)
				s.Equal("Rack Mount", hw.Chassis)
			},
		},
		{
			name: "os_release maps to device.os extension fields",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Platform:    &platform.Info{OS: "linux"},
				OSRelease: &osrelease.Info{
					ID:              "ubuntu",
					VersionID:       "22.04",
					VersionCodename: "jammy",
					VariantID:       "server",
					Name:            "Ubuntu",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("ubuntu", event.Device.OS.DistributionID)
				s.Equal("22.04", event.Device.OS.VersionID)
				s.Equal("jammy", event.Device.OS.VersionCodename)
				s.Equal("server", event.Device.OS.VariantID)
			},
		},
		{
			name: "hostnamectl cpe_name maps to device.os",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Platform:    &platform.Info{OS: "linux"},
				Hostnamectl: &hostnamectl.Info{
					OperatingSystemCPEName: "cpe:/o:canonical:ubuntu:22.04",
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("cpe:/o:canonical:ubuntu:22.04", event.Device.OS.CPEName)
			},
		},
		{
			name: "init maps to device extension field",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Init:        &initd.Info{Name: "systemd"},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("systemd", event.Device.InitSystem)
			},
		},
		{
			name:  "windows os type_id",
			facts: &gohai.Facts{CollectTime: time.Now(), Platform: &platform.Info{OS: "windows"}},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal(100, event.Device.OS.TypeID)
			},
		},
		{
			name:  "unknown os type_id",
			facts: &gohai.Facts{CollectTime: time.Now(), Platform: &platform.Info{OS: "freebsd"}},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal(0, event.Device.OS.TypeID)
			},
		},
		{
			name: "network without default interface returns empty primary IP",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Network: &network.Info{
					Interfaces: []network.Interface{
						{
							Name:      "lo",
							Addresses: []network.Address{{Addr: "127.0.0.1", Family: "inet"}},
						},
					},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Empty(event.Device.IP)
			},
		},
		{
			name: "os_release fills name when platform name is empty",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Platform:    &platform.Info{OS: "linux"},
				OSRelease:   &osrelease.Info{Name: "Alpine Linux"},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("Alpine Linux", event.Device.OS.Name)
			},
		},
		{
			name: "nil network returns nil interfaces",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Nil(event.Device.NetworkInterfaces)
			},
		},
		{
			name: "empty network interfaces returns nil",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Network:     &network.Info{Interfaces: []network.Interface{}},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Nil(event.Device.NetworkInterfaces)
			},
		},
		{
			name: "findPrimaryIP with no inet match returns empty",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Network: &network.Info{
					DefaultInterface: "eth0",
					Interfaces: []network.Interface{
						{
							Name:      "eth0",
							Addresses: []network.Address{{Addr: "fe80::1", Family: "inet6"}},
						},
					},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Empty(event.Device.IP)
			},
		},
		{
			name: "interface without addresses still maps",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Network: &network.Info{
					Interfaces: []network.Interface{
						{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff"},
					},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Len(event.Device.NetworkInterfaces, 1)
				s.Empty(event.Device.NetworkInterfaces[0].IP)
			},
		},
		{
			name: "encapsulation maps to network interface type",
			facts: &gohai.Facts{
				CollectTime: time.Now(),
				Network: &network.Info{
					Interfaces: []network.Interface{
						{Name: "eth0", Encapsulation: "Ethernet"},
					},
				},
			},
			verify: func(event *ocsf.InventoryInfo) {
				s.Equal("Ethernet", event.Device.NetworkInterfaces[0].Type)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			event := ocsf.FromFacts(tc.facts)
			tc.verify(event)
		})
	}
}
