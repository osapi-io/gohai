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

// Package virtualbox collects VirtualBox guest additions data from a guest VM.
// It requires VBoxControl to be installed (part of VirtualBox Guest Additions).
// When the host is not a VirtualBox guest, Collect returns nil with no error.
// Only the guest role is supported; host-side VBoxManage enumeration is not
// implemented (Ohai's host branch requires VBoxManage, which is a much heavier
// dependency).
package virtualbox

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds VirtualBox guest additions data gathered from the guest VM.
type Info struct {
	HostVersion            string `json:"version,omitempty"`                  // VBoxVer guest property — VirtualBox host version
	HostRevision           string `json:"revision,omitempty"`                 // VBoxRev guest property — VirtualBox host revision
	GuestAdditionsVersion  string `json:"guest_additions_version,omitempty"`  // GuestAdd/VersionExt — Guest Additions version
	GuestAdditionsRevision string `json:"guest_additions_revision,omitempty"` // GuestAdd/Revision — Guest Additions revision
	LanguageID             string `json:"language_id,omitempty"`              // LanguageID guest property — host locale
}

// Collector is the public interface every virtualbox variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "virtualbox".
func (base) Name() string { return "virtualbox" }

// Category returns "virtualization".
func (base) Category() string { return collector.CategoryVirtualization }

// DefaultEnabled returns false — virtualbox is opt-in only.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the virtualbox collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// reLanguageID extracts the LanguageID value from VBoxControl output.
var reLanguageID = regexp.MustCompile(`LanguageID, value: (\S+),`)

// reVBoxVer extracts the VBoxVer value (host VirtualBox version).
var reVBoxVer = regexp.MustCompile(`VBoxVer, value: (\S+),`)

// reVBoxRev extracts the VBoxRev value (host VirtualBox revision).
var reVBoxRev = regexp.MustCompile(`VBoxRev, value: (\S+),`)

// reGuestAddVer extracts the GuestAdd/VersionExt value.
var reGuestAddVer = regexp.MustCompile(`GuestAdd/VersionExt, value: (\S+),`)

// reGuestAddRev extracts the GuestAdd/Revision value.
var reGuestAddRev = regexp.MustCompile(`GuestAdd/Revision, value: (\S+),`)

// parseVBoxControlOutput parses `VBoxControl guestproperty enumerate` output
// into an Info struct. Mirrors Ohai's virtualbox.rb regex set for the guest
// branch.
func parseVBoxControlOutput(
	output []byte,
) *Info {
	info := &Info{}
	sc := bufio.NewScanner(strings.NewReader(string(output)))
	for sc.Scan() {
		line := sc.Text()
		if m := reLanguageID.FindStringSubmatch(line); m != nil {
			info.LanguageID = m[1]
			continue
		}
		if m := reVBoxVer.FindStringSubmatch(line); m != nil {
			info.HostVersion = m[1]
			continue
		}
		if m := reVBoxRev.FindStringSubmatch(line); m != nil {
			info.HostRevision = m[1]
			continue
		}
		if m := reGuestAddVer.FindStringSubmatch(line); m != nil {
			info.GuestAdditionsVersion = m[1]
			continue
		}
		if m := reGuestAddRev.FindStringSubmatch(line); m != nil {
			info.GuestAdditionsRevision = m[1]
		}
	}
	return info
}
