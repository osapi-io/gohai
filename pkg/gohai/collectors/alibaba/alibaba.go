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
// the link-local metadata server at http://100.100.100.200/. Walks
// the metadata tree recursively (matches Ohai's
// Ohai::Mixin::AlibabaMetadata#fetch_metadata) so new fields Alibaba
// adds are surfaced without code changes. Returns nil with no error
// when the endpoint is not reachable.
package alibaba

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

// metadataTimeout matches Ohai's 6s read + keep-alive timeout in
// mixin/alibaba_metadata.rb.
const metadataTimeout = 6 * time.Second

// dmiVendorSignature is the substring Alibaba writes to
// /sys/class/dmi/id/sys_vendor. Matches Ohai's has_ali_dmi?.
const dmiVendorSignature = "Alibaba"

// Info is the Alibaba ECS view. Every field the IMDS exposes under
// /meta-data/ is surfaced as a typed member. Credential-bearing paths
// (ram/security-credentials, user-data, public-keys) are deliberately
// skipped.
type Info struct {
	// Identity.
	InstanceID     string `json:"instance_id"`
	InstanceName   string `json:"instance_name,omitempty"`
	InstanceType   string `json:"instance_type,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	ImageID        string `json:"image_id,omitempty"`
	SerialNumber   string `json:"serial_number,omitempty"`
	NetworkType    string `json:"network_type,omitempty"`
	OwnerAccountID string `json:"owner_account_id,omitempty"`
	SourceAddress  string `json:"source_address,omitempty"`

	// Location.
	Region string `json:"region,omitempty"`
	Zone   string `json:"zone,omitempty"`

	// Virtualization layer (meta-data/instance/).
	VirtualizationSolution        string `json:"virtualization_solution,omitempty"`
	VirtualizationSolutionVersion string `json:"virtualization_solution_version,omitempty"`

	// Spot lifecycle.
	SpotTerminationTime string `json:"spot_termination_time,omitempty"`

	// Network — primary ENI flattened onto the top level.
	MAC          string   `json:"mac,omitempty"`
	PrivateIPv4  string   `json:"private_ipv4,omitempty"`
	PublicIPv4   string   `json:"public_ipv4,omitempty"`
	VPCID        string   `json:"vpc_id,omitempty"`
	VPCCIDRBlock string   `json:"vpc_cidr_block,omitempty"`
	VSwitchID    string   `json:"vswitch_id,omitempty"`
	VSwitchCIDR  string   `json:"vswitch_cidr_block,omitempty"`
	Nameservers  []string `json:"dns_nameservers,omitempty"`
	NTPServers   []string `json:"ntp_servers,omitempty"`

	// All ENIs (including the primary) keyed by MAC.
	NetworkInterfaces map[string]NetworkInterface `json:"network_interfaces,omitempty"`

	// Bandwidth caps (Mbps, when present).
	MaxBandwidthIngress int64 `json:"max_bandwidth_ingress,omitempty"`
	MaxBandwidthEgress  int64 `json:"max_bandwidth_egress,omitempty"`

	// IAM.
	RAMRoleName string `json:"ram_role_name,omitempty"`

	// Attached disks keyed by serial number.
	Disks map[string]Disk `json:"disks,omitempty"`

	// Marketplace info (only on Marketplace-sourced images).
	Marketplace *Marketplace `json:"marketplace,omitempty"`

	// User-defined instance tags.
	Tags map[string]string `json:"tags,omitempty"`
}

// NetworkInterface is one ENI attached to the instance.
type NetworkInterface struct {
	NetworkInterfaceID   string   `json:"network_interface_id,omitempty"`
	PrimaryIPAddress     string   `json:"primary_ip_address,omitempty"`
	PrivateIPv4s         []string `json:"private_ipv4s,omitempty"`
	IPv4Prefixes         []string `json:"ipv4_prefixes,omitempty"`
	Netmask              string   `json:"netmask,omitempty"`
	Gateway              string   `json:"gateway,omitempty"`
	VPCID                string   `json:"vpc_id,omitempty"`
	VPCCIDRBlock         string   `json:"vpc_cidr_block,omitempty"`
	VPCIPv6CIDRBlocks    []string `json:"vpc_ipv6_cidr_blocks,omitempty"`
	VSwitchID            string   `json:"vswitch_id,omitempty"`
	VSwitchCIDRBlock     string   `json:"vswitch_cidr_block,omitempty"`
	VSwitchIPv6CIDRBlock string   `json:"vswitch_ipv6_cidr_block,omitempty"`
	IPv6s                []string `json:"ipv6s,omitempty"`
	IPv6Prefixes         []string `json:"ipv6_prefixes,omitempty"`
	IPv6Gateway          string   `json:"ipv6_gateway,omitempty"`
}

// Disk is one attached disk keyed by its serial number.
type Disk struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Marketplace describes billing metadata Alibaba surfaces on
// Marketplace-sourced images.
type Marketplace struct {
	ProductCode string `json:"product_code,omitempty"`
	ChargeType  string `json:"charge_type,omitempty"`
}

// Collector fetches Alibaba's metadata tree via a recursive walk.
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at Alibaba's metadata server.
func New() *Collector {
	return NewWithClient(
		cloudmetadata.New(
			metadataBaseURL,
			cloudmetadata.WithTimeout(metadataTimeout),
		),
	)
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

// Collect gates the fetch on a DMI sys_vendor match, then walks the
// metadata tree recursively (matches Ohai's fetch_metadata). First-
// call failure returns (nil, nil); subsequent path failures are
// tolerated. `/user-data` is excluded from the walk to avoid
// surfacing cloud-init scripts that may contain credentials.
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onAlibaba(prior) {
		return nil, nil
	}
	tree, err := walk(ctx, c.client, "")
	if err != nil {
		// First probe failed — not on Alibaba (or endpoint down).
		return nil, nil
	}
	return transform(tree), nil
}

// onAlibaba checks the dmi collector's sys_vendor (exposed as
// Product.Vendor by ghw, which reads /sys/class/dmi/id/sys_vendor)
// for the "Alibaba" substring. Fails open when dmi wasn't run.
func onAlibaba(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil || info.Product == nil {
		return true
	}
	return strings.Contains(info.Product.Vendor, dmiVendorSignature)
}

// walk issues a GET against path (relative to the Client's baseURL).
// When the response is a newline-separated directory listing, walk
// recurses into each entry. Leaves are parsed as JSON when possible
// and fall back to raw text — matches Ohai's has_trailing_slash? +
// parse_json fallback idiom.
//
// The first call uses path "" (the `/2016-01-01/` listing). At that
// level only, Ohai explicitly excludes `/user-data` from the walk.
func walk(
	ctx context.Context,
	c *cloudmetadata.Client,
	path string,
) (map[string]any, error) {
	listing, err := c.Get(ctx, "/"+path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	for _, line := range strings.Split(string(listing), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Replicate Ohai's root-level "/user-data" skip.
		if path == "" && strings.TrimRight(line, "/") == "user-data" {
			continue
		}
		key := sanitizeKey(line)
		child := fmt.Sprintf("%s%s", path, line)
		if strings.HasSuffix(line, "/") {
			sub, err := walk(ctx, c, child)
			if err != nil {
				continue
			}
			result[key] = sub
		} else {
			leaf, err := c.Get(ctx, "/"+child)
			if err != nil {
				continue
			}
			var jv any
			if json.Unmarshal(leaf, &jv) == nil {
				result[key] = jv
			} else {
				result[key] = strings.TrimSpace(string(leaf))
			}
		}
	}
	return result, nil
}

// sanitizeKey mirrors Ohai's sanitize_key + trailing-underscore strip.
// Dashes and slashes become underscores; trailing `_` is removed.
func sanitizeKey(
	k string,
) string {
	k = strings.NewReplacer("-", "_", "/", "_").Replace(k)
	return strings.TrimRight(k, "_")
}

// transform extracts the canonical typed fields from the walked tree.
// Every IMDS field surfaces as a typed member; credential-bearing
// paths are never included.
func transform(
	tree map[string]any,
) *Info {
	info := &Info{}
	md, _ := tree["meta_data"].(map[string]any)
	if md == nil {
		return info
	}
	info.InstanceID = strVal(md, "instance_id")
	info.Hostname = strVal(md, "hostname")
	info.Region = strVal(md, "region_id")
	info.Zone = strVal(md, "zone_id")
	info.ImageID = strVal(md, "image_id")
	info.SerialNumber = strVal(md, "serial_number")
	info.NetworkType = strVal(md, "network_type")
	info.OwnerAccountID = strVal(md, "owner_account_id")
	info.SourceAddress = strVal(md, "source_address")
	info.MAC = strVal(md, "mac")
	info.PrivateIPv4 = strVal(md, "private_ipv4")
	info.PublicIPv4 = strVal(md, "eipv4")
	info.VPCID = strVal(md, "vpc_id")
	info.VPCCIDRBlock = strVal(md, "vpc_cidr_block")
	info.VSwitchID = strVal(md, "vswitch_id")
	info.VSwitchCIDR = strVal(md, "vswitch_cidr_block")

	if inst, ok := md["instance"].(map[string]any); ok {
		info.InstanceType = strVal(inst, "instance_type")
		info.InstanceName = strVal(inst, "instance_name")
		info.MaxBandwidthIngress = intVal(inst, "max_netbw_ingress")
		info.MaxBandwidthEgress = intVal(inst, "max_netbw_egress")
		info.VirtualizationSolution = strVal(inst, "virtualization_solution")
		info.VirtualizationSolutionVersion = strVal(inst, "virtualization_solution_version")
		if spot, ok := inst["spot"].(map[string]any); ok {
			info.SpotTerminationTime = strVal(spot, "termination_time")
		}
	}
	if dns, ok := md["dns_conf"].(map[string]any); ok {
		info.Nameservers = splitSpace(strVal(dns, "nameservers"))
	}
	if ntp, ok := md["ntp_conf"].(map[string]any); ok {
		info.NTPServers = splitSpace(strVal(ntp, "ntp_servers"))
	}
	if ram, ok := md["ram"].(map[string]any); ok {
		info.RAMRoleName = strVal(ram, "role_name")
	}
	if img, ok := md["image"].(map[string]any); ok {
		if mp, ok := img["market_place"].(map[string]any); ok {
			m := &Marketplace{
				ProductCode: strVal(mp, "product_code"),
				ChargeType:  strVal(mp, "charge_type"),
			}
			if m.ProductCode != "" || m.ChargeType != "" {
				info.Marketplace = m
			}
		}
	}
	if tags, ok := md["tags"].(map[string]any); ok {
		if inst, ok := tags["instance"].(map[string]any); ok {
			out := make(map[string]string, len(inst))
			for k, v := range inst {
				if s, ok := v.(string); ok {
					out[k] = s
				}
			}
			if len(out) > 0 {
				info.Tags = out
			}
		}
	}
	if disks, ok := md["disks"].(map[string]any); ok {
		out := make(map[string]Disk, len(disks))
		for serial, v := range disks {
			d, ok := v.(map[string]any)
			if !ok {
				continue
			}
			out[serial] = Disk{
				ID:   strVal(d, "id"),
				Name: strVal(d, "name"),
			}
		}
		if len(out) > 0 {
			info.Disks = out
		}
	}
	if net, ok := md["network"].(map[string]any); ok {
		if interfaces, ok := net["interfaces"].(map[string]any); ok {
			if macs, ok := interfaces["macs"].(map[string]any); ok {
				out := make(map[string]NetworkInterface, len(macs))
				for mac, v := range macs {
					e, ok := v.(map[string]any)
					if !ok {
						continue
					}
					out[mac] = NetworkInterface{
						NetworkInterfaceID:   strVal(e, "network_interface_id"),
						PrimaryIPAddress:     strVal(e, "primary_ip_address"),
						PrivateIPv4s:         splitCommaOrSpace(strVal(e, "private_ipv4s")),
						IPv4Prefixes:         splitCommaOrSpace(strVal(e, "ipv4_prefixes")),
						Netmask:              strVal(e, "netmask"),
						Gateway:              strVal(e, "gateway"),
						VPCID:                strVal(e, "vpc_id"),
						VPCCIDRBlock:         strVal(e, "vpc_cidr_block"),
						VPCIPv6CIDRBlocks:    splitCommaOrSpace(strVal(e, "vpc_ipv6_cidr_blocks")),
						VSwitchID:            strVal(e, "vswitch_id"),
						VSwitchCIDRBlock:     strVal(e, "vswitch_cidr_block"),
						VSwitchIPv6CIDRBlock: strVal(e, "vswitch_ipv6_cidr_block"),
						IPv6s:                splitCommaOrSpace(strVal(e, "ipv6s")),
						IPv6Prefixes:         splitCommaOrSpace(strVal(e, "ipv6_prefixes")),
						IPv6Gateway:          strVal(e, "ipv6_gateway"),
					}
				}
				if len(out) > 0 {
					info.NetworkInterfaces = out
				}
			}
		}
	}
	return info
}

// splitCommaOrSpace handles Alibaba's mixed list encoding — some
// fields use comma separation, others use whitespace. Both collapse
// to the same []string.
func splitCommaOrSpace(
	s string,
) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// Commas first; if none, whitespace.
	if strings.Contains(s, ",") {
		out := make([]string, 0)
		for _, part := range strings.Split(s, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return strings.Fields(s)
}

// strVal returns the string value at key, or empty when absent / wrong type.
func strVal(
	m map[string]any,
	key string,
) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// intVal returns the integer value at key. json.Unmarshal parses
// JSON numbers as float64, so that's the only dynamic type we handle
// — anything else (absent, string, nested map) returns 0.
func intVal(
	m map[string]any,
	key string,
) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

// splitSpace splits whitespace-separated entries. Empty / whitespace-
// only input returns nil so Info's `omitempty` suppresses the field.
func splitSpace(
	s string,
) []string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	return parts
}
