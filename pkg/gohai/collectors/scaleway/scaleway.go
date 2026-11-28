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

// Package scaleway collects Scaleway instance metadata from the
// link-local metadata server at http://169.254.42.42/. The collector
// returns nil with no error when the endpoint is not reachable —
// that's the signal that the host isn't running on Scaleway.
package scaleway

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudScaleway, which re-exports this constant.
const ProviderName = "scaleway"

// metadataBaseURL is Scaleway's unusual link-local metadata address
// (not the standard 169.254.169.254 — matches Ohai's SCALEWAY_METADATA_ADDR).
const metadataBaseURL = "http://169.254.42.42"

// metadataPath returns the entire instance config as JSON in one GET.
const metadataPath = "/conf?format=json"

// metadataTimeout matches Ohai's 6s read timeout in mixin/scaleway_metadata.rb.
const metadataTimeout = 6 * time.Second

// cmdlineSignature is the substring Ohai looks for in /proc/cmdline
// to gate metadata fetch (has_scaleway_cmdline?).
const cmdlineSignature = "scaleway"

// procCmdlinePath is the kernel command-line file read for the
// cmdline detection signal, through the collector's FS.
const procCmdlinePath = "/proc/cmdline"

// Info is the Scaleway instance view. Flat shape, same pattern as
// every other gohai cloud collector.
type Info struct {
	ID              string   `json:"id"`
	Name            string   `json:"name,omitempty"`
	Hostname        string   `json:"hostname,omitempty"`
	AccountUID      string   `json:"account_uid,omitempty"`
	ProjectUID      string   `json:"project_uid,omitempty"`
	Type            string   `json:"type,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	StateDetail     string   `json:"state_detail,omitempty"`
	PublicIP        string   `json:"public_ip,omitempty"`
	PublicIPID      string   `json:"public_ip_id,omitempty"`
	PublicIPDynamic bool     `json:"public_ip_dynamic,omitempty"`
	PrivateIP       string   `json:"private_ip,omitempty"`
	IPv6Address     string   `json:"ipv6_address,omitempty"`
	IPv6Netmask     string   `json:"ipv6_netmask,omitempty"`
	IPv6Gateway     string   `json:"ipv6_gateway,omitempty"`
	Zone            string   `json:"zone,omitempty"`
	PlatformID      string   `json:"platform_id,omitempty"`
	SSHPublicKeys   []string `json:"ssh_public_keys,omitempty"`
	Volumes         []Volume `json:"volumes,omitempty"`

	Timezone   string      `json:"timezone,omitempty"`
	Bootscript *Bootscript `json:"bootscript,omitempty"`
}

// Bootscript is Scaleway's legacy boot configuration. Deprecated in
// favor of local boot — modern instances may return null. Typed for
// parity with Ohai which surfaces it when present.
type Bootscript struct {
	ID           string `json:"id"`
	Title        string `json:"title,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Kernel       string `json:"kernel,omitempty"`
	Initrd       string `json:"initrd,omitempty"`
	Bootcmdargs  string `json:"bootcmdargs,omitempty"`
	Organization string `json:"organization,omitempty"`
	Public       bool   `json:"public,omitempty"`
}

// Volume is one attached volume.
type Volume struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	VolumeType string `json:"volume_type,omitempty"`
	Size       int64  `json:"size,omitempty"`
	ExportURI  string `json:"export_uri,omitempty"`
}

// raw mirrors Scaleway's JSON shape for verbatim unmarshal. Separate
// from Info so we can reshape nested objects (public_ip, ipv6,
// location, ssh_public_keys) into flat fields.
type raw struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Hostname       string         `json:"hostname"`
	Organization   string         `json:"organization"`
	Project        string         `json:"project"`
	CommercialType string         `json:"commercial_type"`
	Tags           []string       `json:"tags"`
	StateDetail    string         `json:"state_detail"`
	PublicIP       *rawPublicIP   `json:"public_ip"`
	PrivateIP      string         `json:"private_ip"`
	IPv6           *rawIPv6       `json:"ipv6"`
	Location       *rawLocation   `json:"location"`
	SSHPublicKeys  []rawSSHKey    `json:"ssh_public_keys"`
	Volumes        rawVolumes     `json:"volumes"`
	Timezone       string         `json:"timezone"`
	Bootscript     *rawBootscript `json:"bootscript"`
}

type rawPublicIP struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Dynamic bool   `json:"dynamic"`
}

type rawIPv6 struct {
	Address string `json:"address"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
}

type rawLocation struct {
	ZoneID     string `json:"zone_id"`
	PlatformID string `json:"platform_id"`
}

type rawSSHKey struct {
	Key string `json:"key"`
}

type rawBootscript struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Architecture string `json:"architecture"`
	Kernel       string `json:"kernel"`
	Initrd       string `json:"initrd"`
	Bootcmdargs  string `json:"bootcmdargs"`
	Organization string `json:"organization"`
	Public       bool   `json:"public"`
}

// rawVolumes is Scaleway's oddly-shaped `{"0": {...}, "1": {...}}`
// index-keyed map. We flatten it to a slice ordered by key.
type rawVolumes map[string]rawVolume

type rawVolume struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	VolumeType string `json:"volume_type"`
	Size       int64  `json:"size"`
	ExportURI  string `json:"export_uri"`
}

// Collector fetches Scaleway's single-JSON metadata response.
type Collector struct {
	client *cloudmetadata.Client
	// FS is the filesystem that /proc/cmdline is read from.
	FS avfs.VFS
}

var _ collector.Collector = (*Collector)(nil)

// New returns a default Collector pointed at Scaleway's metadata
// server with Ohai-matching 6s timeout.
func New() *Collector {
	return NewWith(source.Sources{})
}

// NewWith returns a Collector that reads the metadata service through
// src.HTTP, so a fake host can answer it. A zero src.HTTP is the shared
// transport.
func NewWith(
	src source.Sources,
) *Collector {
	return collector.Wire(&Collector{
		client: cloudmetadata.New(
			metadataBaseURL,
			cloudmetadata.WithTimeout(metadataTimeout),
			cloudmetadata.WithTransport(src.HTTP),
		),
	}, src)
}

// NewWithClient returns a Collector backed by a caller-supplied client.
// Tests point it at an httptest.Server.
func NewWithClient(
	c *cloudmetadata.Client,
) *Collector {
	return collector.Wire(&Collector{client: c}, source.Sources{})
}

// Name returns "scaleway".
func (*Collector) Name() string { return "scaleway" }

// Category returns "cloud".
func (*Collector) Category() collector.Category { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies returns dmi: a Scaleway instance's DMI sys_vendor is
// Scaleway. The kernel cmdline signature of the old bootscript boot is a
// second signal.
func (*Collector) Dependencies() []string { return []string{"dmi"} }

// Collect checks DMI sys_vendor, then /proc/cmdline, for Scaleway, then
// fetches the single /conf?format=json document. Returns (nil, nil) when
// neither signal names Scaleway or the endpoint is unreachable, and the
// context's error when the collection was cancelled.
func (c *Collector) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onScaleway(c.FS, prior) {
		return nil, ctx.Err()
	}
	body, err := c.client.Get(ctx, metadataPath)
	if err != nil {
		// Any fetch failure (transport, 404, body-read) means we
		// can't determine whether we're on Scaleway — treat as "not
		// on this cloud" rather than propagating noise.
		return nil, ctx.Err()
	}
	var r raw
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return transform(r), nil
}

// onScaleway reports whether the host is a Scaleway instance: its DMI
// sys_vendor is Scaleway, as on instances that boot through UEFI or GRUB,
// or /proc/cmdline has Scaleway's signature, as Ohai's has_scaleway_cmdline?
// checks, from the old bootscript boot.
func onScaleway(
	fs avfs.VFS,
	prior collector.PriorResults,
) bool {
	if info, ok := collector.GetDep[*dmi.Info](prior, "dmi"); ok && info != nil &&
		info.Product != nil &&
		strings.Contains(strings.ToLower(info.Product.VendorName), cmdlineSignature) {
		return true
	}
	b, err := fs.ReadFile(procCmdlinePath)
	if err != nil {
		return false
	}
	return bytes.Contains(bytes.ToLower(b), []byte(cmdlineSignature))
}

// transform reshapes the raw response into the flat Info.
func transform(
	r raw,
) *Info {
	info := &Info{
		ID:          r.ID,
		Name:        r.Name,
		Hostname:    r.Hostname,
		AccountUID:  r.Organization,
		ProjectUID:  r.Project,
		Type:        r.CommercialType,
		Tags:        r.Tags,
		StateDetail: r.StateDetail,
		PrivateIP:   r.PrivateIP,
		Timezone:    r.Timezone,
	}
	if r.PublicIP != nil {
		info.PublicIP = r.PublicIP.Address
		info.PublicIPID = r.PublicIP.ID
		info.PublicIPDynamic = r.PublicIP.Dynamic
	}
	if r.IPv6 != nil {
		info.IPv6Address = r.IPv6.Address
		info.IPv6Netmask = r.IPv6.Netmask
		info.IPv6Gateway = r.IPv6.Gateway
	}
	if r.Location != nil {
		info.Zone = r.Location.ZoneID
		info.PlatformID = r.Location.PlatformID
	}
	for _, k := range r.SSHPublicKeys {
		k.Key = strings.TrimSpace(k.Key)
		if k.Key != "" {
			info.SSHPublicKeys = append(info.SSHPublicKeys, k.Key)
		}
	}
	// Scaleway keys the volumes "0", "1", ...: list them in that order, so
	// each run reports the same list.
	keys := make([]string, 0, len(r.Volumes))
	for k := range r.Volumes {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})
	for _, k := range keys {
		v := r.Volumes[k]
		info.Volumes = append(info.Volumes, Volume(v))
	}
	if r.Bootscript != nil {
		bs := Bootscript(*r.Bootscript)
		info.Bootscript = &bs
	}
	return info
}
