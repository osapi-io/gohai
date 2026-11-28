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

// Package family names a host's platform and its platform family, as Chef's
// Ohai does: the platform is the ID of /etc/os-release with a few IDs
// remapped, and the family follows from the platform. Collectors pick their
// variant from Detect, and the platform collector reports Platform and Of.
// osapi names its behaviors' platforms with the same families.
//
// One difference from Ohai is on purpose: on macOS the platform and the
// family are "darwin", the Go and kernel name, not Ohai's "mac_os_x".
//
// Source: https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/platform.rb
package family

import (
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
)

// The families. Each is a value Of and Detect return.
const (
	Darwin     = "darwin"
	Debian     = "debian"
	RHEL       = "rhel"
	Amazon     = "amazon"
	SUSE       = "suse"
	Fedora     = "fedora"
	WRLinux    = "wrlinux"
	Gentoo     = "gentoo"
	Arch       = "arch"
	Exherbo    = "exherbo"
	Alpine     = "alpine"
	ClearLinux = "clearlinux"
	Mandriva   = "mandriva"
	Slackware  = "slackware"
)

// platformRemap is Ohai's platform_id_remap: the os-release IDs whose
// platform name differs from the ID.
var platformRemap = map[string]string{
	"alinux":        "alibabalinux",
	"amzn":          "amazon",
	"archarm":       "arch",
	"cumulus-linux": "cumulus",
	"ol":            "oracle",
	"opensuse-leap": "opensuseleap",
	"rhel":          "redhat",
	"sles_sap":      "suse",
	"sles":          "suse",
	"xenenterprise": "xenserver",
}

// Platform is the platform name of an os-release ID, as Ohai names it: the
// ID in lower case, or its remapped name, such as redhat for rhel and
// amazon for amzn.
func Platform(
	id string,
) string {
	id = strings.ToLower(id)
	if p, ok := platformRemap[id]; ok {
		return p
	}

	return id
}

// families is Ohai's platform_family_from_platform, in its order: the first
// family whose any match is in the platform name is the family. Ohai matches
// by regular expression; each of its expressions is a plain substring, but
// for sle[sd\-_], which is four.
var families = []struct {
	family  string
	matches []string
}{
	{Debian, []string{"ubuntu", "debian", "linuxmint", "raspbian", "cumulus", "kali", "pop"}},
	// Ohai keeps rhel for rebuilds that track RHEL's versions; other RPM
	// distributions are fedora.
	{RHEL, []string{
		"centos", "redhat", "oracle", "almalinux", "rocky", "scientific",
		"enterpriseenterprise", "xenserver", "xcp-ng", "cloudlinux", "alibabalinux",
		"sangoma", "clearos", "parallels", "ibm_powerkvm", "nexus_centos", "bigip",
		"virtuozzo",
	}},
	{Amazon, []string{"amazon"}},
	{SUSE, []string{"suse", "sles", "sled", "sle-", "sle_"}},
	{Fedora, []string{"fedora", "arista_eos"}},
	{WRLinux, []string{"nexus", "ios_xr"}},
	{Gentoo, []string{"gentoo"}},
	{Arch, []string{"arch", "manjaro"}},
	{Exherbo, []string{"exherbo"}},
	{Alpine, []string{"alpine"}},
	{ClearLinux, []string{"clearlinux"}},
	// Ohai spells this "mangeia"; mageia is the distribution.
	{Mandriva, []string{"mangeia", "mageia"}},
	{Slackware, []string{"slackware"}},
}

// Of is the family of a platform, such as debian for ubuntu and rhel for
// rocky, or "" for a platform Ohai has no family for. darwin is darwin.
func Of(
	platform string,
) string {
	platform = strings.ToLower(platform)
	if platform == Darwin {
		return Darwin
	}
	for _, f := range families {
		for _, m := range f.matches {
			if strings.Contains(platform, m) {
				return f.family
			}
		}
	}

	return ""
}

// osReleasePaths are the files that name the distribution, in the order
// that os-release(5) gives: /etc/os-release, then /usr/lib/os-release when
// the first does not exist.
var osReleasePaths = [...]string{"/etc/os-release", "/usr/lib/os-release"}

// ReadOSRelease is the content of the host's os-release file, read through
// vfs: /etc/os-release, or /usr/lib/os-release when the first does not
// exist. Its error is the last path's when neither exists, so
// errors.Is(err, fs.ErrNotExist) tells a host with no file, and any other
// read error is returned at once.
func ReadOSRelease(
	vfs avfs.VFS,
) ([]byte, error) {
	var err error
	for _, path := range osReleasePaths {
		var b []byte
		b, err = vfs.ReadFile(path)
		if err == nil {
			return b, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}

	return nil, err
}

// hostFS is the filesystem Detect reads. Swapped in tests.
var hostFS avfs.VFS = osfs.NewWithNoIdm()

// goos is the operating system the binary runs on. Swapped in tests.
var goos = runtime.GOOS

// Detect is the family of this host: darwin on macOS, else the family of
// the platform that /etc/os-release names. A Linux host whose platform has
// no family returns its platform, so a caller can still tell it is Linux,
// and "" is a host whose /etc/os-release could not be read.
func Detect() string {
	if goos == Darwin {
		return Darwin
	}

	return FromFS(hostFS)
}

// macOSVersion is the file that each macOS host has and no Linux host has.
const macOSVersion = "/System/Library/CoreServices/SystemVersion.plist"

// FromFS is the family of the host whose filesystem is fs: darwin when fs
// has macOS's SystemVersion.plist and no /etc/os-release; otherwise the
// family of the platform that its /etc/os-release ID names, the platform
// itself when it has no family, or "" when the file has no ID.
func FromFS(
	fs avfs.VFS,
) string {
	data, err := ReadOSRelease(fs)
	if err != nil {
		if _, statErr := fs.Stat(macOSVersion); statErr == nil {
			return Darwin
		}
		return ""
	}
	var id, like string
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "ID="); ok {
			id = strings.Trim(strings.TrimSpace(v), `"'`)
		}
		if v, ok := strings.CutPrefix(line, "ID_LIKE="); ok {
			like = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	if id == "" {
		return ""
	}
	p := Platform(id)
	if f := Of(p); f != "" {
		return f
	}
	// A derivative that Ohai's table does not name, such as elementary or
	// zorin, names its parents in ID_LIKE: the first parent with a family
	// gives it.
	for _, parent := range strings.Fields(like) {
		if f := Of(Platform(parent)); f != "" {
			return f
		}
	}

	return p
}
