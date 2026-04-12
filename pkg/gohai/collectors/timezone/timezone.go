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

// Package timezone collects the system timezone (IANA name, short
// abbreviation, and offset from UTC).
package timezone

import (
	"strings"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// localtimePath is the standard symlink path on both Linux and macOS.
const localtimePath = "/etc/localtime"

// Info holds timezone data.
type Info struct {
	Name   string `json:"name"`             // IANA name: "America/Los_Angeles"
	Abbrev string `json:"abbrev,omitempty"` // Short code: "PDT", "PST", "UTC"
	Offset int    `json:"offset"`           // Seconds from UTC
}

// Collector is the public interface every timezone variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "timezone" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the timezone collector variant for the host OS. Linux and
// macOS use different zoneinfo prefixes; the per-OS structs carry their
// own.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// clockZone extracts the abbreviation and offset from the Go runtime's
// local clock. Shared by both variants — identical across OSes.
func clockZone(
	now func() time.Time,
) (string, int) {
	return now().Zone()
}

// resolveName tries readlink(localtimePath) first; on hosts where
// /etc/localtime is a copied file (Debian/Ubuntu installer output,
// container images) falls back to the file named by the `fallback`
// reader. Returns empty when neither source yields a name.
func resolveName(
	readlink func(string) (string, error),
	fallback func() (string, error),
	zoneinfoPrefix string,
) string {
	if target, err := readlink(localtimePath); err == nil {
		return strings.TrimPrefix(target, zoneinfoPrefix)
	}
	if name, err := fallback(); err == nil {
		return strings.TrimSpace(name)
	}
	return ""
}
