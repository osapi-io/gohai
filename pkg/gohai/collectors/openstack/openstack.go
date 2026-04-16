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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
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

// Provider names emitted in Info.Provider. Matches Ohai's
// openstack_provider return values.
const (
	providerOpenStack = "openstack"
	providerDreamhost = "dreamhost"
)

// passwdPath is the system passwd file used for the dreamhost-vs-openstack
// distinction. Package-level var so tests can swap it. Ohai's check is
// `Etc::Passwd.entries.map(&:name).include?("dhc-user")`.
var passwdPath = "/etc/passwd"

// dreamhostUser is the username Dreamhost shipped in their OpenStack
// images pre-2016. Matches Ohai's literal string.
const dreamhostUser = "dhc-user"

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
	Name        string            `json:"name,omitempty"`
	ProjectID   string            `json:"project_id,omitempty"`
	UUID        string            `json:"uuid,omitempty"`
	LaunchIndex int               `json:"launch_index,omitempty"`
	MetaData    map[string]string `json:"meta_data,omitempty"`
	PublicKeys  map[string]string `json:"public_keys,omitempty"`

	// Devices is the attached-block-device list from meta_data.json.
	// Each entry's shape varies by volume type (ephemeral/cinder
	// attached/boot); we surface the common fields.
	Devices []Device `json:"devices,omitempty"`

	// Provider distinguishes "openstack" from the legacy "dreamhost"
	// (pre-2016 Dreamhost OpenStack images shipped a `dhc-user`
	// account). Matches Ohai's openstack[:provider].
	Provider string `json:"provider,omitempty"`
}

// Device is one attached block-device entry from
// /openstack/latest/meta_data.json `devices`. OpenStack reports
// slightly different keys depending on the volume source; we capture
// the common ones for inventory / audit.
type Device struct {
	Type    string   `json:"type,omitempty"`    // "disk" / "cdrom" / ...
	Bus     string   `json:"bus,omitempty"`     // "virtio" / "scsi" / ...
	Serial  string   `json:"serial,omitempty"`  // block-device serial
	Path    string   `json:"path,omitempty"`    // device node path
	Address string   `json:"address,omitempty"` // PCI address
	Tags    []string `json:"tags,omitempty"`
}

// openStackMetaDoc mirrors the subset of /openstack/latest/meta_data.json
// we extract. `network_info` and `random_seed` remain intentionally
// skipped — the former is deployment-specific Neutron data, the latter
// is sensitive and has no inventory value.
type openStackMetaDoc struct {
	UUID             string            `json:"uuid"`
	Name             string            `json:"name"`
	ProjectID        string            `json:"project_id"`
	Hostname         string            `json:"hostname"`
	Meta             map[string]string `json:"meta"`
	PublicKeys       map[string]string `json:"public_keys"`
	AvailabilityZone string            `json:"availability_zone"`
	LaunchIndex      int               `json:"launch_index"`
	Devices          []rawDevice       `json:"devices"`
}

type rawDevice struct {
	Type    string   `json:"type"`
	Bus     string   `json:"bus"`
	Serial  string   `json:"serial"`
	Path    string   `json:"path"`
	Address string   `json:"address"`
	Tags    []string `json:"tags"`
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

// Collect gates the fetch on a DMI product_name match, walks the
// EC2-compatible /latest/meta-data tree recursively, and fetches the
// Nova-specific meta_data.json. The provider field is populated
// independently from /etc/passwd ("dreamhost" if dhc-user is present,
// "openstack" otherwise — matches Ohai). First-call failure returns
// (nil, nil); per-path failures are tolerated.
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onOpenStack(prior) {
		return nil, nil
	}

	tree, err := walk(ctx, c.client, "/latest/meta-data/")
	if err != nil {
		// Even if EC2-mirror walk fails, try the Nova doc — some
		// minimal Nova deployments only serve the JSON. Don't drop
		// detection just because /latest 404s.
		tree = nil
	}
	info := &Info{
		Provider: detectProvider(),
	}
	if tree != nil {
		populateFromTree(info, tree)
	}

	if body, err := c.fetchNovaDoc(ctx); err == nil {
		var doc openStackMetaDoc
		if jerr := json.Unmarshal(body, &doc); jerr == nil {
			info.UUID = doc.UUID
			info.Name = doc.Name
			info.ProjectID = doc.ProjectID
			info.LaunchIndex = doc.LaunchIndex
			info.MetaData = doc.Meta
			info.PublicKeys = doc.PublicKeys
			for _, d := range doc.Devices {
				info.Devices = append(info.Devices, Device(d))
			}
			if info.AvailabilityZone == "" {
				info.AvailabilityZone = doc.AvailabilityZone
			}
			if info.Hostname == "" {
				info.Hostname = doc.Hostname
			}
		}
	}

	if tree == nil && info.UUID == "" {
		// Nothing came back from either source — not actually OpenStack.
		return nil, nil
	}
	return info, nil
}

// detectProvider checks /etc/passwd for the "dhc-user" account that
// Dreamhost shipped in their pre-2016 OpenStack images. Matches
// Ohai's openstack_provider.
func detectProvider() string {
	f, err := os.Open(passwdPath)
	if err != nil {
		return providerOpenStack
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		// passwd entries: username:x:uid:gid:gecos:home:shell
		line := sc.Bytes()
		if i := bytes.IndexByte(line, ':'); i >= 0 {
			if string(line[:i]) == dreamhostUser {
				return providerDreamhost
			}
		}
	}
	return providerOpenStack
}

// walk fetches a directory listing at path and recurses into entries
// ending with "/". Leaf entries are fetched as values. Matches the
// recursive style Ohai's ec2_metadata mixin uses for OpenStack's
// EC2-compatible tree.
func walk(
	ctx context.Context,
	c *cloudmetadata.Client,
	path string,
) (map[string]any, error) {
	listing, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	for _, line := range strings.Split(string(listing), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key := sanitizeKey(line)
		child := path + line
		if strings.HasSuffix(line, "/") {
			sub, err := walk(ctx, c, child)
			if err != nil {
				continue
			}
			result[key] = sub
		} else {
			leaf, err := c.Get(ctx, child)
			if err != nil {
				continue
			}
			result[key] = strings.TrimSpace(string(leaf))
		}
	}
	return result, nil
}

// sanitizeKey mirrors Ohai's metadata_key — strip trailing slash,
// then dashes and slashes become underscores.
func sanitizeKey(
	k string,
) string {
	k = strings.TrimRight(k, "/")
	return strings.NewReplacer("-", "_", "/", "_").Replace(k)
}

// populateFromTree fills the typed Info fields from the walked tree.
// Unknown leaves are still preserved under Info.Raw.
func populateFromTree(
	info *Info,
	tree map[string]any,
) {
	info.AMIID = strVal(tree, "ami_id")
	info.InstanceID = strVal(tree, "instance_id")
	info.InstanceType = strVal(tree, "instance_type")
	info.Hostname = strVal(tree, "hostname")
	info.LocalHostname = strVal(tree, "local_hostname")
	info.LocalIPv4 = strVal(tree, "local_ipv4")
	info.PublicHostname = strVal(tree, "public_hostname")
	info.PublicIPv4 = strVal(tree, "public_ipv4")
	info.ReservationID = strVal(tree, "reservation_id")
	info.KernelID = strVal(tree, "kernel_id")
	info.RamdiskID = strVal(tree, "ramdisk_id")
	if placement, ok := tree["placement"].(map[string]any); ok {
		info.AvailabilityZone = strVal(placement, "availability_zone")
	}
	if sg := strVal(tree, "security_groups"); sg != "" {
		info.SecurityGroups = strings.Split(sg, "\n")
	}
}

// strVal returns the string value at key, or empty when absent or
// the wrong type.
func strVal(
	m map[string]any,
	key string,
) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
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
