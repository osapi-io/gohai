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
	"encoding/json"
	"strings"

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

// dmiBIOSSignature is the bios_vendor substring EC2 writes. Matches
// Ohai's has_ec2_amazon_dmi? check.
const dmiBIOSSignature = "Amazon"

// IMDSv2 token flow constants. Matches Ohai's mixin/ec2_metadata.rb.
const (
	tokenPath          = "/latest/api/token"
	tokenTTLHeader     = "X-aws-ec2-metadata-token-ttl-seconds"
	tokenTTLValue      = "60"
	tokenRequestHeader = "X-aws-ec2-metadata-token"
)

// Metadata paths we fetch in order. The first probe (ami-id) doubles
// as the reachability check — its failure shorts the whole collector.
var metadataPaths = []string{
	"/latest/meta-data/ami-id",
	"/latest/meta-data/ami-launch-index",
	"/latest/meta-data/ami-manifest-path",
	"/latest/meta-data/hostname",
	"/latest/meta-data/instance-id",
	"/latest/meta-data/instance-type",
	"/latest/meta-data/instance-life-cycle",
	"/latest/meta-data/local-hostname",
	"/latest/meta-data/local-ipv4",
	"/latest/meta-data/mac",
	"/latest/meta-data/placement/availability-zone",
	"/latest/meta-data/placement/region",
	"/latest/meta-data/public-hostname",
	"/latest/meta-data/public-ipv4",
	"/latest/meta-data/reservation-id",
	"/latest/meta-data/security-groups",
	"/latest/meta-data/profile",
}

// iamInfoPath holds the IAM instance profile ARN and last-updated
// timestamp. Ohai fetches this under iam/info and strips the
// security-credentials leaf because it contains secrets.
const iamInfoPath = "/latest/meta-data/iam/info"

// identityDocPath is the dynamic identity document — JSON blob with
// account / region / AZ fields Ohai lifts to the top level.
const identityDocPath = "/latest/dynamic/instance-identity/document"

// Info is the EC2 view. Flat shape merging meta-data tree values
// with the richer instance-identity document.
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
	PublicIPv4     string   `json:"public_ipv4,omitempty"`
	MAC            string   `json:"mac,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`

	// Placement (from meta-data/placement + identity doc).
	Region           string `json:"region,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
	AccountID        string `json:"account_id,omitempty"`

	// Reservation / IAM.
	ReservationID string           `json:"reservation_id,omitempty"`
	Profile       string           `json:"profile,omitempty"`
	IAMInfo       *IAMInstanceInfo `json:"iam_info,omitempty"`
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

// New returns a default Collector pointed at EC2's metadata server.
func New() *Collector {
	return NewWithClient(cloudmetadata.New(metadataBaseURL))
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

// Dependencies declares dmi — EC2 writes "Amazon EC2" / "Amazon" as
// bios_vendor. Matches Ohai's has_ec2_amazon_dmi? check.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect gates the fetch on a DMI bios_vendor match, fetches an
// IMDSv2 token, then walks the meta-data + identity-document paths.
// Token fetch failure falls back to IMDSv1 (no token header). Any
// path failure after the first is tolerated. First-probe failure
// returns (nil, nil).
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onEC2(prior) {
		return nil, nil
	}

	token := c.fetchIMDSv2Token(ctx)
	headers := imdsHeaders(token)

	values := make(map[string]string, len(metadataPaths))
	for i, p := range metadataPaths {
		body, err := c.client.GetWithHeaders(ctx, p, headers)
		if err != nil {
			if i == 0 {
				return nil, nil
			}
			continue
		}
		values[p] = strings.TrimSpace(string(body))
	}

	info := transformMetadata(values)

	// iam/info is JSON; silently skip on any failure — instances
	// without an attached IAM profile legitimately 404.
	if body, err := c.client.GetWithHeaders(ctx, iamInfoPath, headers); err == nil {
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

	// identity-document gives us accountId + canonical region/AZ
	// fields. Ohai pulls these out as top-level ec2[:account_id],
	// ec2[:region], ec2[:availability_zone].
	if body, err := c.client.GetWithHeaders(ctx, identityDocPath, headers); err == nil {
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
	return info, nil
}

// fetchIMDSv2Token requests an IMDSv2 token with a 60s TTL. Returns
// empty string on failure — callers then proceed without the token
// header (IMDSv1 fallback), matching Ohai's behavior when the token
// PUT returns 404.
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

// imdsHeaders builds the per-request header set. Empty when no token
// was obtained — falls back to unauthenticated IMDSv1 requests.
func imdsHeaders(
	token string,
) map[string]string {
	if token == "" {
		return nil
	}
	return map[string]string{tokenRequestHeader: token}
}

// onEC2 checks the dmi collector's bios.vendor for the "Amazon"
// substring. Fails open when dmi wasn't run — endpoint probe will
// still detect or rule out.
func onEC2(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	if !ok || info == nil || info.BIOS == nil {
		return true
	}
	return strings.Contains(info.BIOS.Vendor, dmiBIOSSignature)
}

// transformMetadata populates Info from the per-path value map.
func transformMetadata(
	v map[string]string,
) *Info {
	info := &Info{
		InstanceID:        v["/latest/meta-data/instance-id"],
		InstanceType:      v["/latest/meta-data/instance-type"],
		InstanceLifecycle: v["/latest/meta-data/instance-life-cycle"],
		AMIID:             v["/latest/meta-data/ami-id"],
		AMILaunchIndex:    v["/latest/meta-data/ami-launch-index"],
		AMIManifestPath:   v["/latest/meta-data/ami-manifest-path"],
		Hostname:          v["/latest/meta-data/hostname"],
		LocalHostname:     v["/latest/meta-data/local-hostname"],
		PublicHostname:    v["/latest/meta-data/public-hostname"],
		LocalIPv4:         v["/latest/meta-data/local-ipv4"],
		PublicIPv4:        v["/latest/meta-data/public-ipv4"],
		MAC:               v["/latest/meta-data/mac"],
		Region:            v["/latest/meta-data/placement/region"],
		AvailabilityZone:  v["/latest/meta-data/placement/availability-zone"],
		ReservationID:     v["/latest/meta-data/reservation-id"],
		Profile:           v["/latest/meta-data/profile"],
	}
	if sg := v["/latest/meta-data/security-groups"]; sg != "" {
		info.SecurityGroups = strings.Split(sg, "\n")
	}
	return info
}
