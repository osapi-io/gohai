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

// Package azure collects Azure instance metadata from the link-local
// metadata server at http://169.254.169.254/metadata/instance. The
// collector returns nil with no error when the endpoint is not
// reachable — that's the signal that the host isn't running on Azure.
package azure

import (
	"context"
	"errors"
	"os"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudAzure, which re-exports this constant.
const ProviderName = "azure"

// metadataBaseURL is Azure's link-local metadata endpoint.
const metadataBaseURL = "http://169.254.169.254"

// metadataPath + apiVersion is the canonical full path Azure expects.
// The api-version parameter is required — requests without it return
// 400. Ohai does a version negotiation handshake; we pin to the
// latest Ohai-supported version (2023-07-01) and rely on Azure's
// backward-compatibility for older VMs.
const (
	metadataPath = "/metadata/instance"
	apiVersion   = "2023-07-01"
)

// metadataHeaderKey / metadataHeaderValue is Azure's required
// anti-SSRF header. Matches Ohai's Metadata: true behavior.
const (
	metadataHeaderKey   = "Metadata"
	metadataHeaderValue = "true"
)

// waagentPath is the canonical Azure Linux Agent binary path.
// Existence of this file is Ohai's has_waagent? signal on Linux.
// Package-level var so tests can swap it.
var waagentPath = "/usr/sbin/waagent"

// Info is the Azure view — identity, placement, OS, and network
// merged into the flat struct pattern.
type Info struct {
	// Identity.
	VMID              string `json:"vm_id"`
	Name              string `json:"name,omitempty"`
	VMSize            string `json:"vm_size,omitempty"`
	ResourceID        string `json:"resource_id,omitempty"`
	ResourceGroupName string `json:"resource_group_name,omitempty"`
	VMScaleSetName    string `json:"vm_scale_set_name,omitempty"`
	Priority          string `json:"priority,omitempty"`
	EvictionPolicy    string `json:"eviction_policy,omitempty"`

	// Placement.
	Location             string `json:"location,omitempty"`
	Zone                 string `json:"zone,omitempty"`
	PlacementGroupID     string `json:"placement_group_id,omitempty"`
	PlatformFaultDomain  string `json:"platform_fault_domain,omitempty"`
	PlatformUpdateDomain string `json:"platform_update_domain,omitempty"`

	// Account.
	SubscriptionID string `json:"subscription_id,omitempty"`
	AzEnvironment  string `json:"az_environment,omitempty"`

	// Image.
	Offer          string          `json:"offer,omitempty"`
	Publisher      string          `json:"publisher,omitempty"`
	SKU            string          `json:"sku,omitempty"`
	Version        string          `json:"version,omitempty"`
	LicenseType    string          `json:"license_type,omitempty"`
	OSType         string          `json:"os_type,omitempty"`
	Provider       string          `json:"provider,omitempty"`
	Plan           *Plan           `json:"plan,omitempty"`
	StorageProfile *StorageProfile `json:"storage_profile,omitempty"`

	// Free-form data.
	Tags                     string `json:"tags,omitempty"`
	TagsList                 []Tag  `json:"tags_list,omitempty"`
	UserData                 string `json:"user_data,omitempty"`
	CustomData               string `json:"custom_data,omitempty"`
	IsHostCompatibilityLayer bool   `json:"is_host_compatibility_layer_vm,omitempty"`

	// Security.
	SecurityProfile *SecurityProfile `json:"security_profile,omitempty"`
	PublicKeys      []PublicKey      `json:"public_keys,omitempty"`

	// Network.
	Interfaces []Interface `json:"interfaces,omitempty"`
	PublicIPv4 []string    `json:"public_ipv4,omitempty"`
	LocalIPv4  []string    `json:"local_ipv4,omitempty"`
	PublicIPv6 []string    `json:"public_ipv6,omitempty"`
	LocalIPv6  []string    `json:"local_ipv6,omitempty"`
}

// Plan is the marketplace plan associated with the VM image (if any).
type Plan struct {
	Name      string `json:"name,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	Product   string `json:"product,omitempty"`
}

// StorageProfile is a condensed view of the VM's storage layout.
type StorageProfile struct {
	OSDisk    *Disk  `json:"os_disk,omitempty"`
	DataDisks []Disk `json:"data_disks,omitempty"`
}

// Disk is one OS or data disk.
type Disk struct {
	Name              string       `json:"name,omitempty"`
	DiskSizeGB        string       `json:"disk_size_gb,omitempty"`
	Caching           string       `json:"caching,omitempty"`
	CreateOption      string       `json:"create_option,omitempty"`
	WriteAccelEnabled string       `json:"write_accelerator_enabled,omitempty"`
	ManagedDisk       *ManagedDisk `json:"managed_disk,omitempty"`
	Lun               int          `json:"lun,omitempty"` // data disks only
}

// ManagedDisk is the storage tier for a managed disk.
type ManagedDisk struct {
	ID                 string `json:"id,omitempty"`
	StorageAccountType string `json:"storage_account_type,omitempty"`
}

// Tag is one parsed key=value from the tagsList array.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SecurityProfile carries the VM's security posture flags.
type SecurityProfile struct {
	SecureBootEnabled string `json:"secure_boot_enabled,omitempty"`
	VirtualTpmEnabled string `json:"virtual_tpm_enabled,omitempty"`
	EncryptionAtHost  string `json:"encryption_at_host,omitempty"`
}

// PublicKey is one SSH key attached to the VM via Azure's profile.
type PublicKey struct {
	KeyData string `json:"key_data"`
	Path    string `json:"path,omitempty"`
}

// Interface is one attached network interface.
type Interface struct {
	MACAddress string   `json:"mac_address,omitempty"`
	IPv4       *IPAddrs `json:"ipv4,omitempty"`
	IPv6       *IPAddrs `json:"ipv6,omitempty"`
}

// IPAddrs are the per-address-family sub-objects on an Interface.
type IPAddrs struct {
	IPAddresses []IPAddress `json:"ip_addresses,omitempty"`
	Subnets     []Subnet    `json:"subnets,omitempty"`
}

// IPAddress is one private/public pair on an interface.
type IPAddress struct {
	PrivateIP string `json:"private_ip,omitempty"`
	PublicIP  string `json:"public_ip,omitempty"`
}

// Subnet is one subnet range attached to an interface.
type Subnet struct {
	Address string `json:"address,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
}

// raw mirrors Azure's JSON response shape.
type raw struct {
	Compute *rawCompute `json:"compute"`
	Network *rawNetwork `json:"network"`
}

type rawCompute struct {
	AzEnvironment            string              `json:"azEnvironment"`
	CustomData               string              `json:"customData"`
	EvictionPolicy           string              `json:"evictionPolicy"`
	IsHostCompatibilityLayer bool                `json:"isHostCompatibilityLayerVm"`
	LicenseType              string              `json:"licenseType"`
	Location                 string              `json:"location"`
	Name                     string              `json:"name"`
	Offer                    string              `json:"offer"`
	OSType                   string              `json:"osType"`
	PlacementGroupID         string              `json:"placementGroupId"`
	Plan                     *Plan               `json:"plan"`
	PlatformFaultDomain      string              `json:"platformFaultDomain"`
	PlatformUpdateDomain     string              `json:"platformUpdateDomain"`
	Priority                 string              `json:"priority"`
	Provider                 string              `json:"provider"`
	PublicKeys               []rawPublicKey      `json:"publicKeys"`
	Publisher                string              `json:"publisher"`
	ResourceGroupName        string              `json:"resourceGroupName"`
	ResourceID               string              `json:"resourceId"`
	SecurityProfile          *rawSecurityProfile `json:"securityProfile"`
	SKU                      string              `json:"sku"`
	StorageProfile           *rawStorageProfile  `json:"storageProfile"`
	SubscriptionID           string              `json:"subscriptionId"`
	Tags                     string              `json:"tags"`
	TagsList                 []Tag               `json:"tagsList"`
	UserData                 string              `json:"userData"`
	Version                  string              `json:"version"`
	VMID                     string              `json:"vmId"`
	VMScaleSetName           string              `json:"vmScaleSetName"`
	VMSize                   string              `json:"vmSize"`
	Zone                     string              `json:"zone"`
}

type rawSecurityProfile struct {
	SecureBootEnabled string `json:"secureBootEnabled"`
	VirtualTpmEnabled string `json:"virtualTpmEnabled"`
	EncryptionAtHost  string `json:"encryptionAtHost"`
}

type rawStorageProfile struct {
	OSDisk    *rawDisk  `json:"osDisk"`
	DataDisks []rawDisk `json:"dataDisks"`
}

type rawDisk struct {
	Name              string          `json:"name"`
	DiskSizeGB        string          `json:"diskSizeGB"`
	Caching           string          `json:"caching"`
	CreateOption      string          `json:"createOption"`
	WriteAccelEnabled string          `json:"writeAcceleratorEnabled"`
	ManagedDisk       *rawManagedDisk `json:"managedDisk"`
	Lun               int             `json:"lun"`
}

type rawManagedDisk struct {
	ID                 string `json:"id"`
	StorageAccountType string `json:"storageAccountType"`
}

type rawPublicKey struct {
	KeyData string `json:"keyData"`
	Path    string `json:"path"`
}

type rawNetwork struct {
	Interface []rawInterface `json:"interface"`
}

type rawInterface struct {
	MACAddress string    `json:"macAddress"`
	IPv4       rawIPInfo `json:"ipv4"`
	IPv6       rawIPInfo `json:"ipv6"`
}

type rawIPInfo struct {
	IPAddress []rawIPAddress `json:"ipAddress"`
	Subnet    []rawSubnet    `json:"subnet"`
}

type rawIPAddress struct {
	PrivateIPAddress string `json:"privateIpAddress"`
	PublicIPAddress  string `json:"publicIpAddress"`
}

type rawSubnet struct {
	Address string `json:"address"`
	Prefix  string `json:"prefix"`
}

// Collector fetches Azure's single-JSON metadata response.
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at Azure's metadata server
// with the required Metadata: true header.
func New() *Collector {
	return NewWithClient(
		cloudmetadata.New(
			metadataBaseURL,
			cloudmetadata.WithHeader(metadataHeaderKey, metadataHeaderValue),
		),
	)
}

// NewWithClient returns a Collector backed by a caller-supplied client.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return &Collector{client: c}
}

// Name returns "azure".
func (*Collector) Name() string { return "azure" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies. Azure's Linux detection uses
// the waagent binary's presence; Ohai has no DMI check we mirror.
func (*Collector) Dependencies() []string { return nil }

// Collect gates the fetch on the presence of the Azure Linux Agent
// binary (Ohai's has_waagent?). Returns (nil, nil) when we're not
// on Azure or the endpoint is unreachable.
func (c *Collector) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if !onAzure() {
		return nil, nil
	}
	var r raw
	if err := c.client.GetJSON(ctx, metadataPath+"?api-version="+apiVersion, &r); err != nil {
		if errors.Is(err, cloudmetadata.ErrNotAvailable) {
			return nil, nil
		}
		return nil, err
	}
	return transform(r), nil
}

// onAzure returns true when the Azure Linux Agent binary exists.
// Matches Ohai's has_waagent? on Linux. Non-Linux hosts (macOS,
// Windows) return false — Windows has its own `C:\WindowsAzure`
// directory signal that we don't support yet.
func onAzure() bool {
	_, err := os.Stat(waagentPath)
	return err == nil
}

// transform reshapes Azure's two-section response into the flat Info,
// aggregating per-interface IP addresses into top-level lists.
func transform(
	r raw,
) *Info {
	info := &Info{}
	if r.Compute != nil {
		info.VMID = r.Compute.VMID
		info.Name = r.Compute.Name
		info.VMSize = r.Compute.VMSize
		info.ResourceID = r.Compute.ResourceID
		info.ResourceGroupName = r.Compute.ResourceGroupName
		info.VMScaleSetName = r.Compute.VMScaleSetName
		info.Priority = r.Compute.Priority
		info.EvictionPolicy = r.Compute.EvictionPolicy
		info.Location = r.Compute.Location
		info.Zone = r.Compute.Zone
		info.PlacementGroupID = r.Compute.PlacementGroupID
		info.PlatformFaultDomain = r.Compute.PlatformFaultDomain
		info.PlatformUpdateDomain = r.Compute.PlatformUpdateDomain
		info.SubscriptionID = r.Compute.SubscriptionID
		info.AzEnvironment = r.Compute.AzEnvironment
		info.Offer = r.Compute.Offer
		info.Publisher = r.Compute.Publisher
		info.SKU = r.Compute.SKU
		info.Version = r.Compute.Version
		info.LicenseType = r.Compute.LicenseType
		info.OSType = r.Compute.OSType
		info.Provider = r.Compute.Provider
		info.Plan = r.Compute.Plan
		info.Tags = r.Compute.Tags
		info.TagsList = r.Compute.TagsList
		info.UserData = r.Compute.UserData
		info.CustomData = r.Compute.CustomData
		info.IsHostCompatibilityLayer = r.Compute.IsHostCompatibilityLayer

		if r.Compute.SecurityProfile != nil {
			sp := SecurityProfile(*r.Compute.SecurityProfile)
			info.SecurityProfile = &sp
		}
		for _, pk := range r.Compute.PublicKeys {
			info.PublicKeys = append(info.PublicKeys, PublicKey(pk))
		}
		if r.Compute.StorageProfile != nil {
			info.StorageProfile = &StorageProfile{}
			if r.Compute.StorageProfile.OSDisk != nil {
				d := convertDisk(*r.Compute.StorageProfile.OSDisk)
				info.StorageProfile.OSDisk = &d
			}
			for _, d := range r.Compute.StorageProfile.DataDisks {
				info.StorageProfile.DataDisks = append(
					info.StorageProfile.DataDisks, convertDisk(d))
			}
		}
	}
	if r.Network != nil {
		for _, ri := range r.Network.Interface {
			iface := Interface{MACAddress: ri.MACAddress}
			if len(ri.IPv4.IPAddress) > 0 || len(ri.IPv4.Subnet) > 0 {
				iface.IPv4 = &IPAddrs{}
				for _, a := range ri.IPv4.IPAddress {
					iface.IPv4.IPAddresses = append(iface.IPv4.IPAddresses, IPAddress{
						PrivateIP: a.PrivateIPAddress,
						PublicIP:  a.PublicIPAddress,
					})
					if a.PrivateIPAddress != "" {
						info.LocalIPv4 = append(info.LocalIPv4, a.PrivateIPAddress)
					}
					if a.PublicIPAddress != "" {
						info.PublicIPv4 = append(info.PublicIPv4, a.PublicIPAddress)
					}
				}
				for _, sn := range ri.IPv4.Subnet {
					iface.IPv4.Subnets = append(iface.IPv4.Subnets, Subnet(sn))
				}
			}
			if len(ri.IPv6.IPAddress) > 0 || len(ri.IPv6.Subnet) > 0 {
				iface.IPv6 = &IPAddrs{}
				for _, a := range ri.IPv6.IPAddress {
					iface.IPv6.IPAddresses = append(iface.IPv6.IPAddresses, IPAddress{
						PrivateIP: a.PrivateIPAddress,
						PublicIP:  a.PublicIPAddress,
					})
					if a.PrivateIPAddress != "" {
						info.LocalIPv6 = append(info.LocalIPv6, a.PrivateIPAddress)
					}
					if a.PublicIPAddress != "" {
						info.PublicIPv6 = append(info.PublicIPv6, a.PublicIPAddress)
					}
				}
				for _, sn := range ri.IPv6.Subnet {
					iface.IPv6.Subnets = append(iface.IPv6.Subnets, Subnet(sn))
				}
			}
			info.Interfaces = append(info.Interfaces, iface)
		}
	}
	return info
}

func convertDisk(
	r rawDisk,
) Disk {
	d := Disk{
		Name:              r.Name,
		DiskSizeGB:        r.DiskSizeGB,
		Caching:           r.Caching,
		CreateOption:      r.CreateOption,
		WriteAccelEnabled: r.WriteAccelEnabled,
		Lun:               r.Lun,
	}
	if r.ManagedDisk != nil {
		md := ManagedDisk(*r.ManagedDisk)
		d.ManagedDisk = &md
	}
	return d
}
