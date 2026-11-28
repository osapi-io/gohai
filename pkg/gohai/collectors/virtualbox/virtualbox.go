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
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pattern"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds VirtualBox guest additions data gathered from the guest VM.
type Info struct {
	HostVersion            string `json:"version,omitempty"`                  // VBoxVer guest property: the VirtualBox version of the host
	HostRevision           string `json:"revision,omitempty"`                 // VBoxRev guest property: the VirtualBox revision of the host
	GuestAdditionsVersion  string `json:"guest_additions_version,omitempty"`  // GuestAdd/VersionExt: the Guest Additions version
	GuestAdditionsRevision string `json:"guest_additions_revision,omitempty"` // GuestAdd/Revision: the Guest Additions revision
	LanguageID             string `json:"language_id,omitempty"`              // LanguageID guest property: the locale of the host
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
func (base) Category() collector.Category { return collector.CategoryVirtualization }

// DefaultEnabled returns false, because virtualbox is opt-in only.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the virtualbox collector variant appropriate to the detected
// host OS.
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

// reLanguageID extracts the LanguageID value from VBoxControl output.
var reLanguageID = pattern.New(`LanguageID, value: (\S+),`)

// reVBoxVer extracts the VBoxVer value (host VirtualBox version).
var reVBoxVer = pattern.New(`VBoxVer, value: (\S+),`)

// reVBoxRev extracts the VBoxRev value (host VirtualBox revision).
var reVBoxRev = pattern.New(`VBoxRev, value: (\S+),`)

// reGuestAddVer extracts the GuestAdd/VersionExt value.
var reGuestAddVer = pattern.New(`GuestAdd/VersionExt, value: (\S+),`)

// reGuestAddRev extracts the GuestAdd/Revision value.
var reGuestAddRev = pattern.New(`GuestAdd/Revision, value: (\S+),`)

// parseVBoxControlOutput parses `VBoxControl guestproperty enumerate` output
// into an Info struct. Mirrors Ohai's virtualbox.rb regex set for the guest
// branch.
func parseVBoxControlOutput(
	output []byte,
) *Info {
	info := &Info{}
	// strings.Lines, not a bufio.Scanner, which stops with no error at a
	// line over 64 KiB and drops every line after it.
	for rawLine := range strings.Lines(string(output)) {
		line := strings.TrimRight(rawLine, "\r\n")
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
