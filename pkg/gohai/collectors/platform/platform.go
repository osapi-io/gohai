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

// Package platform reports OS identification — canonical name,
// version, family, and architecture. The collector prefers
// `/etc/os-release` (systemd standard) and falls back through legacy
// release files (redhat-release, SuSE-release, debian_version, etc.)
// so pre-systemd and appliance distributions are still identified.
// On macOS, `sw_vers` supplements gopsutil with the build identifier
// and the Rapid Security Response patch suffix.
package platform

import (
	"context"
	"regexp"
	"runtime"
	"strings"

	"github.com/avfs/avfs"
	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	plat "github.com/osapi-io/gohai/internal/platform"
)

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
// Kept private so importers don't transitively need gopsutil.
var hostInfoFn = host.InfoWithContext

// readPlatform is the production bridge to gopsutil. Returns
// pre-populated *Info plus the raw KernelVersion that Darwin
// optionally consumes as Build.
func readPlatform(
	ctx context.Context,
) (*Info, string, error) {
	info := &Info{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	h, err := hostInfoFn(ctx)
	if err != nil {
		return nil, "", err
	}
	if h == nil {
		return info, "", nil
	}
	info.Name = canonicalizePlatform(h.Platform)
	info.Version = h.PlatformVersion
	info.Family = h.PlatformFamily
	return info, h.KernelVersion, nil
}

// Info holds platform identification data.
type Info struct {
	OS           string `json:"os"`                      // runtime.GOOS: "linux", "darwin", "windows"
	Name         string `json:"name"`                    // distro/product: "ubuntu", "redhat", "darwin"
	Version      string `json:"version"`                 // "24.04", "14.4.1"
	VersionExtra string `json:"version_extra,omitempty"` // extra version info (macOS RSR patches)
	Family       string `json:"family"`                  // "debian", "rhel", "mac_os_x"
	Architecture string `json:"architecture"`            // "amd64", "arm64"
	Build        string `json:"build,omitempty"`         // OS build identifier (macOS BuildVersion)
}

// Collector is the public interface every platform variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "platform" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the platform variant for the host OS.
func New() Collector {
	if plat.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// platformIDRemap normalizes distro IDs into a canonical form so
// consumers don't have to special-case per-distro quirks. Matches
// Ohai's OS_RELEASE_PLATFORM_REMAP table.
//
// Source: https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/platform.rb
var platformIDRemap = map[string]string{
	"alinux":        "alibabalinux",
	"amzn":          "amazon",
	"archarm":       "arch",    // ArchLinux ARM (Pi, Pinebook)
	"cumulus-linux": "cumulus", // Cumulus Networks
	"ol":            "oracle",
	"opensuse-leap": "opensuseleap",
	"rhel":          "redhat",
	"rocky":         "rocky",
	"sles":          "suse",
	"sles_sap":      "suse", // SLES for SAP
	"xenenterprise": "xenserver",
}

// canonicalizePlatform applies platformIDRemap; returns input
// unchanged if no entry exists.
func canonicalizePlatform(
	p string,
) string {
	if m, ok := platformIDRemap[p]; ok {
		return m
	}
	return p
}

// platformFamilyTable is Ohai's exhaustive distro → family mapping
// (linux/platform.rb platform_family_from_platform). Used to fill in
// `family` when gopsutil's value is empty or unknown.
//
// Source: https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/platform.rb
var platformFamilyTable = map[string]string{
	// debian
	"debian": "debian", "ubuntu": "debian", "cumulus": "debian",
	"kali": "debian", "pop": "debian", "linuxmint": "debian", "raspbian": "debian",
	// rhel
	"rhel": "rhel", "redhat": "rhel", "centos": "rhel",
	"almalinux": "rhel", "rocky": "rhel",
	"alibabalinux": "rhel", "sangoma": "rhel", "clearos": "rhel",
	"parallels": "rhel", "virtuozzo": "rhel", "ibm_powerkvm": "rhel",
	"nexus_centos": "rhel", "bigip": "rhel", "xenserver": "rhel",
	"xcp-ng": "rhel", "cloudlinux": "rhel", "scientific": "rhel",
	"enterpriseenterprise": "rhel", "oracle": "rhel", "amazon": "rhel",
	// fedora
	"fedora": "fedora", "arista_eos": "fedora",
	// suse
	"suse": "suse", "sles": "suse", "opensuse": "suse",
	"opensuseleap": "suse", "sled": "suse",
	// arch
	"arch": "arch", "manjaro": "arch", "antergos": "arch",
	// gentoo
	"gentoo": "gentoo",
	// wrlinux
	"wrlinux": "wrlinux", "nexus": "wrlinux", "ios_xr": "wrlinux",
}

// familyFromPlatform looks up the canonical family for a distro ID.
// Returns "" when the ID is unknown — caller can fall back to
// gopsutil's value.
func familyFromPlatform(
	id string,
) string {
	return platformFamilyTable[id]
}

// minorVersionDistros are RHEL-family IDs (post-canonicalization)
// where /etc/os-release ships only the major version; the dotted
// minor lives in /etc/redhat-release.
var minorVersionDistros = map[string]bool{
	"centos":     true,
	"rocky":      true,
	"almalinux":  true,
	"redhat":     true,
	"rhel":       true,
	"scientific": true,
	"oracle":     true,
	"amazon":     true,
}

// majorOnlyVersionRE matches a `VERSION_ID` like `7` or `9` with no
// dot — the case that warrants the redhat-release supplement.
var majorOnlyVersionRE = regexp.MustCompile(`^\d+$`)

// redhatReleaseVersionRE extracts the dotted version from
// `/etc/redhat-release` lines like
// `CentOS Linux release 7.9.2009 (Core)` or
// `Red Hat Enterprise Linux release 9.3 (Plow)`.
var redhatReleaseVersionRE = regexp.MustCompile(`release\s+(\d[\d.]*)`)

// applyRedhatReleaseSupplement inspects info.Name + info.Version and
// supplements with /etc/redhat-release or /etc/debian_version when
// gopsutil's VERSION_ID is incomplete.
func applyRedhatReleaseSupplement(
	fs avfs.VFS,
	info *Info,
) {
	if minorVersionDistros[info.Name] && majorOnlyVersionRE.MatchString(info.Version) {
		if b, err := fs.ReadFile("/etc/redhat-release"); err == nil {
			if m := redhatReleaseVersionRE.FindStringSubmatch(string(b)); m != nil {
				info.Version = m[1]
			}
		}
	}
	if info.Name == "debian" && info.Version == "" {
		if b, err := fs.ReadFile("/etc/debian_version"); err == nil {
			info.Version = strings.TrimSpace(string(b))
		}
	}
}

// applyFamilyFallback fills info.Family from our table when gopsutil
// returned an empty family (long-tail distros gopsutil doesn't
// recognize).
func applyFamilyFallback(
	info *Info,
) {
	if info.Family == "" {
		info.Family = familyFromPlatform(info.Name)
	}
}

// legacyReleaseFile describes a /etc/*-release fallback parser.
type legacyReleaseFile struct {
	path   string
	parse  func(content string) (name, version string)
	family string
}

// genericReleaseRE matches lines like
// `Amazon Linux release 2 (Karoo)` or
// `CentOS release 6.10 (Final)`.
var genericReleaseRE = regexp.MustCompile(`^([A-Za-z][A-Za-z _]*?)\s+release\s+(\d[\d.]*)`)

// suseReleaseVersionRE / suseReleasePatchLevelRE parse
// /etc/SuSE-release.
var (
	suseReleaseVersionRE    = regexp.MustCompile(`(?m)^VERSION\s*=\s*(\S+)`)
	suseReleasePatchLevelRE = regexp.MustCompile(`(?m)^PATCHLEVEL\s*=\s*(\S+)`)
)

// parseGenericRelease returns the distro name and version from a
// generic `<name> release <version>` line. The captured name is
// reduced to its first whitespace-separated token, lowercased — so
// `CentOS Linux release 7.9` and `Gentoo Base System release 2.13`
// produce `centos` / `gentoo` rather than the noisier full prefix.
func parseGenericRelease(
	content string,
) (string, string) {
	m := genericReleaseRE.FindStringSubmatch(content)
	if m == nil {
		return "", ""
	}
	// genericReleaseRE requires `[A-Za-z]+` so m[1] always has at
	// least one whitespace-separated token after Fields.
	first := strings.Fields(m[1])
	return strings.ToLower(first[0]), m[2]
}

// parseSuseRelease returns ("suse", "<version>.<patchlevel>") from
// /etc/SuSE-release content. Drops PATCHLEVEL when absent.
func parseSuseRelease(
	content string,
) (string, string) {
	v := suseReleaseVersionRE.FindStringSubmatch(content)
	if v == nil {
		return "", ""
	}
	version := v[1]
	if pl := suseReleasePatchLevelRE.FindStringSubmatch(content); pl != nil {
		version = version + "." + pl[1]
	}
	return "suse", version
}

// legacyReleaseFiles is the cascade order matching Ohai's
// legacy_platform_detection.
var legacyReleaseFiles = []legacyReleaseFile{
	{path: "/etc/redhat-release", parse: parseGenericRelease, family: "rhel"},
	{path: "/etc/SuSE-release", parse: parseSuseRelease, family: "suse"},
	{path: "/etc/f5-release", parse: parseGenericRelease, family: "rhel"},
	{path: "/etc/system-release", parse: parseGenericRelease, family: "rhel"},
	{path: "/etc/debian_version", parse: func(c string) (string, string) {
		return "debian", strings.TrimSpace(c)
	}, family: "debian"},
	{
		path:   "/etc/arch-release",
		parse:  func(string) (string, string) { return "arch", "" },
		family: "arch",
	},
	{path: "/etc/gentoo-release", parse: parseGenericRelease, family: "gentoo"},
	{path: "/etc/slackware-version", parse: parseGenericRelease, family: "slackware"},
	{path: "/etc/enterprise-release", parse: parseGenericRelease, family: "rhel"},
	{path: "/etc/exherbo-release", parse: parseGenericRelease, family: "exherbo"},
}

// applyLegacyReleaseFallback walks the legacy /etc/*-release cascade
// when info.Name is empty (i.e., gopsutil + os-release produced
// nothing). First file that yields a name wins; the canonical
// platformIDRemap is applied to the result.
func applyLegacyReleaseFallback(
	fs avfs.VFS,
	info *Info,
) {
	if info.Name != "" {
		return
	}
	for _, f := range legacyReleaseFiles {
		b, err := fs.ReadFile(f.path)
		if err != nil {
			continue
		}
		name, version := f.parse(string(b))
		if name == "" {
			continue
		}
		info.Name = canonicalizePlatform(name)
		info.Version = version
		info.Family = f.family
		return
	}
}
