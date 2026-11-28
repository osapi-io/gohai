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

// Package platform reports OS identification: canonical name,
// version, family, and architecture. The collector prefers
// `/etc/os-release` (systemd standard) and falls back through legacy
// release files (redhat-release, SuSE-release, debian_version, etc.)
// so pre-systemd and appliance distributions are still identified.
// On macOS, `sw_vers` gives the version, the build identifier and the
// Rapid Security Response patch suffix.
package platform

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// readOSRelease fills info's name, version and family from
// /etc/os-release, as Ohai does. It leaves them empty when fs has no such
// file, for the old release files to fill. A file that is there and
// can't be read is an error.
func readOSRelease(
	fs avfs.VFS,
	info *Info,
) error {
	b, err := family.ReadOSRelease(fs)
	if collector.Absent(err) {
		return nil
	}
	if err != nil {
		return err
	}
	pairs := family.ParseOSRelease(b)
	if id, ok := pairs["ID"]; ok {
		info.Name = family.Platform(id)
	}
	if version, ok := pairs["VERSION_ID"]; ok {
		info.Version = version
	}
	info.Family = family.Of(info.Name)
	return nil
}

// readOptional returns the content of path, or nil when fs has no such
// file. A file that is there and can't be read is an error.
func readOptional(
	fs avfs.VFS,
	path string,
) ([]byte, error) {
	b, err := fs.ReadFile(path)
	if collector.Absent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
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

func (base) Name() string                 { return "platform" }
func (base) Category() collector.Category { return collector.CategorySystem }
func (base) DefaultEnabled() bool         { return true }
func (base) Dependencies() []string       { return nil }

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
// dot. That is the case that warrants the redhat-release supplement.
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
) error {
	if minorVersionDistros[info.Name] && majorOnlyVersionRE.MatchString(info.Version) {
		b, err := readOptional(fs, "/etc/redhat-release")
		if err != nil {
			return err
		}
		if m := redhatReleaseVersionRE.FindStringSubmatch(string(b)); m != nil {
			info.Version = m[1]
		}
	}
	if info.Name == "debian" && info.Version == "" {
		b, err := readOptional(fs, "/etc/debian_version")
		if err != nil {
			return err
		}
		info.Version = strings.TrimSpace(string(b))
	}
	return nil
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

// slackwareVersionRE matches /etc/slackware-version, which has no
// `release` word: `Slackware 15.0`.
var slackwareVersionRE = regexp.MustCompile(`^Slackware\s+(\d[\d.]*)`)

// suseReleaseVersionRE / suseReleasePatchLevelRE parse
// /etc/SuSE-release.
var (
	suseReleaseVersionRE    = regexp.MustCompile(`(?m)^VERSION\s*=\s*(\S+)`)
	suseReleasePatchLevelRE = regexp.MustCompile(`(?m)^PATCHLEVEL\s*=\s*(\S+)`)
)

// parseGenericRelease returns the distro name and version from a
// generic `<name> release <version>` line. The captured name is
// reduced to its first whitespace-separated token, lowercased, so
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

// parseSlackwareVersion returns ("slackware", "<version>") from
// /etc/slackware-version content, as Ohai's /Slackware ([\d\.]+)/ does.
func parseSlackwareVersion(
	content string,
) (string, string) {
	m := slackwareVersionRE.FindStringSubmatch(content)
	if m == nil {
		return "", ""
	}
	return "slackware", m[1]
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
	{path: "/etc/slackware-version", parse: parseSlackwareVersion, family: "slackware"},
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
) error {
	if info.Name != "" {
		return nil
	}
	for _, f := range legacyReleaseFiles {
		b, err := readOptional(fs, f.path)
		if err != nil {
			return err
		}
		if b == nil {
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
		return nil
	}
	return nil
}

// goArch is the architecture of the gohai binary: the answer when the host
// cannot be asked.
var goArch = runtime.GOARCH

// machineArch maps a kernel's machine name, as `uname -m` prints it, to the
// Go name of the architecture.
var machineArch = map[string]string{
	"x86_64":  "amd64",
	"amd64":   "amd64",
	"i386":    "386",
	"i486":    "386",
	"i586":    "386",
	"i686":    "386",
	"aarch64": "arm64",
	"arm64":   "arm64",
	"armv6l":  "arm",
	"armv7l":  "arm",
	"armv8l":  "arm",
	"ppc64le": "ppc64le",
	"ppc64":   "ppc64",
	"s390x":   "s390x",
	"riscv64": "riscv64",
	"mips64":  "mips64",
	"loong64": "loong64",
}

// unameArch is the Go name of the host's architecture, from `uname -m`.
// A machine name with no Go name is reported as uname gives it. When exec
// is nil, uname is not on the host or prints nothing, it is the binary's
// architecture. A uname that runs and fails is an error, as the binary's
// architecture can differ from the host's.
func unameArch(
	ctx context.Context,
	exec source.Runner,
) (string, error) {
	if exec == nil {
		return goArch, nil
	}
	out, err := exec.Execute(ctx, "uname", "-m")
	if collector.Absent(err) {
		return goArch, nil
	}
	if err != nil {
		return "", fmt.Errorf("uname -m: %w", err)
	}
	m := strings.TrimSpace(string(out))
	if m == "" {
		return goArch, nil
	}
	if a, ok := machineArch[m]; ok {
		return a, nil
	}

	return m, nil
}

// darwinArch is the Go name of a Mac's architecture. `sysctl -n
// hw.optional.arm64` is 1 on Apple silicon, also for a process that Rosetta
// translates, where `uname -m` says x86_64. An Intel Mac has no such key,
// and its sysctl fails, so any other answer means uname's answer.
func darwinArch(
	ctx context.Context,
	exec source.Runner,
) (string, error) {
	if exec != nil {
		if out, err := exec.Execute(ctx, "sysctl", "-n", "hw.optional.arm64"); err == nil &&
			strings.TrimSpace(string(out)) == "1" {
			return "arm64", nil
		}
	}

	return unameArch(ctx, exec)
}
