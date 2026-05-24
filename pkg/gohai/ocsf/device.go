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

// Device is the OCSF device object — the primary payload of an
// inventory_info event. Standard OCSF attributes are listed first;
// gohai extension (uid 1337) attributes follow.
type Device struct {
	// Standard OCSF device attributes.
	Hostname   string `json:"hostname,omitempty"`
	Name       string `json:"name,omitempty"`
	UID        string `json:"uid,omitempty"`
	Domain     string `json:"domain,omitempty"`
	IP         string `json:"ip,omitempty"`
	Type       string `json:"type,omitempty"`
	TypeID     int    `json:"type_id"`
	Region     string `json:"region,omitempty"`
	Hypervisor string `json:"hypervisor,omitempty"`

	OS                *OS                `json:"os,omitempty"`
	HWInfo            *DeviceHWInfo      `json:"hw_info,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty"`

	// gohai extension (uid 1337) attributes on device.
	FQDN           string            `json:"fqdn,omitempty"`
	MachineID      string            `json:"machine_id,omitempty"`
	InitSystem     string            `json:"init_system,omitempty"`
	BootTime       int64             `json:"boot_time,omitempty"`
	UptimeSeconds  uint64            `json:"uptime_seconds,omitempty"`
	TimezoneName   string            `json:"timezone_name,omitempty"`
	TimezoneOffset int               `json:"timezone_offset,omitempty"`
	VirtRole       string            `json:"virtualization_role,omitempty"`
	VirtSystems    map[string]string `json:"virtualization_systems,omitempty"`
}

// OS is the OCSF os object nested inside device.
type OS struct {
	// Standard OCSF os attributes.
	Name          string `json:"name,omitempty"`
	Version       string `json:"version,omitempty"`
	Type          string `json:"type,omitempty"`
	TypeID        int    `json:"type_id,omitempty"`
	KernelRelease string `json:"kernel_release,omitempty"`
	Build         string `json:"build,omitempty"`
	CPEName       string `json:"cpe_name,omitempty"`

	// gohai extension (uid 1337) attributes on os.
	Family          string `json:"family,omitempty"`
	CPUArchitecture string `json:"cpu_architecture,omitempty"`
	KernelName      string `json:"kernel_name,omitempty"`
	KernelVersion   string `json:"kernel_version,omitempty"`
	DistributionID  string `json:"distribution_id,omitempty"`
	VersionID       string `json:"version_id,omitempty"`
	VersionCodename string `json:"version_codename,omitempty"`
	VariantID       string `json:"variant_id,omitempty"`
}

// DeviceHWInfo is the OCSF device_hw_info object nested inside device.
type DeviceHWInfo struct {
	// Standard OCSF device_hw_info attributes.
	CPUCount         int     `json:"cpu_count,omitempty"`
	CPUCores         int     `json:"cpu_cores,omitempty"`
	CPUType          string  `json:"cpu_type,omitempty"`
	CPUSpeed         float64 `json:"cpu_speed,omitempty"`
	RAMSize          uint64  `json:"ram_size,omitempty"`
	SerialNumber     string  `json:"serial_number,omitempty"`
	BIOSManufacturer string  `json:"bios_manufacturer,omitempty"`
	BIOSVer          string  `json:"bios_ver,omitempty"`
	BIOSDate         string  `json:"bios_date,omitempty"`
	Chassis          string  `json:"chassis,omitempty"`
	UUID             string  `json:"uuid,omitempty"`
	VendorName       string  `json:"vendor_name,omitempty"`

	// gohai extension (uid 1337) attributes on device_hw_info.
	CPUSockets         int               `json:"cpu_sockets,omitempty"`
	CPUVendorID        string            `json:"cpu_vendor_id,omitempty"`
	CPUFamily          string            `json:"cpu_family,omitempty"`
	CPUModelID         string            `json:"cpu_model_id,omitempty"`
	CPUStepping        int32             `json:"cpu_stepping,omitempty"`
	CPUFlags           []string          `json:"cpu_flags,omitempty"`
	CPUVulnerabilities map[string]string `json:"cpu_vulnerabilities,omitempty"`
}

// NetworkInterface is the OCSF network_interface object.
type NetworkInterface struct {
	// Standard OCSF network_interface attributes.
	Name string `json:"name,omitempty"`
	MAC  string `json:"mac,omitempty"`
	IP   string `json:"ip,omitempty"`
	Type string `json:"type,omitempty"`

	// gohai extension (uid 1337) attributes on network_interface.
	MTU    int      `json:"mtu,omitempty"`
	Speed  string   `json:"speed,omitempty"`
	Driver string   `json:"driver,omitempty"`
	Flags  []string `json:"flags,omitempty"`
}
