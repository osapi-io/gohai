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

// Package platform provides cross-platform OS/distro detection used by
// gohai collectors to dispatch to the correct per-OS implementation.
// Mirrors OSAPI's pkg/sdk/platform so we share the same dispatch vocabulary.
package platform

import (
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

// HostInfoFn is the function used to retrieve host information. Override
// in tests to simulate different platforms.
var HostInfoFn = host.Info

// debianFamily lists distributions that share the debian provider
// implementations (apt, systemd, netplan). Matches OSAPI's list.
var debianFamily = map[string]bool{
	"ubuntu":   true,
	"debian":   true,
	"raspbian": true,
}

// rhelFamily lists distributions that share RHEL-derived tooling (dnf/yum,
// systemd). Added by gohai for collectors whose Debian variant doesn't
// cover them (e.g. package_mgr).
var rhelFamily = map[string]bool{
	"rhel":   true,
	"redhat": true,
	"centos": true,
	"fedora": true,
	"rocky":  true,
	"alma":   true,
	"amzn":   true,
	"amazon": true,
	"oracle": true,
	"ol":     true,
}

// Detect returns the OS family name used to pick a collector variant.
//
// Return values:
//   - "darwin" on macOS
//   - "debian" for debian/ubuntu/raspbian
//   - "rhel"   for rhel/redhat/centos/fedora/rocky/alma/amazon/oracle
//   - ""       for generic linux (arch, alpine, suse, gentoo, etc.) or
//     unknown. Collectors treat empty as "use the generic Linux variant".
//
// Callers dispatch with a switch statement:
//
//	switch platform.Detect() {
//	case "darwin":  return NewDarwin()
//	case "debian":  return NewDebian()
//	case "rhel":    return NewRHEL()
//	default:        return NewLinux()
//	}
func Detect() string {
	info, _ := HostInfoFn()
	if info == nil {
		return ""
	}

	p := strings.ToLower(info.Platform)
	if p == "" && strings.ToLower(info.OS) == "darwin" {
		return "darwin"
	}

	if debianFamily[p] {
		return "debian"
	}
	if rhelFamily[p] {
		return "rhel"
	}
	return p
}

// IsLinux reports whether the host is any Linux distribution, including
// debian- and rhel-family hosts. Convenience for collectors that do the
// same thing on all Linux variants.
func IsLinux() bool {
	p := Detect()
	return p != "darwin" && p != ""
}

// IsDarwin reports whether the host is macOS.
func IsDarwin() bool {
	return Detect() == "darwin"
}
