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

// Package linode detects Linode hosts and reports their public /
// private IPs. Linode does not expose a metadata endpoint that Ohai
// uses — detection is heuristic (apt sources list), and the output
// comes entirely from the host's own network interfaces. Returns
// nil with no error when no Linode signature is found.
package linode

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudLinode, which re-exports this constant.
const ProviderName = "linode"

// aptSourcesPath is the file read for the has_linode_apt_repos? signal.
// Linode's official images ship an apt source referencing linode.com;
// custom-built images often do too. Package-level var for tests.
var aptSourcesPath = "/etc/apt/sources.list"

// linodeSignature is the substring we look for in apt sources.
const linodeSignature = "linode"

// interfaceAddrs is the "given an interface name, return its
// addresses" seam. Tests swap it to return canned addresses without
// touching the host's NICs.
var interfaceAddrs = defaultInterfaceAddrs

func defaultInterfaceAddrs(
	name string,
) ([]net.Addr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	return iface.Addrs()
}

// Info is the Linode view. Mirrors Ohai's node['linode'] which only
// reports two facts: public_ip and private_ip. No metadata service
// call — these come from the host's own eth0 / eth0:1 aliases.
type Info struct {
	PublicIP  string `json:"public_ip,omitempty"`
	PrivateIP string `json:"private_ip,omitempty"`
}

// Collector detects Linode hosts and reads interface-derived IPs.
type Collector struct{}

var _ collector.Collector = (*Collector)(nil)

// New returns a new Collector.
func New() *Collector { return &Collector{} }

// Name returns "linode".
func (*Collector) Name() string { return "linode" }

// Category returns "cloud".
func (*Collector) Category() string { return collector.CategoryCloud }

// DefaultEnabled returns false — cloud collectors are opt-in.
func (*Collector) DefaultEnabled() bool { return false }

// Dependencies declares hostname so the FQDN/Domain check can run
// against the host's resolved name (matches Ohai's has_linode_domain?
// signal which reads node['domain']).
func (*Collector) Dependencies() []string { return []string{"hostname"} }

// Collect returns the Linode Info when any detection signal fires
// (apt-sources OR domain contains "linode"), else (nil, nil).
// Mirrors Ohai's looks_like_linode? OR chain.
func (c *Collector) Collect(
	_ context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onLinode(prior) {
		return nil, nil
	}
	info := &Info{}
	info.PublicIP = firstIPv4("eth0")
	info.PrivateIP = firstIPv4("eth0:1")
	return info, nil
}

// onLinode runs Ohai's full non-hint detection chain:
//   - /etc/apt/sources.list contains "linode" (has_linode_apt_repos?)
//   - The host's FQDN or Domain contains "linode" (has_linode_domain?,
//     e.g. members.linode.com)
func onLinode(
	prior collector.PriorResults,
) bool {
	if b, err := os.ReadFile(aptSourcesPath); err == nil {
		if bytes.Contains(bytes.ToLower(b), []byte(linodeSignature)) {
			return true
		}
	}
	if h, ok := collector.GetDep[*hostname.Info](prior, "hostname"); ok && h != nil {
		if strings.Contains(strings.ToLower(h.FQDN), linodeSignature) ||
			strings.Contains(strings.ToLower(h.Domain), linodeSignature) {
			return true
		}
	}
	return false
}

// firstIPv4 returns the first non-link-local IPv4 address on the
// named interface. Empty when the interface is missing or has no
// qualifying address.
func firstIPv4(
	name string,
) string {
	addrs, err := interfaceAddrs(name)
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil {
			continue
		}
		if ip4.IsLinkLocalUnicast() {
			continue
		}
		return ip4.String()
	}
	return ""
}
