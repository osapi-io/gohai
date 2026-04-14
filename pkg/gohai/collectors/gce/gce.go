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

// Package gce collects instance and project metadata from the Google
// Compute Engine metadata server (http://metadata.google.internal/).
// The collector returns nil with no error when the endpoint is not
// reachable — that's the signal that the host isn't running on GCE.
package gce

import (
	"context"
	"errors"
	"strings"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

// dmiProductName is the SMBIOS product_name signature GCE VMs
// advertise. Matches Ohai's has_gce_dmi? substring check against
// /sys/class/dmi/id/product_name.
const dmiProductName = "Google Compute Engine"

// metadataBaseURL is GCE's link-local metadata service. The trailing
// dot on the hostname defeats the host's DNS search path (matches
// Ohai's Ohai::Mixin::GCEMetadata::GCE_METADATA_ADDR).
const metadataBaseURL = "http://metadata.google.internal./computeMetadata/v1"

// metadataFlavorHeader is required by GCE — without it the service
// rejects the request with 403. Protects against lateral SSRF-style
// requests that wouldn't know to set it.
const metadataFlavorHeader = "Metadata-Flavor"

// Info is the GCE view surfaced by gohai — every field from GCE's
// `?recursive=true` response, flattened to match the rest of the
// codebase's single-top-level-struct convention. Resource paths
// (machineType, zone, image, network) are normalized to their short
// forms; everything else is passed through verbatim.
type Info struct {
	// Identity.
	InstanceID  int64    `json:"instance_id"`
	Name        string   `json:"name"`
	Hostname    string   `json:"hostname"`
	CPUPlatform string   `json:"cpu_platform,omitempty"`
	MachineType string   `json:"machine_type"`
	Image       string   `json:"image,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Scheduling.
	Preemptible       bool   `json:"preemptible"`
	AutomaticRestart  string `json:"automatic_restart,omitempty"`
	OnHostMaintenance string `json:"on_host_maintenance,omitempty"`
	MaintenanceEvent  string `json:"maintenance_event,omitempty"`

	// Location.
	Zone   string `json:"zone"`
	Region string `json:"region"`

	// Project.
	ProjectID         string            `json:"project_id"`
	NumericProjectID  int64             `json:"numeric_project_id"`
	ProjectAttributes map[string]string `json:"project_attributes,omitempty"`

	// Licensing — raw GCP license IDs attached to the VM.
	Licenses []string `json:"licenses,omitempty"`

	// Instance metadata attributes. Often contains user-uploaded SSH
	// keys and startup scripts — same content Ohai surfaces under
	// node['gce']['instance']['attributes'].
	Attributes map[string]string `json:"attributes,omitempty"`

	// Network interfaces.
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty"`

	// Attached disks.
	Disks []Disk `json:"disks,omitempty"`

	// Service accounts attached to the VM.
	ServiceAccounts []ServiceAccount `json:"service_accounts,omitempty"`
}

// NetworkInterface is one attached VNIC. All fields GCE reports,
// including advanced networking bits (ipAliases, forwardedIps, mtu).
type NetworkInterface struct {
	IP                string         `json:"ip"`
	MAC               string         `json:"mac"`
	Network           string         `json:"network"`
	Subnetmask        string         `json:"subnetmask,omitempty"`
	Gateway           string         `json:"gateway,omitempty"`
	DNSServers        []string       `json:"dns_servers,omitempty"`
	IPAliases         []string       `json:"ip_aliases,omitempty"`
	ForwardedIPs      []string       `json:"forwarded_ips,omitempty"`
	TargetInstanceIPs []string       `json:"target_instance_ips,omitempty"`
	MTU               int            `json:"mtu,omitempty"`
	AccessConfigs     []AccessConfig `json:"access_configs,omitempty"`
}

// AccessConfig is one external-access config on a VNIC — the public
// IP and NAT type. A typical VM has zero or one; ILB-fronted VMs and
// multi-NIC configs can have several.
type AccessConfig struct {
	ExternalIP string `json:"external_ip"`
	Type       string `json:"type,omitempty"`
}

// Disk is one attached disk, with the encryption and bus-interface
// fields that matter to compliance / performance auditing.
type Disk struct {
	DeviceName string `json:"device_name"`
	Type       string `json:"type"`
	Mode       string `json:"mode"`
	Index      int    `json:"index"`
	Interface  string `json:"interface,omitempty"` // "SCSI" / "NVME"
	Encrypted  bool   `json:"encrypted,omitempty"`
}

// ServiceAccount is one attached GCP service account. Key is the
// map key GCE uses (usually "default" or the SA email); Email is the
// underlying identity; Aliases and Scopes cover OAuth details for IAM
// auditing.
type ServiceAccount struct {
	Key     string   `json:"key"`
	Email   string   `json:"email"`
	Aliases []string `json:"aliases,omitempty"`
	Scopes  []string `json:"scopes,omitempty"`
}

// rawResponse is the shape of GCE's `?recursive=true` JSON, unmarshalled
// verbatim so we can transform it into our curated Info.
type rawResponse struct {
	Instance rawInstance `json:"instance"`
	Project  rawProject  `json:"project"`
}

type rawInstance struct {
	ID                int64                        `json:"id"`
	Name              string                       `json:"name"`
	Hostname          string                       `json:"hostname"`
	Zone              string                       `json:"zone"`
	MachineType       string                       `json:"machineType"`
	CPUPlatform       string                       `json:"cpuPlatform"`
	Image             string                       `json:"image"`
	Tags              []string                     `json:"tags"`
	Disks             []rawDisk                    `json:"disks"`
	NetworkInterfaces []rawNetworkInterface        `json:"networkInterfaces"`
	ServiceAccounts   map[string]rawServiceAccount `json:"serviceAccounts"`
	Scheduling        rawScheduling                `json:"scheduling"`
	Description       string                       `json:"description"`
	Attributes        map[string]string            `json:"attributes"`
	Licenses          []rawLicense                 `json:"licenses"`
	MaintenanceEvent  string                       `json:"maintenanceEvent"`
}

type rawDisk struct {
	DeviceName string `json:"deviceName"`
	Index      int    `json:"index"`
	Mode       string `json:"mode"`
	Type       string `json:"type"`
	Interface  string `json:"interface"`
	Encrypted  bool   `json:"encrypted"`
}

type rawNetworkInterface struct {
	IP                string            `json:"ip"`
	MAC               string            `json:"mac"`
	Network           string            `json:"network"`
	Subnetmask        string            `json:"subnetmask"`
	Gateway           string            `json:"gateway"`
	DNSServers        []string          `json:"dnsServers"`
	IPAliases         []string          `json:"ipAliases"`
	ForwardedIPs      []string          `json:"forwardedIps"`
	TargetInstanceIPs []string          `json:"targetInstanceIps"`
	MTU               int               `json:"mtu"`
	AccessConfigs     []rawAccessConfig `json:"accessConfigs"`
}

type rawAccessConfig struct {
	ExternalIP string `json:"externalIp"`
	Type       string `json:"type"`
}

type rawServiceAccount struct {
	Email   string   `json:"email"`
	Aliases []string `json:"aliases"`
	Scopes  []string `json:"scopes"`
}

type rawScheduling struct {
	Preemptible       string `json:"preemptible"` // "TRUE" / "FALSE"
	AutomaticRestart  string `json:"automaticRestart"`
	OnHostMaintenance string `json:"onHostMaintenance"`
}

type rawLicense struct {
	ID string `json:"id"`
}

type rawProject struct {
	ProjectID        string            `json:"projectId"`
	NumericProjectID int64             `json:"numericProjectId"`
	Attributes       map[string]string `json:"attributes"`
}

// Collector fetches the GCE metadata tree via the cloudmetadata helper.
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at GCE's metadata server.
func New() *Collector {
	return NewWithClient(
		cloudmetadata.New(
			metadataBaseURL,
			cloudmetadata.WithHeader(metadataFlavorHeader, "Google"),
		),
	)
}

// NewWithClient returns a Collector backed by a caller-supplied
// cloudmetadata.Client. Tests point the client at an httptest.Server.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return &Collector{client: c}
}

// Name returns "gce".
func (*Collector) Name() string { return "gce" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies declares dmi — the DMI pre-check lets us skip the 2s
// metadata-endpoint timeout on non-GCE hosts. Matches Ohai's
// has_gce_dmi? gate in lib/ohai/plugins/gce.rb.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect gates the metadata fetch on a DMI product_name match.
// Returns (nil, nil) when we're not on GCE or when the endpoint is
// unreachable — either way the Gce field drops cleanly from Facts.
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onGCE(prior) {
		return nil, nil
	}
	var raw rawResponse
	if err := c.client.GetJSON(ctx, "/?recursive=true", &raw); err != nil {
		if errors.Is(err, cloudmetadata.ErrNotAvailable) {
			return nil, nil
		}
		return nil, err
	}
	return transform(raw), nil
}

// onGCE returns true when the dmi collector's product.name indicates
// Google Compute Engine. When dmi is absent from prior (disabled by
// the user, or the dep chain otherwise skipped it) we fail open and
// try the metadata endpoint — the endpoint probe is itself a valid
// detection, just slower on non-GCE hosts.
func onGCE(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil || info.Product == nil {
		return true
	}
	return strings.Contains(info.Product.Name, dmiProductName)
}

// transform reshapes GCE's raw response into the curated Info we expose.
// GCE returns full resource paths (projects/.../zones/us-central1-a);
// consumers want the short form (us-central1-a), so we normalize here.
func transform(
	r rawResponse,
) *Info {
	info := &Info{
		InstanceID:        r.Instance.ID,
		Name:              r.Instance.Name,
		Hostname:          r.Instance.Hostname,
		CPUPlatform:       r.Instance.CPUPlatform,
		MachineType:       lastSegment(r.Instance.MachineType),
		Image:             lastSegment(r.Instance.Image),
		Description:       r.Instance.Description,
		Tags:              r.Instance.Tags,
		Preemptible:       strings.EqualFold(r.Instance.Scheduling.Preemptible, "TRUE"),
		AutomaticRestart:  r.Instance.Scheduling.AutomaticRestart,
		OnHostMaintenance: r.Instance.Scheduling.OnHostMaintenance,
		MaintenanceEvent:  r.Instance.MaintenanceEvent,
		Zone:              lastSegment(r.Instance.Zone),
		ProjectID:         r.Project.ProjectID,
		NumericProjectID:  r.Project.NumericProjectID,
		ProjectAttributes: r.Project.Attributes,
		Attributes:        r.Instance.Attributes,
	}
	info.Region = zoneToRegion(info.Zone)

	for _, lic := range r.Instance.Licenses {
		info.Licenses = append(info.Licenses, lic.ID)
	}

	for _, d := range r.Instance.Disks {
		info.Disks = append(info.Disks, Disk{
			DeviceName: d.DeviceName,
			Type:       d.Type,
			Mode:       d.Mode,
			Index:      d.Index,
			Interface:  d.Interface,
			Encrypted:  d.Encrypted,
		})
	}
	for _, n := range r.Instance.NetworkInterfaces {
		iface := NetworkInterface{
			IP:                n.IP,
			MAC:               n.MAC,
			Network:           lastSegment(n.Network),
			Subnetmask:        n.Subnetmask,
			Gateway:           n.Gateway,
			DNSServers:        n.DNSServers,
			IPAliases:         n.IPAliases,
			ForwardedIPs:      n.ForwardedIPs,
			TargetInstanceIPs: n.TargetInstanceIPs,
			MTU:               n.MTU,
		}
		for _, ac := range n.AccessConfigs {
			iface.AccessConfigs = append(iface.AccessConfigs, AccessConfig(ac))
		}
		info.NetworkInterfaces = append(info.NetworkInterfaces, iface)
	}
	for key, sa := range r.Instance.ServiceAccounts {
		info.ServiceAccounts = append(info.ServiceAccounts, ServiceAccount{
			Key:     key,
			Email:   sa.Email,
			Aliases: sa.Aliases,
			Scopes:  sa.Scopes,
		})
	}
	return info
}

// lastSegment returns the piece after the final "/" — GCE metadata
// reports machineType / zone / image / network as full resource paths
// (projects/X/zones/us-central1-a) but consumers want short names.
func lastSegment(
	s string,
) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// zoneToRegion strips the trailing "-<letter>" from a GCE zone to
// produce its region. "us-central1-a" → "us-central1". Empty in,
// empty out — the collector returns empty for missing data.
func zoneToRegion(
	zone string,
) string {
	if i := strings.LastIndex(zone, "-"); i >= 0 {
		return zone[:i]
	}
	return zone
}
