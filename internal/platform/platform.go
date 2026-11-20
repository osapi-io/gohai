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

// Package platform is how a collector picks its variant: Detect is the
// host's family, as pkg/gohai/family names it, and a test swaps it to force
// any branch without reading the host.
package platform

import "github.com/osapi-io/gohai/pkg/gohai/family"

// Detect returns the host's family: darwin on macOS, a family such as
// debian, rhel, fedora or amazon on Linux, the platform itself for a Linux
// host without a family, or "" when the host could not be read. Collectors
// dispatch on it:
//
//	switch platform.Detect() {
//	case family.Darwin: return NewDarwin()
//	case family.Debian: return NewDebian()
//	default:            return NewLinux()
//	}
//
// Tests override it:
//
//	orig := platform.Detect
//	defer func() { platform.Detect = orig }()
//	platform.Detect = func() string { return "darwin" }
var Detect = family.Detect

// IsLinux reports whether the host is any Linux distribution. For
// collectors that do the same thing on each one.
func IsLinux() bool {
	p := Detect()
	return p != family.Darwin && p != ""
}

// IsDarwin reports whether the host is macOS.
func IsDarwin() bool {
	return Detect() == family.Darwin
}
