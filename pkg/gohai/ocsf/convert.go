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

package ocsf

import (
	"github.com/osapi-io/gohai/pkg/gohai"
)

const schemaVersion = "1.8.0"

// FromFacts converts collector-centric Facts into an OCSF
// inventory_info event (class_uid 5001). Standard OCSF attributes
// map directly; gohai extension (uid 1337) attributes carry fields
// OCSF doesn't yet cover.
func FromFacts(
	f *gohai.Facts,
) *InventoryInfo {
	event := &InventoryInfo{
		ActivityID:  ActivityIDCollect,
		CategoryUID: CategoryUIDDiscovery,
		ClassUID:    ClassUIDInventoryInfo,
		ClassName:   "Device Inventory Info",
		SeverityID:  SeverityIDInfo,
		Time:        f.CollectTime.UnixMilli(),
		TypeUID:     TypeUIDCollect,
		Metadata: &Metadata{
			Version: schemaVersion,
			Product: &Product{
				Name:       "gohai",
				VendorName: "osapi-io",
			},
		},
		Device: buildDevice(f),
		Cloud:  buildCloud(f),
	}

	return event
}

func buildDevice(
	f *gohai.Facts,
) *Device {
	d := &Device{
		TypeID: 0,
	}

	if f.Hostname != nil {
		d.Hostname = f.Hostname.Name
		d.Domain = f.Hostname.Domain
		d.FQDN = f.Hostname.FQDN
	}

	if f.MachineID != nil {
		d.MachineID = f.MachineID.ID
	}

	if f.Platform != nil {
		d.OS = buildOS(f)
	}

	if f.CPU != nil || f.Memory != nil || f.DMI != nil {
		d.HWInfo = buildHWInfo(f)
	}

	if f.Network != nil {
		d.NetworkInterfaces = buildNetworkInterfaces(f)
		if f.Network.DefaultInterface != "" {
			d.IP = findPrimaryIP(f)
		}
	}

	if f.Virtualization != nil {
		d.Hypervisor = f.Virtualization.System
		d.VirtRole = f.Virtualization.Role
		d.VirtSystems = f.Virtualization.Systems
	}

	if f.Init != nil {
		d.InitSystem = f.Init.Name
	}

	if f.Uptime != nil {
		d.BootTime = int64(f.Uptime.BootTime)
		d.UptimeSeconds = f.Uptime.Seconds
	}

	if f.Timezone != nil {
		d.TimezoneName = f.Timezone.Name
		d.TimezoneOffset = f.Timezone.Offset
	}

	return d
}

func buildOS(
	f *gohai.Facts,
) *OS {
	o := &OS{}

	if f.Platform != nil {
		o.Name = f.Platform.Name
		o.Version = f.Platform.Version
		o.Build = f.Platform.Build
		o.Family = f.Platform.Family
		o.CPUArchitecture = f.Platform.CPUArchitecture
		o.Type = f.Platform.OS
		o.TypeID = osTypeID(f.Platform.OS)
	}

	if f.Kernel != nil {
		o.KernelRelease = f.Kernel.Release
		o.KernelName = f.Kernel.Name
		o.KernelVersion = f.Kernel.Version
	}

	if f.OSRelease != nil {
		o.DistributionID = f.OSRelease.ID
		o.VersionID = f.OSRelease.VersionID
		o.VersionCodename = f.OSRelease.VersionCodename
		o.VariantID = f.OSRelease.VariantID
		if o.Name == "" {
			o.Name = f.OSRelease.Name
		}
	}

	if f.Hostnamectl != nil && f.Hostnamectl.OperatingSystemCPEName != "" {
		o.CPEName = f.Hostnamectl.OperatingSystemCPEName
	}

	return o
}

func buildHWInfo(
	f *gohai.Facts,
) *DeviceHWInfo {
	hw := &DeviceHWInfo{}

	if f.CPU != nil {
		hw.CPUCount = f.CPU.Count
		hw.CPUCores = f.CPU.Cores
		hw.CPUType = f.CPU.ModelName
		hw.CPUSpeed = f.CPU.Speed
		hw.CPUSockets = f.CPU.Sockets
		hw.CPUVendorID = f.CPU.VendorID
		hw.CPUFamily = f.CPU.Family
		hw.CPUModelID = f.CPU.ModelID
		hw.CPUStepping = f.CPU.Stepping
		hw.CPUFlags = f.CPU.Flags
		hw.CPUVulnerabilities = f.CPU.Vulnerabilities
	}

	if f.Memory != nil {
		hw.RAMSize = f.Memory.Total
	}

	if f.DMI != nil {
		if f.DMI.BIOS != nil {
			hw.BIOSManufacturer = f.DMI.BIOS.Manufacturer
			hw.BIOSVer = f.DMI.BIOS.Ver
			hw.BIOSDate = f.DMI.BIOS.Date
		}
		if f.DMI.Product != nil {
			hw.SerialNumber = f.DMI.Product.SerialNumber
			hw.UUID = f.DMI.Product.UUID
			hw.VendorName = f.DMI.Product.VendorName
		}
		if f.DMI.Chassis != nil {
			hw.Chassis = f.DMI.Chassis.Type
		}
	}

	return hw
}

func buildNetworkInterfaces(
	f *gohai.Facts,
) []NetworkInterface {
	if f.Network == nil || len(f.Network.Interfaces) == 0 {
		return nil
	}

	out := make([]NetworkInterface, 0, len(f.Network.Interfaces))
	for _, iface := range f.Network.Interfaces {
		ni := NetworkInterface{
			Name:   iface.Name,
			MAC:    iface.MAC,
			MTU:    iface.MTU,
			Speed:  iface.Speed,
			Driver: iface.Driver,
			Flags:  iface.Flags,
		}

		if len(iface.Addresses) > 0 {
			ni.IP = iface.Addresses[0].Addr
		}

		if iface.Encapsulation != "" {
			ni.Type = iface.Encapsulation
		}

		out = append(out, ni)
	}

	return out
}

func buildCloud(
	f *gohai.Facts,
) *Cloud {
	c := &Cloud{}
	found := false

	if f.Ec2 != nil {
		found = true
		c.Provider = "AWS"
		c.Region = f.Ec2.Region
		c.Zone = f.Ec2.Zone
		c.Account = &Account{UID: f.Ec2.AccountUID}
		c.CloudPartition = f.Ec2.CloudPartition
	} else if f.Gce != nil {
		found = true
		c.Provider = "GCP"
		c.Region = f.Gce.Region
		c.Zone = f.Gce.Zone
		c.ProjectUID = f.Gce.ProjectUID
	} else if f.Azure != nil {
		found = true
		c.Provider = "Azure"
		c.Region = f.Azure.Region
		c.Zone = f.Azure.Zone
		c.Account = &Account{UID: f.Azure.AccountUID}
		c.CloudPartition = f.Azure.CloudPartition
	} else if f.OCI != nil {
		found = true
		c.Provider = "OCI"
		c.Region = f.OCI.Region
		c.Zone = f.OCI.Zone
		c.Account = &Account{UID: f.OCI.AccountUID}
	} else if f.Alibaba != nil {
		found = true
		c.Provider = "Alibaba Cloud"
		c.Region = f.Alibaba.Region
		c.Zone = f.Alibaba.Zone
		c.Account = &Account{UID: f.Alibaba.AccountUID}
	} else if f.DigitalOcean != nil {
		found = true
		c.Provider = "DigitalOcean"
		c.Region = f.DigitalOcean.Region
	} else if f.OpenStack != nil {
		found = true
		c.Provider = "OpenStack"
		c.Zone = f.OpenStack.Zone
		c.ProjectUID = f.OpenStack.ProjectUID
	} else if f.Scaleway != nil {
		found = true
		c.Provider = "Scaleway"
		c.Zone = f.Scaleway.Zone
		c.Account = &Account{UID: f.Scaleway.AccountUID}
		c.ProjectUID = f.Scaleway.ProjectUID
	}

	if !found {
		return nil
	}

	return c
}

func findPrimaryIP(
	f *gohai.Facts,
) string {
	if f.Network == nil {
		return ""
	}

	for _, iface := range f.Network.Interfaces {
		if iface.Name == f.Network.DefaultInterface {
			for _, addr := range iface.Addresses {
				if addr.Family == "inet" {
					return addr.Addr
				}
			}
		}
	}

	return ""
}

func osTypeID(
	osType string,
) int {
	switch osType {
	case "linux":
		return 200
	case "darwin":
		return 300
	case "windows":
		return 100
	default:
		return 0
	}
}
