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
// On macOS, `sw_vers` gives the version, the build identifier and the
// Rapid Security Response patch suffix.
package platform

import (
	"regexp"
	"strings"

	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/family"
)

// osRelease names the distribution: its ID and VERSION_ID.
const osRelease = "/etc/os-release"

// readOSRelease fills info's name, version and family from
// /etc/os-release, as Ohai does. It leaves them empty when fs has no such
// file, for the old release files to fill.
func readOSRelease(
	fs avfs.VFS,
	info *Info,
) {
	b, err := fs.ReadFile(osRelease)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch key {
		case "ID":
			info.Name = family.Platform(value)
		case "VERSION_ID":
			info.Version = value
		}
	}
	info.Family = family.Of(info.Name)
}

// Info holds platform identification data.
type Info struct {
	OS              string `json:"os"`                      // runtime.GOOS: "linux", "darwin", "windows"
	Name            string `json:"name"`                    // distro/product: "ubuntu", "redhat", "darwin"
	Version         string `json:"version"`                 // "24.04", "14.4.1"
	VersionExtra    string `json:"version_extra,omitempty"` // extra version info (macOS RSR patches)
	Family          string `json:"family"`                  // "debian", "rhel", "darwin"
	CPUArchitecture string `json:"cpu_architecture"`        // "amd64", "arm64"
	Build           string `json:"build,omitempty"`         // OS build identifier (macOS BuildVersion)
}

// Collector is the public interface every platform variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "platform" }
func (base) Category() string       { return collector.CategorySystem }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the platform variant for the host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
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
// the VERSION_ID of os-release is incomplete.
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
	// Ohai's get_redhatish_platform: Red Hat's own file names redhat, not
	// its first word.
	if strings.HasPrefix(strings.ToLower(m[1]), "red hat") {
		return "redhat", m[2]
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
// when info.Name is empty (i.e., os-release produced
// nothing). First file that yields a name wins; the canonical
// family.Platform is applied to the result.
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
		info.Name = family.Platform(name)
		info.Version = version
		info.Family = family.Of(info.Name)
		if info.Family == "" {
			info.Family = f.family
		}
		return
	}
}
