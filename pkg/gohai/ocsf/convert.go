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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
)

const schemaVersion = "1.8.0"

// FromFacts converts collector-centric Facts into an OCSF
// inventory_info event (class_uid 5001). Standard OCSF attributes
// map directly; gohai extension (uid 1337) attributes carry fields
// OCSF doesn't yet cover.
// OS type IDs as OCSF defines them.
const (
	osTypeWindows = 100
	osTypeLinux   = 200
	osTypeMacOS   = 300
)

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
	d := &Device{TypeID: 0}

	applyIdentity(d, f)
	applyHardware(d, f)
	applyNetworking(d, f)
	applyRuntime(d, f)

	return d
}

// applyIdentity copies what names the machine.
func applyIdentity(
	d *Device,
	f *gohai.Facts,
) {
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
}

// applyHardware copies the hardware description, which is assembled from
// three collectors and reported only when at least one of them ran.
func applyHardware(
	d *Device,
	f *gohai.Facts,
) {
	if f.CPU != nil || f.Memory != nil || f.DMI != nil {
		d.HWInfo = buildHWInfo(f)
	}
}

// applyNetworking copies the interfaces, and the address of the default
// one as the device's own.
func applyNetworking(
	d *Device,
	f *gohai.Facts,
) {
	if f.Network == nil {
		return
	}

	d.NetworkInterfaces = buildNetworkInterfaces(f)

	if f.Network.DefaultInterface != "" {
		d.IP = findPrimaryIP(f)
	}
}

// applyRuntime copies what the machine is doing rather than what it is.
func applyRuntime(
	d *Device,
	f *gohai.Facts,
) {
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

	applyCPUInfo(hw, f)

	if f.Memory != nil {
		hw.RAMSize = f.Memory.Total
	}

	applyDMIInfo(hw, f)

	return hw
}

// applyCPUInfo copies what the cpu collector reported.
func applyCPUInfo(
	hw *DeviceHWInfo,
	f *gohai.Facts,
) {
	if f.CPU == nil {
		return
	}

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

// applyDMIInfo copies the firmware and chassis description. Each of the
// three groups is reported independently by the machine.
func applyDMIInfo(
	hw *DeviceHWInfo,
	f *gohai.Facts,
) {
	if f.DMI == nil {
		return
	}

	if b := f.DMI.BIOS; b != nil {
		hw.BIOSManufacturer = b.Manufacturer
		hw.BIOSVer = b.Ver
		hw.BIOSDate = b.Date
	}

	if p := f.DMI.Product; p != nil {
		hw.SerialNumber = p.SerialNumber
		hw.UUID = p.UUID
		hw.VendorName = p.VendorName
	}

	if c := f.DMI.Chassis; c != nil {
		hw.Chassis = c.Type
	}
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
	// The first provider that reported anything wins; a host is not on
	// two clouds at once.
	switch {
	case f.Ec2 != nil:
		return &Cloud{
			Provider:       "AWS",
			Region:         f.Ec2.Region,
			Zone:           f.Ec2.Zone,
			Account:        &Account{UID: f.Ec2.AccountUID},
			CloudPartition: f.Ec2.CloudPartition,
		}
	case f.Gce != nil:
		return &Cloud{
			Provider:   "GCP",
			Region:     f.Gce.Region,
			Zone:       f.Gce.Zone,
			ProjectUID: f.Gce.ProjectUID,
		}
	case f.Azure != nil:
		return &Cloud{
			Provider:       "Azure",
			Region:         f.Azure.Region,
			Zone:           f.Azure.Zone,
			Account:        &Account{UID: f.Azure.AccountUID},
			CloudPartition: f.Azure.CloudPartition,
		}
	case f.OCI != nil:
		return &Cloud{
			Provider: "OCI",
			Region:   f.OCI.Region,
			Zone:     f.OCI.Zone,
			Account:  &Account{UID: f.OCI.AccountUID},
		}
	case f.Alibaba != nil:
		return &Cloud{
			Provider: "Alibaba Cloud",
			Region:   f.Alibaba.Region,
			Zone:     f.Alibaba.Zone,
			Account:  &Account{UID: f.Alibaba.AccountUID},
		}
	case f.DigitalOcean != nil:
		return &Cloud{
			Provider: "DigitalOcean",
			Region:   f.DigitalOcean.Region,
		}
	case f.OpenStack != nil:
		return &Cloud{
			Provider:   "OpenStack",
			Zone:       f.OpenStack.Zone,
			ProjectUID: f.OpenStack.ProjectUID,
		}
	case f.Scaleway != nil:
		return &Cloud{
			Provider:   "Scaleway",
			Zone:       f.Scaleway.Zone,
			Account:    &Account{UID: f.Scaleway.AccountUID},
			ProjectUID: f.Scaleway.ProjectUID,
		}
	default:
		// Not on a cloud this collector recognises.
		return nil
	}
}

func findPrimaryIP(
	f *gohai.Facts,
) string {
	for _, iface := range f.Network.Interfaces {
		if iface.Name != f.Network.DefaultInterface {
			continue
		}

		return firstIPv4Addr(iface.Addresses)
	}

	return ""
}

// firstIPv4Addr returns the first IPv4 address of an interface.
func firstIPv4Addr(
	addrs []network.Address,
) string {
	for _, addr := range addrs {
		if addr.Family == "inet" {
			return addr.Addr
		}
	}

	return ""
}

func osTypeID(
	osType string,
) int {
	switch osType {
	case "linux":
		return osTypeLinux
	case "darwin":
		return osTypeMacOS
	case "windows":
		return osTypeWindows
	default:
		return 0
	}
}
