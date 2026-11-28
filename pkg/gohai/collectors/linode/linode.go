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
// uses. Detection is heuristic (apt sources list), and the output
// comes entirely from the host's own addresses, which `ip -o -4 addr
// show` gives through the sources. Returns nil with no error when no
// Linode signature is found.
package linode

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// ProviderName is the canonical cloud identifier this collector
// populates. Consumers switching on Facts.Cloud().Name match against
// gohai.CloudLinode, which re-exports this constant.
const ProviderName = "linode"

// aptSourcesPath is the file read for the has_linode_apt_repos? signal.
// Linode's official images ship an apt source referencing linode.com;
// custom-built images often do too.
const aptSourcesPath = "/etc/apt/sources.list"

// aptSourcesDir holds the apt source files of newer images, such as
// Ubuntu 24.04 and Debian 13, which keep their sources in
// sources.list.d/*.sources and not in sources.list.
const aptSourcesDir = "/etc/apt/sources.list.d"

// dmiVendorName is the SMBIOS sys_vendor of a Linode (Akamai) VM.
// cloud-init's ds-identify finds its Akamai datasource by it. It finds
// a Linode on an image with no apt file, such as an RPM distribution.
const dmiVendorName = "Linode"

// linodeSignature is the substring we look for in apt sources.
const linodeSignature = "linode"

// Info is the Linode view. Mirrors Ohai's node['linode'] which only
// reports two facts: public_ip and private_ip. No metadata service
// call: these come from the host's own eth0 / eth0:1 aliases.
type Info struct {
	PublicIP  string `json:"public_ip,omitempty"`
	PrivateIP string `json:"private_ip,omitempty"`
}

// Host is the linode collector. It detects Linode hosts from the host's
// files and reads the interface-derived IPs.
type Host struct {
	// FS is the filesystem that the apt sources are read from.
	FS avfs.VFS
	// Exec runs `ip -o -4 addr show`, which gives the addresses of eth0
	// and its eth0:1 alias.
	Exec source.Runner
}

// Collector is the public interface of the linode collector.
type Collector interface {
	collector.Collector
}

var _ Collector = (*Host)(nil)

// New returns the linode collector on the real host.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the linode collector reading the host through src.
func NewWith(
	src source.Sources,
) Collector {
	return collector.Wire(&Host{}, src)
}

// Name returns "linode".
func (*Host) Name() string { return "linode" }

// Category returns "cloud".
func (*Host) Category() collector.Category { return collector.CategoryCloud }

// DefaultEnabled returns false, because cloud collectors are opt-in.
func (*Host) DefaultEnabled() bool { return false }

// Dependencies declares dmi, for the vendor of a Linode VM, and hostname
// so the FQDN/Domain check can run against the host's resolved name
// (matches Ohai's has_linode_domain? signal which reads node['domain']).
func (*Host) Dependencies() []string { return []string{"dmi", "hostname"} }

// Collect returns the Linode Info when any detection signal fires
// (the DMI vendor is Linode, an apt source names linode, or the domain
// contains "linode"), else (nil, nil). The apt and domain signals are
// Ohai's looks_like_linode? OR chain.
func (c *Host) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	if !onLinode(c.FS, prior) {
		return nil, ctx.Err()
	}
	info := &Info{}
	if c.Exec == nil {
		return info, nil
	}
	out, err := c.Exec.Execute(ctx, "ip", "-o", "-4", "addr", "show")
	if collector.Absent(err) {
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("linode: ip addr: %w", err)
	}
	addrs := ipv4ByLabel(out)
	info.PublicIP = addrs["eth0"]
	info.PrivateIP = addrs["eth0:1"]
	return info, nil
}

// onLinode runs these detection signals:
//   - dmi's sys_vendor is "Linode" (cloud-init's Akamai signal)
//   - /etc/apt/sources.list, or a file in /etc/apt/sources.list.d,
//     contains "linode" (Ohai's has_linode_apt_repos?, with the
//     directory of newer images added)
//   - The host's FQDN or Domain contains "linode" (has_linode_domain?,
//     e.g. members.linode.com)
func onLinode(
	fs avfs.VFS,
	prior collector.PriorResults,
) bool {
	if d, ok := collector.GetDep[*dmi.Info](prior, "dmi"); ok && d != nil &&
		d.Product != nil && d.Product.VendorName == dmiVendorName {
		return true
	}
	if aptNamesLinode(fs) {
		return true
	}
	if h, ok := collector.GetDep[*hostname.Info](prior, "hostname"); ok && h != nil {
		if strings.Contains(strings.ToLower(h.FQDN), linodeSignature) ||
			strings.Contains(strings.ToLower(h.Domain), linodeSignature) {
			return true
		}
	}
	return false
}

// aptNamesLinode reports whether /etc/apt/sources.list or a file in
// /etc/apt/sources.list.d contains "linode". A file that cannot be read
// is no signal.
func aptNamesLinode(
	fs avfs.VFS,
) bool {
	paths := []string{aptSourcesPath}
	if entries, err := fs.ReadDir(aptSourcesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				paths = append(paths, aptSourcesDir+"/"+e.Name())
			}
		}
	}
	for _, path := range paths {
		b, err := fs.ReadFile(path)
		if err == nil && bytes.Contains(bytes.ToLower(b), []byte(linodeSignature)) {
			return true
		}
	}
	return false
}

// ipv4ByLabel parses `ip -o -4 addr show` into the first IPv4 address
// of each label that is not link-local. The label is the interface name,
// or an alias such as eth0:1, the last word before the backslash:
//
//	2: eth0    inet 50.1.2.3/24 brd 50.1.2.255 scope global eth0\  valid_lft forever
//	2: eth0    inet 10.0.0.5/17 scope global eth0:1\  valid_lft forever
func ipv4ByLabel(
	out []byte,
) map[string]string {
	addrs := map[string]string{}
	for line := range strings.SplitSeq(string(out), "\n") {
		head, _, _ := strings.Cut(line, "\\")
		fields := strings.Fields(head)
		inet := slices.Index(fields, "inet")
		if inet < 0 || inet+1 >= len(fields) {
			continue
		}
		ip, _, err := net.ParseCIDR(fields[inet+1])
		if err != nil || ip.To4() == nil || ip.IsLinkLocalUnicast() {
			continue
		}
		label := fields[len(fields)-1]
		if _, ok := addrs[label]; !ok {
			addrs[label] = ip.String()
		}
	}
	return addrs
}
