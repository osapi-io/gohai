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

// Package alibaba collects Alibaba Cloud ECS instance metadata from
// the link-local metadata server at http://100.100.100.200/. The
// collector returns nil with no error when the endpoint is not
// reachable — that's the signal that the host isn't on Alibaba Cloud.
package alibaba

import (
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudAlibaba, which re-exports this constant.
const ProviderName = "alibaba"

// metadataBaseURL is Alibaba's unusual metadata address — not the
// standard 169.254.169.254 link-local. Matches Ohai's
// Ohai::Mixin::AlibabaMetadata::ALIBABA_METADATA_ADDR constant.
const metadataBaseURL = "http://100.100.100.200/2016-01-01"

// dmiVendorSignature is the substring Alibaba writes to
// /sys/class/dmi/id/sys_vendor. Matches Ohai's has_ali_dmi?.
const dmiVendorSignature = "Alibaba"

// Paths we fetch from the meta-data tree. Order here mirrors the
// left-to-right reading order in Ohai's generated node['alibaba'].
var metadataPaths = []string{
	"/meta-data/hostname",
	"/meta-data/instance-id",
	"/meta-data/region-id",
	"/meta-data/zone-id",
	"/meta-data/image-id",
	"/meta-data/instance/instance-type",
	"/meta-data/instance/instance-name",
	"/meta-data/instance/max-netbw-ingress",
	"/meta-data/instance/max-netbw-egress",
	"/meta-data/mac",
	"/meta-data/private-ipv4",
	"/meta-data/eipv4",
	"/meta-data/vpc-id",
	"/meta-data/vpc-cidr-block",
	"/meta-data/vswitch-id",
	"/meta-data/vswitch-cidr-block",
	"/meta-data/serial-number",
	"/meta-data/network-type",
	"/meta-data/dns-conf/nameservers",
	"/meta-data/ntp-conf/ntp-servers",
	"/meta-data/ram/role-name",
}

// Info is the Alibaba view — identity, location, network, storage.
type Info struct {
	// Identity.
	InstanceID   string `json:"instance_id"`
	InstanceName string `json:"instance_name,omitempty"`
	InstanceType string `json:"instance_type,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	ImageID      string `json:"image_id,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	NetworkType  string `json:"network_type,omitempty"`

	// Location.
	Region string `json:"region,omitempty"`
	Zone   string `json:"zone,omitempty"`

	// Network.
	MAC          string   `json:"mac,omitempty"`
	PrivateIPv4  string   `json:"private_ipv4,omitempty"`
	PublicIPv4   string   `json:"public_ipv4,omitempty"`
	VPCID        string   `json:"vpc_id,omitempty"`
	VPCCIDRBlock string   `json:"vpc_cidr_block,omitempty"`
	VSwitchID    string   `json:"vswitch_id,omitempty"`
	VSwitchCIDR  string   `json:"vswitch_cidr_block,omitempty"`
	Nameservers  []string `json:"dns_nameservers,omitempty"`
	NTPServers   []string `json:"ntp_servers,omitempty"`

	// Bandwidth caps (bytes/sec, when present).
	MaxBandwidthIngress string `json:"max_bandwidth_ingress,omitempty"`
	MaxBandwidthEgress  string `json:"max_bandwidth_egress,omitempty"`

	// IAM.
	RAMRoleName string `json:"ram_role_name,omitempty"`
}

// Collector fetches Alibaba's metadata tree via targeted GETs.
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at Alibaba's metadata server.
func New() *Collector {
	return NewWithClient(cloudmetadata.New(metadataBaseURL))
}

// NewWithClient returns a Collector backed by a caller-supplied client.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return &Collector{client: c}
}

// Name returns "alibaba".
func (*Collector) Name() string { return "alibaba" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies declares dmi — Alibaba writes "Alibaba Cloud" as
// sys_vendor. Matches Ohai's has_ali_dmi? check.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect gates the fetch on a DMI sys_vendor match, then fetches
// each known meta-data path in sequence. Individual failures are
// tolerated — some paths are absent on Classic/non-VPC or lightweight
// instances. A failure on the very first path (hostname) returns
// (nil, nil) with the assumption the endpoint is unreachable.
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onAlibaba(prior) {
		return nil, nil
	}
	values := make(map[string]string, len(metadataPaths))
	for i, p := range metadataPaths {
		body, err := c.client.Get(ctx, p)
		if err != nil {
			if i == 0 {
				// First probe failed outright — not on Alibaba (or
				// endpoint is fully down).
				return nil, nil
			}
			continue
		}
		values[p] = strings.TrimSpace(string(body))
	}
	return transform(values), nil
}

// onAlibaba checks the dmi collector's sys_vendor for the "Alibaba"
// substring. Fails open when dmi wasn't run (endpoint probe will
// still detect or rule out).
func onAlibaba(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil || info.Product == nil {
		return true
	}
	return strings.Contains(info.Product.Vendor, dmiVendorSignature)
}

// transform populates Info from the per-path value map. Each path
// missing from the map leaves its field zero-valued, which produces
// the `omitempty`-suppressed output we want.
func transform(
	v map[string]string,
) *Info {
	return &Info{
		InstanceID:          v["/meta-data/instance-id"],
		InstanceName:        v["/meta-data/instance/instance-name"],
		InstanceType:        v["/meta-data/instance/instance-type"],
		Hostname:            v["/meta-data/hostname"],
		ImageID:             v["/meta-data/image-id"],
		SerialNumber:        v["/meta-data/serial-number"],
		NetworkType:         v["/meta-data/network-type"],
		Region:              v["/meta-data/region-id"],
		Zone:                v["/meta-data/zone-id"],
		MAC:                 v["/meta-data/mac"],
		PrivateIPv4:         v["/meta-data/private-ipv4"],
		PublicIPv4:          v["/meta-data/eipv4"],
		VPCID:               v["/meta-data/vpc-id"],
		VPCCIDRBlock:        v["/meta-data/vpc-cidr-block"],
		VSwitchID:           v["/meta-data/vswitch-id"],
		VSwitchCIDR:         v["/meta-data/vswitch-cidr-block"],
		Nameservers:         splitSpace(v["/meta-data/dns-conf/nameservers"]),
		NTPServers:          splitSpace(v["/meta-data/ntp-conf/ntp-servers"]),
		MaxBandwidthIngress: v["/meta-data/instance/max-netbw-ingress"],
		MaxBandwidthEgress:  v["/meta-data/instance/max-netbw-egress"],
		RAMRoleName:         v["/meta-data/ram/role-name"],
	}
}

// splitSpace splits whitespace-separated DNS/NTP entries. Alibaba
// emits space-separated lists for these paths. Empty / whitespace-only
// input returns nil so Info's `omitempty` JSON tags suppress the field.
func splitSpace(
	s string,
) []string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	return parts
}
