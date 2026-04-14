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

// Package openstack collects OpenStack (Nova) instance metadata from
// the link-local metadata server at http://169.254.169.254/. OpenStack
// intentionally emits an EC2-compatible metadata service so the HTTP
// shape matches ec2's. Returns nil with no error when the endpoint
// isn't reachable or the DMI signature isn't OpenStack.
package openstack

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudOpenStack, which re-exports this constant.
const ProviderName = "openstack"

// metadataBaseURL is the standard IMDS endpoint — OpenStack mirrors
// EC2's. We root at the host so the collector can reach both the
// EC2-mirror tree under /latest/meta-data/ and Nova's enriched
// document under /openstack/latest/meta_data.json.
const metadataBaseURL = "http://169.254.169.254"

// dmiProductSignature is the substring OpenStack writes to
// /sys/class/dmi/id/product_name. Ohai gates on the virtualization
// plugin's openstack flag, which itself looks at DMI — we gate on
// DMI directly for the same effect.
const dmiProductSignature = "OpenStack"

// metadataPaths is the subset of EC2-style meta-data paths OpenStack
// populates. Nova doesn't implement every EC2 path, so this is a
// curated list.
var metadataPaths = []string{
	"/latest/meta-data/ami-id",
	"/latest/meta-data/instance-id",
	"/latest/meta-data/instance-type",
	"/latest/meta-data/hostname",
	"/latest/meta-data/local-hostname",
	"/latest/meta-data/local-ipv4",
	"/latest/meta-data/public-hostname",
	"/latest/meta-data/public-ipv4",
	"/latest/meta-data/placement/availability-zone",
	"/latest/meta-data/reservation-id",
	"/latest/meta-data/kernel-id",
	"/latest/meta-data/ramdisk-id",
	"/latest/meta-data/security-groups",
}

// Info is the OpenStack view. Flat shape; identity + network + a
// few OpenStack-specific fields from the richer meta_data.json.
type Info struct {
	// Identity.
	InstanceID     string `json:"instance_id"`
	InstanceType   string `json:"instance_type,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	LocalHostname  string `json:"local_hostname,omitempty"`
	PublicHostname string `json:"public_hostname,omitempty"`

	// Placement.
	AvailabilityZone string `json:"availability_zone,omitempty"`

	// Network.
	LocalIPv4      string   `json:"local_ipv4,omitempty"`
	PublicIPv4     string   `json:"public_ipv4,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`

	// Image / boot.
	AMIID     string `json:"ami_id,omitempty"`
	KernelID  string `json:"kernel_id,omitempty"`
	RamdiskID string `json:"ramdisk_id,omitempty"`

	// Reservation.
	ReservationID string `json:"reservation_id,omitempty"`

	// From the OpenStack-specific meta_data.json (when present).
	Name       string            `json:"name,omitempty"`
	ProjectID  string            `json:"project_id,omitempty"`
	UUID       string            `json:"uuid,omitempty"`
	MetaData   map[string]string `json:"meta_data,omitempty"`
	PublicKeys map[string]string `json:"public_keys,omitempty"`
}

// openStackMetaDoc mirrors the subset of /openstack/latest/meta_data.json
// we extract. OpenStack ships additional fields (devices, network_info,
// random_seed) that we skip — they're deployment-specific.
type openStackMetaDoc struct {
	UUID             string            `json:"uuid"`
	Name             string            `json:"name"`
	ProjectID        string            `json:"project_id"`
	Hostname         string            `json:"hostname"`
	Meta             map[string]string `json:"meta"`
	PublicKeys       map[string]string `json:"public_keys"`
	AvailabilityZone string            `json:"availability_zone"`
}

// Collector fetches OpenStack's EC2-compatible + Nova-specific docs.
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at OpenStack's metadata
// server.
func New() *Collector {
	return NewWithClient(cloudmetadata.New(metadataBaseURL))
}

// NewWithClient returns a Collector backed by a caller-supplied client.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return &Collector{client: c}
}

// Name returns "openstack".
func (*Collector) Name() string { return "openstack" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies declares dmi — OpenStack writes "OpenStack Nova" or
// "OpenStack Compute" as product_name. Matches Ohai's
// virtualization-plugin gate which itself reads DMI.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect gates the fetch on a DMI product_name match, then walks
// OpenStack's EC2-compatible meta-data paths and the Nova-specific
// meta_data.json document. Individual failures are tolerated; first-
// path failure returns (nil, nil).
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onOpenStack(prior) {
		return nil, nil
	}

	values := make(map[string]string, len(metadataPaths))
	for i, p := range metadataPaths {
		body, err := c.client.Get(ctx, p)
		if err != nil {
			if i == 0 {
				return nil, nil
			}
			continue
		}
		values[p] = strings.TrimSpace(string(body))
	}

	info := transformEC2Paths(values)

	// Nova's enriched JSON is served under /openstack, outside the
	// /latest EC2-mirror tree. The Client's baseURL includes /latest,
	// so this path is actually absolute-from-root — we use a second
	// client rooted at the base address.
	if body, err := c.fetchNovaDoc(ctx); err == nil {
		var doc openStackMetaDoc
		if jerr := json.Unmarshal(body, &doc); jerr == nil {
			info.UUID = doc.UUID
			info.Name = doc.Name
			info.ProjectID = doc.ProjectID
			info.MetaData = doc.Meta
			info.PublicKeys = doc.PublicKeys
			if info.AvailabilityZone == "" {
				info.AvailabilityZone = doc.AvailabilityZone
			}
			if info.Hostname == "" {
				info.Hostname = doc.Hostname
			}
		}
	}
	return info, nil
}

// onOpenStack checks the dmi collector's product.name for the
// "OpenStack" substring. Fails open when dmi wasn't run.
func onOpenStack(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil || info.Product == nil {
		return true
	}
	return strings.Contains(info.Product.Name, dmiProductSignature)
}

// fetchNovaDoc fetches Nova's enriched meta_data.json, which sits
// outside the EC2-mirror tree.
func (c *Collector) fetchNovaDoc(
	ctx context.Context,
) ([]byte, error) {
	return c.client.Get(ctx, "/openstack/latest/meta_data.json")
}

// transformEC2Paths populates Info's EC2-mirror fields from the
// per-path value map.
func transformEC2Paths(
	v map[string]string,
) *Info {
	info := &Info{
		AMIID:            v["/latest/meta-data/ami-id"],
		InstanceID:       v["/latest/meta-data/instance-id"],
		InstanceType:     v["/latest/meta-data/instance-type"],
		Hostname:         v["/latest/meta-data/hostname"],
		LocalHostname:    v["/latest/meta-data/local-hostname"],
		LocalIPv4:        v["/latest/meta-data/local-ipv4"],
		PublicHostname:   v["/latest/meta-data/public-hostname"],
		PublicIPv4:       v["/latest/meta-data/public-ipv4"],
		AvailabilityZone: v["/latest/meta-data/placement/availability-zone"],
		ReservationID:    v["/latest/meta-data/reservation-id"],
		KernelID:         v["/latest/meta-data/kernel-id"],
		RamdiskID:        v["/latest/meta-data/ramdisk-id"],
	}
	if sg := v["/latest/meta-data/security-groups"]; sg != "" {
		info.SecurityGroups = strings.Split(sg, "\n")
	}
	return info
}
