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

// Package digitalocean collects DigitalOcean droplet metadata from
// the link-local metadata server at http://169.254.169.254/metadata/v1.json.
// The collector returns nil with no error when the endpoint is not
// reachable. That is the signal that the host is not a droplet.
package digitalocean

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudDigitalOcean, which re-exports this constant.
const ProviderName = "digital_ocean"

// metadataBaseURL is DigitalOcean's link-local metadata endpoint.
const metadataBaseURL = "http://169.254.169.254"

// metadataPath returns the entire droplet metadata as one JSON blob.
const metadataPath = "/metadata/v1.json"

// metadataTimeout matches Ohai's 6s read timeout in mixin/do_metadata.rb.
const metadataTimeout = 6 * time.Second

// dmiVendorSignature is the exact string DigitalOcean writes to
// /sys/class/dmi/id/bios_vendor. Matches Ohai's has_do_dmi? check.
const dmiVendorSignature = "DigitalOcean"

// Info is the DigitalOcean view: each field DO exposes via
// /metadata/v1.json except `vendor_data`, which is dropped because
// it commonly contains cloud-init user scripts with credentials
// (matches Ohai's explicit drop).
type Info struct {
	ID         int64    `json:"id"`
	Hostname   string   `json:"hostname,omitempty"`
	Region     string   `json:"region,omitempty"`
	PublicKeys []string `json:"public_keys,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Features   []string `json:"features,omitempty"`
	FloatingIP string   `json:"floating_ip,omitempty"`
	ReservedIP string   `json:"reserved_ip,omitempty"` // DO's newer replacement for floating_ip
	IPv4NS     []string `json:"ipv4_nameservers,omitempty"`

	Interfaces []Interface `json:"interfaces,omitempty"`
}

// Interface is one attached network interface. Scope is "public" or
// "private", in DO's two-interface model (public NAT'd + private VLAN).
type Interface struct {
	Scope    string `json:"scope"`
	MAC      string `json:"mac"`
	Type     string `json:"type,omitempty"`
	IPv4     string `json:"ipv4,omitempty"`
	IPv4Mask string `json:"ipv4_netmask,omitempty"`
	IPv4GW   string `json:"ipv4_gateway,omitempty"`
	IPv6     string `json:"ipv6,omitempty"`
	IPv6Mask int    `json:"ipv6_cidr,omitempty"`
	IPv6GW   string `json:"ipv6_gateway,omitempty"`
	Anchor   string `json:"anchor_ipv4,omitempty"`
}

// raw mirrors DO's JSON shape; we reshape into Info for flat access.
type raw struct {
	DropletID  int64    `json:"droplet_id"`
	Hostname   string   `json:"hostname"`
	Region     string   `json:"region"`
	PublicKeys []string `json:"public_keys"`
	Tags       []string `json:"tags"`
	Features   []string `json:"features"`
	FloatingIP *struct {
		IPv4 *struct {
			IPAddress string `json:"ip_address"`
		} `json:"ipv4"`
	} `json:"floating_ip"`
	ReservedIP *struct {
		IPv4 *struct {
			IPAddress string `json:"ip_address"`
		} `json:"ipv4"`
	} `json:"reserved_ip"`
	DNS *struct {
		Nameservers []string `json:"nameservers"`
	} `json:"dns"`
	Interfaces map[string][]rawIface `json:"interfaces"`
}

type rawIface struct {
	MAC        string        `json:"mac"`
	Type       string        `json:"type"`
	IPv4       *rawIfaceIPv4 `json:"ipv4"`
	IPv6       *rawIfaceIPv6 `json:"ipv6"`
	AnchorIPv4 *rawIfaceIPv4 `json:"anchor_ipv4"`
}

type rawIfaceIPv4 struct {
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
}

type rawIfaceIPv6 struct {
	IPAddress string `json:"ip_address"`
	CIDR      int    `json:"cidr"`
	Gateway   string `json:"gateway"`
}

// Metadata fetches DO's single-JSON metadata response.
type Metadata struct {
	// HTTP is the transport of the metadata reads. A nil HTTP is the
	// shared transport.
	HTTP http.RoundTripper
}

// Collector is the public interface of the digitalocean collector.
type Collector interface {
	collector.Collector
}

var _ Collector = (*Metadata)(nil)

// New returns the digitalocean collector on the real host.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the digitalocean collector reading the host through src. It reads
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
		cloudmetadata.WithTimeout(metadataTimeout),
		cloudmetadata.WithTransport(c.HTTP),
	)
}

// Name returns "digital_ocean".
func (*Metadata) Name() string { return "digital_ocean" }

// Category returns "cloud".
func (*Metadata) Category() collector.Category { return collector.CategoryCloud }

// DefaultEnabled returns false, because cloud collectors are opt-in.
func (*Metadata) DefaultEnabled() bool { return false }

// Dependencies declares dmi for the bios_vendor pre-check. Matches
// Ohai's has_do_dmi? gate.
func (*Metadata) Dependencies() []string { return []string{"dmi"} }

// Collect gates the metadata fetch on a DMI bios_vendor match for
// "DigitalOcean". Returns (nil, nil) when the signature is missing
// or the endpoint is unreachable.
func (c *Metadata) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onDigitalOcean(prior) {
		return nil, ctx.Err()
	}
	var r raw
	if err := c.client().GetJSON(ctx, metadataPath, &r); err != nil {
		if errors.Is(err, cloudmetadata.ErrNotAvailable) {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("read digitalocean metadata: %w", err)
	}
	return transform(r), nil
}

// onDigitalOcean checks the dmi collector's bios.vendor field for
// "DigitalOcean". Fails open (tries the HTTP call) when dmi wasn't
// run, because the endpoint probe itself is a valid detection fallback.
func onDigitalOcean(
	prior collector.PriorResults,
) bool {
	info, ok := collector.GetDep[*dmi.Info](prior, "dmi")
	// When dmi did not run, as when a caller asked for this collector
	// alone, only the metadata service can tell. A host whose DMI has no
	// BIOS table, such as a container or a host with no
	// /sys/class/dmi, is not this cloud, so it is not probed.
	if !ok {
		return true
	}
	if info == nil || info.BIOS == nil {
		return false
	}
	return info.BIOS.Manufacturer == dmiVendorSignature
}

// transform reshapes the raw response. Flattens the nested
// floating_ip, dns, and interfaces objects into flat Info fields,
// preserving all upstream data except vendor_data (dropped for
// security; matches Ohai's explicit removal).
func transform(
	r raw,
) *Info {
	info := &Info{
		ID:         r.DropletID,
		Hostname:   r.Hostname,
		Region:     r.Region,
		PublicKeys: r.PublicKeys,
		Tags:       r.Tags,
		Features:   r.Features,
	}
	if r.FloatingIP != nil && r.FloatingIP.IPv4 != nil {
		info.FloatingIP = r.FloatingIP.IPv4.IPAddress
	}
	if r.ReservedIP != nil && r.ReservedIP.IPv4 != nil {
		info.ReservedIP = r.ReservedIP.IPv4.IPAddress
	}
	if r.DNS != nil {
		info.IPv4NS = r.DNS.Nameservers
	}
	// Emit interfaces in deterministic order: public first (matches
	// Ohai's convention), then private.
	for _, scope := range []string{"public", "private"} {
		for _, ri := range r.Interfaces[scope] {
			iface := Interface{
				Scope: scope,
				MAC:   ri.MAC,
				Type:  ri.Type,
			}
			if ri.IPv4 != nil {
				iface.IPv4 = ri.IPv4.IPAddress
				iface.IPv4Mask = ri.IPv4.Netmask
				iface.IPv4GW = ri.IPv4.Gateway
			}
			if ri.IPv6 != nil {
				iface.IPv6 = ri.IPv6.IPAddress
				iface.IPv6Mask = ri.IPv6.CIDR
				iface.IPv6GW = ri.IPv6.Gateway
			}
			if ri.AnchorIPv4 != nil {
				iface.Anchor = ri.AnchorIPv4.IPAddress
			}
			info.Interfaces = append(info.Interfaces, iface)
		}
	}
	return info
}
