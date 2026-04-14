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

package packagemgr

import "context"

// Linux covers generic Linux hosts where we don't know the family —
// arch, alpine, suse, etc. Probes common package managers in order of
// likelihood.
type Linux struct {
	base
}

// NewLinux returns a Linux variant.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect identifies the package manager. Checks distros whose
// platform.Detect return values don't fall into the debian/rhel
// families: zypper (SUSE), pacman (Arch), apk (Alpine),
// xbps-install (Void), emerge (Gentoo). Empty Name + Path if none
// found (likely a minimal image).
//
// apt/dnf/yum are intentionally NOT probed here — those hosts dispatch
// to the Debian or RHEL variant before ever reaching this one.
func (l *Linux) Collect(
	_ context.Context,
) (any, error) {
	name, path := firstFound("zypper", "pacman", "apk", "xbps-install", "emerge")
	return &Info{Name: name, Path: path}, nil
}
