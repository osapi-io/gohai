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
// reachable. That is the signal that the host isn't running on OCI.
package oci

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudOCI, which re-exports this constant.
const ProviderName = "oci"

// metadataBaseURL is OCI's link-local metadata endpoint.
const metadataBaseURL = "http://169.254.169.254/opc/v2"

// metadataTimeout matches Ohai's 6s read timeout in mixin/oci_metadata.rb.
const metadataTimeout = 6 * time.Second

// metadataAuthHeader / Value is OCI's literal Authorization header.
// It is not a JWT, only the hardcoded string their IMDSv2 expects. Matches
// Ohai's Ohai::Mixin::OciMetadata::OCI_METADATA_HEADERS.
const (
	metadataAuthHeader = "Authorization"
	metadataAuthValue  = "Bearer Oracle"
)

// dmiChassisAssetTag is the chassis.asset_tag value OCI writes.
// Matches Ohai's oci_chassis_asset_tag? regex (/OracleCloud.com/).
const dmiChassisAssetTag = "OracleCloud.com"

// Endpoint paths. Each is a separate JSON document Ohai fetches.
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
	Name                string            `json:"name,omitempty"`
	Hostname            string            `json:"hostname,omitempty"`
	Type                string            `json:"type,omitempty"`
	ShapeConfig         *ShapeConfig      `json:"shape_config,omitempty"`
	ImageID             string            `json:"image_id,omitempty"`
	Region              string            `json:"region,omitempty"`
	CanonicalRegionName string            `json:"canonical_region_name,omitempty"`
	Zone                string            `json:"zone,omitempty"`
	FaultDomain         string            `json:"fault_domain,omitempty"`
	CompartmentID       string            `json:"compartment_id,omitempty"`
	AccountUID          string            `json:"account_uid,omitempty"`
	State               string            `json:"state,omitempty"`
	TimeCreated         int64             `json:"time_created,omitempty"`
	DefinedTags         map[string]any    `json:"defined_tags,omitempty"`
	FreeformTags        map[string]string `json:"freeform_tags,omitempty"`
	RegionInfo          *RegionInfo       `json:"region_info,omitempty"`

	// Agent + availability + lifecycle.
	AgentConfig        *AgentConfig        `json:"agent_config,omitempty"`
	AvailabilityConfig *AvailabilityConfig `json:"availability_config,omitempty"`
	InstancePoolID     string              `json:"instance_pool_id,omitempty"`
	DedicatedVMHostID  string              `json:"dedicated_vm_host_id,omitempty"`

	// Launch-time configuration (boot volume, firmware, IMDS encryption).
	LaunchOptions  *LaunchOptions `json:"launch_options,omitempty"`
	SourceDetails  *SourceDetails `json:"source_details,omitempty"`
	PlatformConfig map[string]any `json:"platform_config,omitempty"`

	// From /vnics: virtual NICs.
	VNICs []VNIC `json:"vnics,omitempty"`

	// From /allVolumeAttachments. Keyed by attachment OCID to match
	// Ohai's metadata.volumes[<id>] = v shape.
	VolumeAttachments map[string]VolumeAttachment `json:"volume_attachments,omitempty"`
}

// AgentConfig carries the Oracle Cloud Agent's enablement state and
// plugin toggles. Populated from compute.agentConfig.
type AgentConfig struct {
	IsManagementDisabled bool `json:"is_management_disabled,omitempty"`
	IsMonitoringDisabled bool `json:"is_monitoring_disabled,omitempty"`
	AllPluginsDisabled   bool `json:"are_all_plugins_disabled,omitempty"`
}

// AvailabilityConfig is OCI's maintenance / recovery preferences.
type AvailabilityConfig struct {
	IsLiveMigrationPreferred bool   `json:"is_live_migration_preferred,omitempty"`
	RecoveryAction           string `json:"recovery_action,omitempty"`
}

// LaunchOptions captures VM launch-time choices (firmware type, boot
// volume backend, in-flight encryption). Populated from compute.launchOptions.
type LaunchOptions struct {
	BootVolumeType                  string `json:"boot_volume_type,omitempty"`
	Firmware                        string `json:"firmware,omitempty"`
	NetworkType                     string `json:"network_type,omitempty"`
	RemoteDataVolumeType            string `json:"remote_data_volume_type,omitempty"`
	IsConsistentVolumeNamingEnabled bool   `json:"is_consistent_volume_naming_enabled,omitempty"`
	IsPVEncryptionInTransitEnabled  bool   `json:"is_pv_encryption_in_transit_enabled,omitempty"`
}

// SourceDetails describes the image the instance booted from.
type SourceDetails struct {
	SourceType          string `json:"source_type,omitempty"`
	ImageID             string `json:"image_id,omitempty"`
	BootVolumeID        string `json:"boot_volume_id,omitempty"`
	BootVolumeSizeInGBs int    `json:"boot_volume_size_in_gbs,omitempty"`
	KMSKeyID            string `json:"kms_key_id,omitempty"`
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
	DefinedTags         map[string]any    `json:"definedTags"`
	FreeformTags        map[string]string `json:"freeformTags"`
	RegionInfo          *rawRegionInfo    `json:"regionInfo"`

	AgentConfig        *rawAgentConfig        `json:"agentConfig"`
	AvailabilityConfig *rawAvailabilityConfig `json:"availabilityConfig"`
	InstancePoolID     string                 `json:"instancePoolId"`
	DedicatedVMHostID  string                 `json:"dedicatedVmHostId"`
	LaunchOptions      *rawLaunchOptions      `json:"launchOptions"`
	SourceDetails      *rawSourceDetails      `json:"sourceDetails"`
	PlatformConfig     map[string]any         `json:"platformConfig"`
}

type rawAgentConfig struct {
	IsManagementDisabled bool `json:"isManagementDisabled"`
	IsMonitoringDisabled bool `json:"isMonitoringDisabled"`
	AllPluginsDisabled   bool `json:"areAllPluginsDisabled"`
}

type rawAvailabilityConfig struct {
	IsLiveMigrationPreferred bool   `json:"isLiveMigrationPreferred"`
	RecoveryAction           string `json:"recoveryAction"`
}

type rawLaunchOptions struct {
	BootVolumeType                  string `json:"bootVolumeType"`
	Firmware                        string `json:"firmware"`
	NetworkType                     string `json:"networkType"`
	RemoteDataVolumeType            string `json:"remoteDataVolumeType"`
	IsConsistentVolumeNamingEnabled bool   `json:"isConsistentVolumeNamingEnabled"`
	IsPVEncryptionInTransitEnabled  bool   `json:"isPvEncryptionInTransitEnabled"`
}

type rawSourceDetails struct {
	SourceType          string `json:"sourceType"`
	ImageID             string `json:"imageId"`
	BootVolumeID        string `json:"bootVolumeId"`
	BootVolumeSizeInGBs int    `json:"bootVolumeSizeInGBs"`
	KMSKeyID            string `json:"kmsKeyId"`
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

// Metadata fetches OCI's three metadata endpoints.
type Metadata struct {
	// HTTP is the transport of the metadata reads. A nil HTTP is the
	// shared transport.
	HTTP http.RoundTripper
}

// Collector is the public interface of the oci collector.
type Collector interface {
	collector.Collector
}

var _ Collector = (*Metadata)(nil)

// New returns the oci collector on the real host.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the oci collector reading the host through src. It reads
// the metadata service through src.HTTP, so a fake host can answer it. A
// zero src.HTTP is the shared transport.
func NewWith(
	src source.Sources,
) Collector {
	return collector.Wire(&Metadata{}, src)
}

// client is the metadata client of c, which reads through c.HTTP.
func (c *Metadata) client() *cloudmetadata.Client {
	return cloudmetadata.New(
		metadataBaseURL,
		cloudmetadata.WithHeader(metadataAuthHeader, metadataAuthValue),
		cloudmetadata.WithTimeout(metadataTimeout),
		cloudmetadata.WithTransport(c.HTTP),
	)
}

// Name returns "oci".
func (*Metadata) Name() string { return "oci" }

// Category returns "cloud".
func (*Metadata) Category() collector.Category { return collector.CategoryCloud }

// DefaultEnabled returns false, because cloud collectors are opt-in.
func (*Metadata) DefaultEnabled() bool { return false }

// Dependencies declares dmi, because OCI's signature is chassis.asset_tag
// containing "OracleCloud.com". Matches Ohai's oci_chassis_asset_tag?.
func (*Metadata) Dependencies() []string { return []string{"dmi"} }

// Collect gates the fetch on a DMI chassis.asset_tag match, then
// fetches the three OCI metadata documents. Any per-endpoint failure
// that wraps ErrNotAvailable (404 on /vnics or /allVolumeAttachments
// on lightweight shapes) is tolerated, and we populate what we can.
// Returns (nil, nil) when the instance doc fetch fails, matching
// "not on OCI" semantics.
func (c *Metadata) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onOCI(prior) {
		return nil, ctx.Err()
	}

	var ri rawInstance
	if err := c.client().GetJSON(ctx, pathInstance, &ri); err != nil {
		if errors.Is(err, cloudmetadata.ErrNotAvailable) {
			return nil, ctx.Err()
		}
		return nil, err
	}
	info := transformInstance(ri)

	// The instance document decides that the host is on OCI. A VNIC or
	// volume document that cannot be read or decoded leaves its part
	// empty, and keeps the instance's facts. A cancelled context is the
	// error of the collection.
	var vnics []rawVNIC
	if err := c.client().GetJSON(ctx, pathVNICs, &vnics); err == nil {
		for _, v := range vnics {
			info.VNICs = append(info.VNICs, VNIC(v))
		}
	} else if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	var vols []rawVolumeAttachment
	if err := c.client().GetJSON(ctx, pathVolumes, &vols); err == nil {
		if len(vols) > 0 {
			info.VolumeAttachments = make(map[string]VolumeAttachment, len(vols))
		}
		for _, v := range vols {
			if v.ID == "" {
				continue
			}
			info.VolumeAttachments[v.ID] = VolumeAttachment(v)
		}
	} else if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	return info, nil
}

// onOCI checks the dmi collector's chassis.asset_tag for the
// "OracleCloud.com" signature. Fails open when dmi wasn't run.
func onOCI(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	// When dmi did not run, as when a caller asked for this collector
	// alone, only the metadata service can tell. A host whose DMI has no
	// such table, such as a container or a host with no
	// /sys/class/dmi, is not this cloud, so it is not probed.
	if !ok {
		return true
	}
	if info == nil || info.Chassis == nil {
		return false
	}
	return strings.Contains(info.Chassis.AssetTag, dmiChassisAssetTag)
}

// transformInstance maps the raw instance response to Info.
func transformInstance(
	r rawInstance,
) *Info {
	info := &Info{
		ID:                  r.ID,
		Name:                r.DisplayName,
		Hostname:            r.Hostname,
		Type:                r.Shape,
		ImageID:             r.Image,
		Region:              r.Region,
		CanonicalRegionName: r.CanonicalRegionName,
		Zone:                r.AvailabilityDomain,
		FaultDomain:         r.FaultDomain,
		CompartmentID:       r.CompartmentID,
		AccountUID:          r.TenantID,
		State:               r.State,
		TimeCreated:         r.TimeCreated,
		DefinedTags:         r.DefinedTags,
		FreeformTags:        r.FreeformTags,
		InstancePoolID:      r.InstancePoolID,
		DedicatedVMHostID:   r.DedicatedVMHostID,
		PlatformConfig:      r.PlatformConfig,
	}
	if r.ShapeConfig != nil {
		sc := ShapeConfig(*r.ShapeConfig)
		info.ShapeConfig = &sc
	}
	if r.RegionInfo != nil {
		ri := RegionInfo(*r.RegionInfo)
		info.RegionInfo = &ri
	}
	if r.AgentConfig != nil {
		ac := AgentConfig(*r.AgentConfig)
		info.AgentConfig = &ac
	}
	if r.AvailabilityConfig != nil {
		av := AvailabilityConfig(*r.AvailabilityConfig)
		info.AvailabilityConfig = &av
	}
	if r.LaunchOptions != nil {
		lo := LaunchOptions(*r.LaunchOptions)
		info.LaunchOptions = &lo
	}
	if r.SourceDetails != nil {
		sd := SourceDetails(*r.SourceDetails)
		info.SourceDetails = &sd
	}
	return info
}
