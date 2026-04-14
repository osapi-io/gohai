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

// Package oci collects Oracle Cloud Infrastructure instance metadata
// from the link-local metadata server at http://169.254.169.254/opc/v2.
// The collector returns nil with no error when the endpoint is not
// reachable — that's the signal that the host isn't running on OCI.
package oci

import (
	"context"
	"errors"
	"strings"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudOCI, which re-exports this constant.
const ProviderName = "oci"

// metadataBaseURL is OCI's link-local metadata endpoint.
const metadataBaseURL = "http://169.254.169.254/opc/v2"

// metadataAuthHeader / Value is OCI's literal Authorization header.
// Not a JWT — just the hardcoded string their IMDSv2 expects. Matches
// Ohai's Ohai::Mixin::OciMetadata::OCI_METADATA_HEADERS.
const (
	metadataAuthHeader = "Authorization"
	metadataAuthValue  = "Bearer Oracle"
)

// dmiChassisAssetTag is the chassis.asset_tag value OCI writes.
// Matches Ohai's oci_chassis_asset_tag? regex (/OracleCloud.com/).
const dmiChassisAssetTag = "OracleCloud.com"

// Endpoint paths — each is a separate JSON document Ohai fetches.
const (
	pathInstance = "/instance"
	pathVNICs    = "/vnics"
	pathVolumes  = "/allVolumeAttachments"
)

// Info is the OCI view. Flat shape combining fields from OCI's three
// metadata endpoints (instance, vnics, volume attachments).
type Info struct {
	// From /instance.
	ID                  string            `json:"id"`
	DisplayName         string            `json:"display_name,omitempty"`
	Hostname            string            `json:"hostname,omitempty"`
	Shape               string            `json:"shape,omitempty"`
	ShapeConfig         *ShapeConfig      `json:"shape_config,omitempty"`
	Image               string            `json:"image,omitempty"`
	Region              string            `json:"region,omitempty"`
	CanonicalRegionName string            `json:"canonical_region_name,omitempty"`
	AvailabilityDomain  string            `json:"availability_domain,omitempty"`
	FaultDomain         string            `json:"fault_domain,omitempty"`
	CompartmentID       string            `json:"compartment_id,omitempty"`
	TenantID            string            `json:"tenant_id,omitempty"`
	State               string            `json:"state,omitempty"`
	TimeCreated         int64             `json:"time_created,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	DefinedTags         map[string]any    `json:"defined_tags,omitempty"`
	FreeformTags        map[string]string `json:"freeform_tags,omitempty"`
	RegionInfo          *RegionInfo       `json:"region_info,omitempty"`

	// From /vnics — virtual NICs.
	VNICs []VNIC `json:"vnics,omitempty"`

	// From /allVolumeAttachments.
	VolumeAttachments []VolumeAttachment `json:"volume_attachments,omitempty"`
}

// ShapeConfig is the compute shape's resource profile.
type ShapeConfig struct {
	OCPUs                     float64 `json:"ocpus,omitempty"`
	MemoryInGBs               float64 `json:"memory_in_gbs,omitempty"`
	NetworkingBandwidthInGbps float64 `json:"networking_bandwidth_in_gbps,omitempty"`
	MaxVNICAttachments        int     `json:"max_vnic_attachments,omitempty"`
	GPUs                      int     `json:"gpus,omitempty"`
}

// RegionInfo is the geographic identification sub-record.
type RegionInfo struct {
	RealmKey             string `json:"realm_key,omitempty"`
	RealmDomainComponent string `json:"realm_domain_component,omitempty"`
	RegionKey            string `json:"region_key,omitempty"`
	RegionIdentifier     string `json:"region_identifier,omitempty"`
}

// VNIC is one attached virtual network interface.
type VNIC struct {
	VNICID          string `json:"vnic_id"`
	PrivateIP       string `json:"private_ip,omitempty"`
	VLANTag         int    `json:"vlan_tag,omitempty"`
	MACAddr         string `json:"mac_addr,omitempty"`
	VirtualRouterIP string `json:"virtual_router_ip,omitempty"`
	SubnetCIDRBlock string `json:"subnet_cidr_block,omitempty"`
	NICIndex        int    `json:"nic_index,omitempty"`
}

// VolumeAttachment is one attached volume.
type VolumeAttachment struct {
	ID                  string `json:"id"`
	AttachmentType      string `json:"attachment_type,omitempty"`
	DisplayName         string `json:"display_name,omitempty"`
	VolumeID            string `json:"volume_id,omitempty"`
	IsReadOnly          bool   `json:"is_read_only,omitempty"`
	LifecycleState      string `json:"lifecycle_state,omitempty"`
	DevicePath          string `json:"device,omitempty"`
	IQN                 string `json:"iqn,omitempty"`
	IPv4                string `json:"ipv4,omitempty"`
	Port                int    `json:"port,omitempty"`
	EncryptionInTransit bool   `json:"encryption_in_transit,omitempty"`
}

// raw types mirror OCI's response shapes for verbatim unmarshal.
type rawInstance struct {
	ID                  string            `json:"id"`
	DisplayName         string            `json:"displayName"`
	Hostname            string            `json:"hostname"`
	Shape               string            `json:"shape"`
	ShapeConfig         *rawShapeConfig   `json:"shapeConfig"`
	Image               string            `json:"image"`
	Region              string            `json:"region"`
	CanonicalRegionName string            `json:"canonicalRegionName"`
	AvailabilityDomain  string            `json:"availabilityDomain"`
	FaultDomain         string            `json:"faultDomain"`
	CompartmentID       string            `json:"compartmentId"`
	TenantID            string            `json:"tenantId"`
	State               string            `json:"state"`
	TimeCreated         int64             `json:"timeCreated"`
	Metadata            map[string]string `json:"metadata"`
	DefinedTags         map[string]any    `json:"definedTags"`
	FreeformTags        map[string]string `json:"freeformTags"`
	RegionInfo          *rawRegionInfo    `json:"regionInfo"`
}

type rawShapeConfig struct {
	OCPUs                     float64 `json:"ocpus"`
	MemoryInGBs               float64 `json:"memoryInGBs"`
	NetworkingBandwidthInGbps float64 `json:"networkingBandwidthInGbps"`
	MaxVNICAttachments        int     `json:"maxVnicAttachments"`
	GPUs                      int     `json:"gpus"`
}

type rawRegionInfo struct {
	RealmKey             string `json:"realmKey"`
	RealmDomainComponent string `json:"realmDomainComponent"`
	RegionKey            string `json:"regionKey"`
	RegionIdentifier     string `json:"regionIdentifier"`
}

type rawVNIC struct {
	VNICID          string `json:"vnicId"`
	PrivateIP       string `json:"privateIp"`
	VLANTag         int    `json:"vlanTag"`
	MACAddr         string `json:"macAddr"`
	VirtualRouterIP string `json:"virtualRouterIp"`
	SubnetCIDRBlock string `json:"subnetCidrBlock"`
	NICIndex        int    `json:"nicIndex"`
}

type rawVolumeAttachment struct {
	ID                  string `json:"id"`
	AttachmentType      string `json:"attachmentType"`
	DisplayName         string `json:"displayName"`
	VolumeID            string `json:"volumeId"`
	IsReadOnly          bool   `json:"isReadOnly"`
	LifecycleState      string `json:"lifecycleState"`
	DevicePath          string `json:"device"`
	IQN                 string `json:"iqn"`
	IPv4                string `json:"ipv4"`
	Port                int    `json:"port"`
	EncryptionInTransit bool   `json:"encryptionInTransit"`
}

// Collector fetches OCI's three metadata endpoints.
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at OCI's metadata server
// with the required Authorization header pre-configured.
func New() *Collector {
	return NewWithClient(
		cloudmetadata.New(
			metadataBaseURL,
			cloudmetadata.WithHeader(metadataAuthHeader, metadataAuthValue),
		),
	)
}

// NewWithClient returns a Collector backed by a caller-supplied client.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return &Collector{client: c}
}

// Name returns "oci".
func (*Collector) Name() string { return "oci" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies declares dmi — OCI's signature is chassis.asset_tag
// containing "OracleCloud.com". Matches Ohai's oci_chassis_asset_tag?.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect gates the fetch on a DMI chassis.asset_tag match, then
// fetches the three OCI metadata documents. Any per-endpoint failure
// that wraps ErrNotAvailable (404 on /vnics or /allVolumeAttachments
// on lightweight shapes) is tolerated — we populate what we can.
// Returns (nil, nil) when the instance doc fetch fails, matching
// "not on OCI" semantics.
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onOCI(prior) {
		return nil, nil
	}

	var ri rawInstance
	if err := c.client.GetJSON(ctx, pathInstance, &ri); err != nil {
		if errors.Is(err, cloudmetadata.ErrNotAvailable) {
			return nil, nil
		}
		return nil, err
	}
	info := transformInstance(ri)

	var vnics []rawVNIC
	if err := c.client.GetJSON(ctx, pathVNICs, &vnics); err == nil {
		for _, v := range vnics {
			info.VNICs = append(info.VNICs, VNIC(v))
		}
	} else if !errors.Is(err, cloudmetadata.ErrNotAvailable) {
		return nil, err
	}

	var vols []rawVolumeAttachment
	if err := c.client.GetJSON(ctx, pathVolumes, &vols); err == nil {
		for _, v := range vols {
			info.VolumeAttachments = append(info.VolumeAttachments, VolumeAttachment(v))
		}
	} else if !errors.Is(err, cloudmetadata.ErrNotAvailable) {
		return nil, err
	}

	return info, nil
}

// onOCI checks the dmi collector's chassis.asset_tag for the
// "OracleCloud.com" signature. Fails open when dmi wasn't run.
func onOCI(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil || info.Chassis == nil {
		return true
	}
	return strings.Contains(info.Chassis.AssetTag, dmiChassisAssetTag)
}

// transformInstance maps the raw instance response to Info.
func transformInstance(
	r rawInstance,
) *Info {
	info := &Info{
		ID:                  r.ID,
		DisplayName:         r.DisplayName,
		Hostname:            r.Hostname,
		Shape:               r.Shape,
		Image:               r.Image,
		Region:              r.Region,
		CanonicalRegionName: r.CanonicalRegionName,
		AvailabilityDomain:  r.AvailabilityDomain,
		FaultDomain:         r.FaultDomain,
		CompartmentID:       r.CompartmentID,
		TenantID:            r.TenantID,
		State:               r.State,
		TimeCreated:         r.TimeCreated,
		Metadata:            r.Metadata,
		DefinedTags:         r.DefinedTags,
		FreeformTags:        r.FreeformTags,
	}
	if r.ShapeConfig != nil {
		sc := ShapeConfig(*r.ShapeConfig)
		info.ShapeConfig = &sc
	}
	if r.RegionInfo != nil {
		ri := RegionInfo(*r.RegionInfo)
		info.RegionInfo = &ri
	}
	return info
}
