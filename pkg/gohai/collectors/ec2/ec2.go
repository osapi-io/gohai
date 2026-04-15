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

// Package ec2 collects AWS EC2 instance metadata from the link-local
// metadata server at http://169.254.169.254/. Uses IMDSv2 (token
// flow) by default and falls back to IMDSv1 when the token endpoint
// is unreachable. Returns nil with no error when the endpoint isn't
// reachable — that's the signal that the host isn't running on EC2.
package ec2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudAWS, which re-exports this constant.
const ProviderName = "aws"

// metadataBaseURL is EC2's link-local metadata endpoint.
const metadataBaseURL = "http://169.254.169.254"

// metadataTimeout matches Ohai's 10s read + keep-alive timeout in
// mixin/ec2_metadata.rb.
const metadataTimeout = 10 * time.Second

// DMI detection signatures. Matches Ohai's has_ec2_amazon_dmi? and
// has_ec2_xen_dmi? checks.
const (
	dmiBIOSVendorSignature  = "Amazon" // /sys/class/dmi/id/bios_vendor
	dmiBIOSVersionSignature = "amazon" // /sys/class/dmi/id/bios_version (lowercase)
)

// hypervisorUUIDPrefix is what Xen PV/HVM-on-Xen EC2 instances
// announce in /sys/hypervisor/uuid. Matches Ohai's has_ec2_xen_uuid?.
const hypervisorUUIDPrefix = "ec2"

// hypervisorUUIDPath is the sysfs file we read for xen detection.
// Package-level var so tests can swap it.
var hypervisorUUIDPath = "/sys/hypervisor/uuid"

// IMDSv2 token flow constants. Matches Ohai's mixin/ec2_metadata.rb.
const (
	tokenPath          = "/latest/api/token"
	tokenTTLHeader     = "X-aws-ec2-metadata-token-ttl-seconds"
	tokenTTLValue      = "60"
	tokenRequestHeader = "X-aws-ec2-metadata-token"
)

// supportedAPIVersions mirrors Ohai's EC2_SUPPORTED_VERSIONS array.
// When negotiating the version we intersect this list with EC2's own
// version listing and pick the latest match. Fall back to "latest"
// on negotiation failure.
var supportedAPIVersions = []string{
	"1.0", "2007-01-19", "2007-03-01", "2007-08-29", "2007-10-10",
	"2007-12-15", "2008-02-01", "2008-09-01", "2009-04-04", "2011-01-01",
	"2011-05-01", "2012-01-12", "2014-02-25", "2014-11-05", "2015-10-20",
	"2016-04-19", "2016-06-30", "2016-09-02", "2018-03-28", "2018-08-17",
	"2018-09-24", "2019-10-01", "2020-10-27", "2021-01-03", "2021-03-23",
	"2021-07-15",
}

// Paths we fetch under the negotiated /<version>/meta-data/ prefix.
// The first path (ami-id) doubles as the reachability check.
var metadataPaths = []string{
	"/meta-data/ami-id",
	"/meta-data/ami-launch-index",
	"/meta-data/ami-manifest-path",
	"/meta-data/hostname",
	"/meta-data/instance-action",
	"/meta-data/instance-id",
	"/meta-data/instance-type",
	"/meta-data/instance-life-cycle",
	"/meta-data/kernel-id",
	"/meta-data/local-hostname",
	"/meta-data/local-ipv4",
	"/meta-data/mac",
	"/meta-data/placement/availability-zone",
	"/meta-data/placement/availability-zone-id",
	"/meta-data/placement/group-name",
	"/meta-data/placement/host-id",
	"/meta-data/placement/partition-number",
	"/meta-data/placement/region",
	"/meta-data/public-hostname",
	"/meta-data/public-ipv4",
	"/meta-data/ramdisk-id",
	"/meta-data/reservation-id",
	"/meta-data/profile",
	"/meta-data/services/domain",
	"/meta-data/services/partition",
	"/meta-data/spot/instance-action",
	"/meta-data/spot/termination-time",
}

// blockDeviceMappingPath is the directory whose children (ami, root,
// ebs0, ebs1, ephemeral0, ...) each resolve to a device path.
const blockDeviceMappingPath = "/meta-data/block-device-mapping"

// productCodesPath is a newline-delimited list of marketplace product codes.
const productCodesPath = "/meta-data/product-codes"

// publicKeysPath is the directory whose children (0, 1, 2, ...) each
// hold an OpenSSH-format public key under `<N>/openssh-key`.
const publicKeysPath = "/meta-data/public-keys"

// securityGroupsPath is the newline-split leaf (Ohai's EC2_ARRAY_VALUES).
const securityGroupsPath = "/meta-data/security-groups"

// localIPv4sPath is another newline-split leaf (Ohai's EC2_ARRAY_VALUES).
const localIPv4sPath = "/meta-data/local-ipv4s"

// iamInfoPath holds the IAM instance profile ARN. Ohai drops
// security-credentials; we follow suit by only fetching /iam/info.
const iamInfoPath = "/meta-data/iam/info"

// identityDocPath is the dynamic identity document.
const identityDocPath = "/dynamic/instance-identity/document"

// userDataPath is the raw user-data blob.
const userDataPath = "/user-data/"

// macsPath is the root of the per-ENI subtree.
const macsPath = "/meta-data/network/interfaces/macs/"

// scrubKey is the one metadata key Ohai drops entirely because it
// carries credentials (matches the plugin's explicit skip).
const scrubKey = "identity_credentials_ec2_security_credentials_ec2_instance"

// Info is the EC2 view. Flat shape merging /meta-data/ fields with
// the richer dynamic identity document, IAM info, and per-ENI data.
type Info struct {
	// Identity.
	InstanceID        string `json:"instance_id"`
	InstanceType      string `json:"instance_type,omitempty"`
	InstanceLifecycle string `json:"instance_life_cycle,omitempty"`
	AMIID             string `json:"ami_id,omitempty"`
	AMILaunchIndex    string `json:"ami_launch_index,omitempty"`
	AMIManifestPath   string `json:"ami_manifest_path,omitempty"`

	// Naming.
	Hostname       string `json:"hostname,omitempty"`
	LocalHostname  string `json:"local_hostname,omitempty"`
	PublicHostname string `json:"public_hostname,omitempty"`

	// Network.
	LocalIPv4      string   `json:"local_ipv4,omitempty"`
	LocalIPv4s     []string `json:"local_ipv4s,omitempty"`
	PublicIPv4     string   `json:"public_ipv4,omitempty"`
	MAC            string   `json:"mac,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`

	// Per-ENI subtree under /meta-data/network/interfaces/macs/<mac>/.
	// Keyed by MAC address — matches Ohai's shape.
	NetworkInterfaces map[string]NetworkInterface `json:"network_interfaces,omitempty"`

	// Placement (from meta-data/placement + identity doc).
	Region           string `json:"region,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
	AccountID        string `json:"account_id,omitempty"`

	// Placement extras (from meta-data/placement/*). Populated only
	// when IMDS returns non-404 for the respective key — empty on
	// classic VPC instances without dedicated-host / partition-
	// placement features.
	AvailabilityZoneID string `json:"availability_zone_id,omitempty"`
	GroupName          string `json:"group_name,omitempty"`
	HostID             string `json:"host_id,omitempty"`
	PartitionNumber    string `json:"partition_number,omitempty"`

	// AMI / kernel metadata (legacy paravirt instances).
	KernelID  string `json:"kernel_id,omitempty"`
	RamdiskID string `json:"ramdisk_id,omitempty"`

	// Lifecycle + spot signals.
	InstanceAction      string `json:"instance_action,omitempty"`
	SpotInstanceAction  string `json:"spot_instance_action,omitempty"`
	SpotTerminationTime string `json:"spot_termination_time,omitempty"`

	// Services endpoint — useful on GovCloud / China regions.
	ServicesDomain    string `json:"services_domain,omitempty"`
	ServicesPartition string `json:"services_partition,omitempty"`

	// Marketplace product codes (empty on non-marketplace AMIs).
	ProductCodes []string `json:"product_codes,omitempty"`

	// SSH public keys attached at launch (OpenSSH format, one per
	// indexed child under public-keys/).
	PublicKeys []string `json:"public_keys,omitempty"`

	// BlockDeviceMapping maps the AMI's virtual disk names (`ami`,
	// `root`, `ebs0`, `ephemeral0`, ...) to the EC2 device path
	// (`/dev/sda1`, `/dev/xvdb`, ...). Matches Ohai's
	// `block_device_mapping_*` keys.
	BlockDeviceMapping map[string]string `json:"block_device_mapping,omitempty"`

	// Reservation / IAM.
	ReservationID string           `json:"reservation_id,omitempty"`
	Profile       string           `json:"profile,omitempty"`
	IAMInfo       *IAMInstanceInfo `json:"iam_info,omitempty"`

	// APIVersion is the version the collector negotiated with EC2's
	// IMDS. Useful for debugging and for consumers who want to know
	// the feature level of the data they received.
	APIVersion string `json:"api_version,omitempty"`

	// UserData is the raw user-data blob. Base64-encoded when binary,
	// plaintext otherwise — matches Ohai's handling.
	UserData string `json:"user_data,omitempty"`
}

// NetworkInterface is one ENI's metadata subtree. Each field maps
// directly to a path under
// /<version>/meta-data/network/interfaces/macs/<mac>/<field>.
type NetworkInterface struct {
	DeviceNumber         string   `json:"device_number,omitempty"`
	InterfaceID          string   `json:"interface_id,omitempty"`
	LocalHostname        string   `json:"local_hostname,omitempty"`
	LocalIPv4s           []string `json:"local_ipv4s,omitempty"`
	MAC                  string   `json:"mac,omitempty"`
	NetworkCardIndex     string   `json:"network_card_index,omitempty"`
	OwnerID              string   `json:"owner_id,omitempty"`
	PublicHostname       string   `json:"public_hostname,omitempty"`
	PublicIPv4s          []string `json:"public_ipv4s,omitempty"`
	SecurityGroupIDs     []string `json:"security_group_ids,omitempty"`
	SecurityGroups       []string `json:"security_groups,omitempty"`
	SubnetID             string   `json:"subnet_id,omitempty"`
	SubnetIPv4CIDRBlock  string   `json:"subnet_ipv4_cidr_block,omitempty"`
	SubnetIPv6CIDRBlocks []string `json:"subnet_ipv6_cidr_blocks,omitempty"`
	VPCID                string   `json:"vpc_id,omitempty"`
	VPCIPv4CIDRBlock     string   `json:"vpc_ipv4_cidr_block,omitempty"`
	VPCIPv4CIDRBlocks    []string `json:"vpc_ipv4_cidr_blocks,omitempty"`
	VPCIPv6CIDRBlocks    []string `json:"vpc_ipv6_cidr_blocks,omitempty"`
	IPv6s                []string `json:"ipv6s,omitempty"`
}

// IAMInstanceInfo is the iam/info sub-document. Excludes
// security-credentials (secrets) — matches Ohai's explicit drop.
type IAMInstanceInfo struct {
	Code               string `json:"code,omitempty"`
	LastUpdated        string `json:"last_updated,omitempty"`
	InstanceProfileArn string `json:"instance_profile_arn,omitempty"`
	InstanceProfileID  string `json:"instance_profile_id,omitempty"`
}

// identityDoc is the JSON shape of the dynamic identity document.
type identityDoc struct {
	AccountID        string `json:"accountId"`
	Region           string `json:"region"`
	AvailabilityZone string `json:"availabilityZone"`
	InstanceID       string `json:"instanceId"`
}

// rawIAMInfo mirrors iam/info's JSON shape.
type rawIAMInfo struct {
	Code               string `json:"Code"`
	LastUpdated        string `json:"LastUpdated"`
	InstanceProfileArn string `json:"InstanceProfileArn"`
	InstanceProfileID  string `json:"InstanceProfileId"`
}

// Collector fetches EC2 metadata via IMDSv2 (with IMDSv1 fallback).
type Collector struct {
	client *cloudmetadata.Client
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at EC2's metadata server
// with Ohai-matching 10s timeout.
func New() *Collector {
	return NewWithClient(cloudmetadata.New(
		metadataBaseURL,
		cloudmetadata.WithTimeout(metadataTimeout),
	))
}

// NewWithClient returns a Collector backed by a caller-supplied client.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return &Collector{client: c}
}

// Name returns "ec2".
func (*Collector) Name() string { return "ec2" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies declares dmi — EC2 writes "Amazon" as bios_vendor
// and/or "amazon" in bios_version. Matches Ohai's DMI-based gates.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect runs the EC2 IMDS sequence matching Ohai's plugin:
//  1. DMI + hypervisor UUID gates (fail open when dmi absent)
//  2. IMDSv2 token PUT (fall back to IMDSv1 on 404)
//  3. API version negotiation (pin latest supported; fall back to "latest")
//  4. Meta-data tree walk (curated paths + security-groups +
//     local_ipv4s + per-ENI recursive subtree)
//  5. IAM info (credentials scrubbed)
//  6. Dynamic identity document
//  7. User-data (base64 when binary)
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onEC2(prior) {
		return nil, nil
	}

	token := c.fetchIMDSv2Token(ctx)
	headers := imdsHeaders(token)
	version := c.negotiateAPIVersion(ctx, headers)
	versionedGet := func(path string) ([]byte, error) {
		return c.client.GetWithHeaders(ctx, "/"+version+path, headers)
	}

	values := make(map[string]string, len(metadataPaths))
	for i, p := range metadataPaths {
		body, err := versionedGet(p)
		if err != nil {
			if i == 0 {
				return nil, nil
			}
			continue
		}
		values[p] = strings.TrimSpace(string(body))
	}

	info := transformMetadata(values)
	info.APIVersion = version

	// Newline-split array leaves (Ohai's EC2_ARRAY_VALUES).
	if body, err := versionedGet(securityGroupsPath); err == nil {
		info.SecurityGroups = splitLines(string(body))
	}
	if body, err := versionedGet(localIPv4sPath); err == nil {
		info.LocalIPv4s = splitLines(string(body))
	}
	if body, err := versionedGet(productCodesPath); err == nil {
		info.ProductCodes = splitLines(string(body))
	}

	// Variable-keyed subtrees — fetch the listing, then each entry.
	info.BlockDeviceMapping = fetchBlockDeviceMapping(versionedGet)
	info.PublicKeys = fetchPublicKeys(versionedGet)

	// Per-ENI subtree. Tolerate missing on single-interface instances
	// (older shapes sometimes 404 this tree).
	if enis := fetchENIs(versionedGet); len(enis) > 0 {
		info.NetworkInterfaces = enis
	}

	// IAM info — keep profile ARN, drop security-credentials.
	if body, err := versionedGet(iamInfoPath); err == nil {
		var iam rawIAMInfo
		if json.Unmarshal(body, &iam) == nil {
			info.IAMInfo = &IAMInstanceInfo{
				Code:               iam.Code,
				LastUpdated:        iam.LastUpdated,
				InstanceProfileArn: iam.InstanceProfileArn,
				InstanceProfileID:  iam.InstanceProfileID,
			}
		}
	}

	// Identity document: canonical region / AZ / account.
	if body, err := versionedGet(identityDocPath); err == nil {
		var doc identityDoc
		if json.Unmarshal(body, &doc) == nil {
			if info.AccountID == "" {
				info.AccountID = doc.AccountID
			}
			if info.Region == "" {
				info.Region = doc.Region
			}
			if info.AvailabilityZone == "" {
				info.AvailabilityZone = doc.AvailabilityZone
			}
			if info.InstanceID == "" {
				info.InstanceID = doc.InstanceID
			}
		}
	}

	// User-data: plaintext when UTF-8, base64 when binary (matches
	// Ohai's Encoding::BINARY check). 404 (no user-data set) is
	// expected and tolerated.
	if body, err := versionedGet(userDataPath); err == nil {
		info.UserData = encodeUserData(body)
	}

	return info, nil
}

// fetchBlockDeviceMapping walks /meta-data/block-device-mapping/.
// The directory listing is newline-separated child names (ami, root,
// ebs0, ephemeral0, ...); each child resolves to a device path.
func fetchBlockDeviceMapping(
	get func(string) ([]byte, error),
) map[string]string {
	listing, err := get(blockDeviceMappingPath)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, name := range splitLines(string(listing)) {
		body, err := get(blockDeviceMappingPath + "/" + name)
		if err != nil {
			continue
		}
		out[name] = strings.TrimSpace(string(body))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// fetchPublicKeys walks /meta-data/public-keys/. Listing is
// "<index>=<key-name>"; each index has an `openssh-key` leaf.
func fetchPublicKeys(
	get func(string) ([]byte, error),
) []string {
	listing, err := get(publicKeysPath)
	if err != nil {
		return nil
	}
	var keys []string
	for _, line := range splitLines(string(listing)) {
		idx, _, ok := strings.Cut(line, "=")
		if !ok {
			idx = line
		}
		body, err := get(publicKeysPath + "/" + idx + "/openssh-key")
		if err != nil {
			continue
		}
		key := strings.TrimSpace(string(body))
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// onEC2 runs Ohai's has_ec2_* chain (non-Windows):
//   - dmi.BIOS.Vendor contains "Amazon"  (has_ec2_amazon_dmi?)
//   - dmi.BIOS.Version contains "amazon" (has_ec2_xen_dmi?)
//   - /sys/hypervisor/uuid starts with "ec2" (has_ec2_xen_uuid?)
//
// Any match → detected. Falls open when dmi isn't in prior — the
// HTTP probe will still error-out on non-EC2 hosts.
func onEC2(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil {
		return true
	}
	if info.BIOS != nil {
		if strings.Contains(info.BIOS.Vendor, dmiBIOSVendorSignature) {
			return true
		}
		if strings.Contains(strings.ToLower(info.BIOS.Version), dmiBIOSVersionSignature) {
			return true
		}
	}
	if b, err := os.ReadFile(hypervisorUUIDPath); err == nil {
		if strings.HasPrefix(strings.TrimSpace(string(b)), hypervisorUUIDPrefix) {
			return true
		}
	}
	return false
}

// fetchIMDSv2Token requests an IMDSv2 token with a 60s TTL. Empty
// string on failure → IMDSv1 fallback path.
func (c *Collector) fetchIMDSv2Token(
	ctx context.Context,
) string {
	body, err := c.client.Put(ctx, tokenPath, map[string]string{
		tokenTTLHeader: tokenTTLValue,
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// imdsHeaders builds the per-request header set for token-authed reads.
func imdsHeaders(
	token string,
) map[string]string {
	if token == "" {
		return nil
	}
	return map[string]string{tokenRequestHeader: token}
}

// negotiateAPIVersion asks EC2's IMDS which versions it supports and
// picks the latest one gohai knows. On failure (404, empty list, no
// intersection) falls back to "latest" — which EC2 aliases to its
// newest supported version. Matches Ohai's best_api_version.
func (c *Collector) negotiateAPIVersion(
	ctx context.Context,
	headers map[string]string,
) string {
	body, err := c.client.GetWithHeaders(ctx, "/", headers)
	if err != nil {
		return "latest"
	}
	listed := splitLines(string(body))
	if len(listed) == 0 {
		return "latest"
	}
	supported := make(map[string]struct{}, len(supportedAPIVersions))
	for _, v := range supportedAPIVersions {
		supported[v] = struct{}{}
	}
	matches := make([]string, 0, len(listed))
	for _, v := range listed {
		if _, ok := supported[v]; ok {
			matches = append(matches, v)
		}
	}
	if len(matches) == 0 {
		return "latest"
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	return matches[0]
}

// transformMetadata populates Info from the per-path value map.
func transformMetadata(
	v map[string]string,
) *Info {
	info := &Info{
		InstanceID:          v["/meta-data/instance-id"],
		InstanceType:        v["/meta-data/instance-type"],
		InstanceLifecycle:   v["/meta-data/instance-life-cycle"],
		AMIID:               v["/meta-data/ami-id"],
		AMILaunchIndex:      v["/meta-data/ami-launch-index"],
		AMIManifestPath:     v["/meta-data/ami-manifest-path"],
		Hostname:            v["/meta-data/hostname"],
		LocalHostname:       v["/meta-data/local-hostname"],
		PublicHostname:      v["/meta-data/public-hostname"],
		LocalIPv4:           v["/meta-data/local-ipv4"],
		PublicIPv4:          v["/meta-data/public-ipv4"],
		MAC:                 v["/meta-data/mac"],
		Region:              v["/meta-data/placement/region"],
		AvailabilityZone:    v["/meta-data/placement/availability-zone"],
		AvailabilityZoneID:  v["/meta-data/placement/availability-zone-id"],
		GroupName:           v["/meta-data/placement/group-name"],
		HostID:              v["/meta-data/placement/host-id"],
		PartitionNumber:     v["/meta-data/placement/partition-number"],
		ReservationID:       v["/meta-data/reservation-id"],
		Profile:             v["/meta-data/profile"],
		KernelID:            v["/meta-data/kernel-id"],
		RamdiskID:           v["/meta-data/ramdisk-id"],
		InstanceAction:      v["/meta-data/instance-action"],
		SpotInstanceAction:  v["/meta-data/spot/instance-action"],
		SpotTerminationTime: v["/meta-data/spot/termination-time"],
		ServicesDomain:      v["/meta-data/services/domain"],
		ServicesPartition:   v["/meta-data/services/partition"],
	}
	return info
}

// splitLines splits the body on newlines and drops empty trimmed
// lines. Also drops the Ohai security-scrub key. Matches Ohai's
// .split("\n") behavior with the explicit skip.
func splitLines(
	s string,
) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" || line == scrubKey {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// encodeUserData returns the raw user-data as plaintext when valid
// UTF-8, or base64-encoded otherwise — matches Ohai's binary check.
// Empty input returns empty string.
func encodeUserData(
	body []byte,
) string {
	if len(body) == 0 {
		return ""
	}
	if utf8.Valid(body) {
		return string(body)
	}
	return base64.StdEncoding.EncodeToString(body)
}

// fetchENIs walks /meta-data/network/interfaces/macs/, recursing
// into each MAC subdirectory to build a typed NetworkInterface.
// Returns the map keyed by MAC (trailing slash stripped). Empty when
// the tree is absent or empty — tolerated for older instance shapes.
func fetchENIs(
	get func(path string) ([]byte, error),
) map[string]NetworkInterface {
	body, err := get(macsPath)
	if err != nil {
		return nil
	}
	result := make(map[string]NetworkInterface)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each entry is "<mac>/". Strip trailing slash for the key.
		mac := strings.TrimRight(line, "/")
		iface := fetchENI(get, mac)
		iface.MAC = mac
		result[mac] = iface
	}
	return result
}

// fetchENI walks the subdirectory for one MAC and maps each known
// leaf into the typed NetworkInterface struct. Unknown leaves are
// silently ignored (we don't expose a Raw map on individual ENIs —
// new fields are a typed-struct update).
func fetchENI(
	get func(path string) ([]byte, error),
	mac string,
) NetworkInterface {
	iface := NetworkInterface{}
	prefix := macsPath + mac + "/"
	fetch := func(name string) string {
		body, err := get(prefix + name)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(body))
	}
	fetchList := func(name string) []string {
		body, err := get(prefix + name)
		if err != nil {
			return nil
		}
		return splitLines(string(body))
	}
	iface.DeviceNumber = fetch("device-number")
	iface.InterfaceID = fetch("interface-id")
	iface.LocalHostname = fetch("local-hostname")
	iface.LocalIPv4s = fetchList("local-ipv4s")
	iface.NetworkCardIndex = fetch("network-card-index")
	iface.OwnerID = fetch("owner-id")
	iface.PublicHostname = fetch("public-hostname")
	iface.PublicIPv4s = fetchList("public-ipv4s")
	iface.SecurityGroupIDs = fetchList("security-group-ids")
	iface.SecurityGroups = fetchList("security-groups")
	iface.SubnetID = fetch("subnet-id")
	iface.SubnetIPv4CIDRBlock = fetch("subnet-ipv4-cidr-block")
	iface.SubnetIPv6CIDRBlocks = fetchList("subnet-ipv6-cidr-blocks")
	iface.VPCID = fetch("vpc-id")
	iface.VPCIPv4CIDRBlock = fetch("vpc-ipv4-cidr-block")
	iface.VPCIPv4CIDRBlocks = fetchList("vpc-ipv4-cidr-blocks")
	iface.VPCIPv6CIDRBlocks = fetchList("vpc-ipv6-cidr-blocks")
	iface.IPv6s = fetchList("ipv6s")
	return iface
}
