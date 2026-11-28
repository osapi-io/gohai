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
	"path/filepath"
	"strings"
	"time"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pattern"
	"github.com/osapi-io/gohai/pkg/gohai/source"
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

func (base) Name() string                 { return "timezone" }
func (base) Category() collector.Category { return collector.CategorySystem }
func (base) DefaultEnabled() bool         { return true }
func (base) Dependencies() []string       { return nil }

// New returns the timezone collector variant for the host OS. Linux and
// macOS use different zoneinfo prefixes; the per-OS structs carry their
// own.
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

// clockZone is the abbreviation and offset of the clock's time in the zone
// of the host's /etc/localtime, read through fs, so that a change of zone
// shows at the next collection, and a fake filesystem gives its own zone.
// When the file cannot be read or parsed, it is the process's local zone.
func clockZone(
	fs avfs.VFS,
	now func() time.Time,
) (string, int) {
	t := now()
	if fs != nil {
		if data, err := fs.ReadFile(localtimePath); err == nil {
			if loc, err := time.LoadLocationFromTZData("localtime", data); err == nil {
				return t.In(loc).Zone()
			}
		}
	}

	return t.Zone()
}

// ianaName is the shape of an IANA zone name, such as Europe/Paris, Etc/UTC
// or UTC.
var ianaName = pattern.New(`^[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)*$`)

// zoneinfoDir is the directory part that precedes an IANA name in a zone
// file's path, wherever the tzdata lives: /usr/share/zoneinfo/ on glibc,
// /etc/zoneinfo/ on NixOS, /var/db/timezone/zoneinfo/ on macOS.
const zoneinfoDir = "/zoneinfo/"

// nameFromPath is the IANA name in a zone file's path: the part after the
// last zoneinfo directory, or "" when the path has none or the rest is not
// an IANA name.
func nameFromPath(
	path string,
) string {
	i := strings.LastIndex(path, zoneinfoDir)
	if i < 0 {
		return ""
	}
	name := path[i+len(zoneinfoDir):]
	if !ianaName.MatchString(name) {
		return ""
	}

	return name
}

// resolveName is the IANA name of the host's zone: from the target of the
// /etc/localtime link, else from the first fallback that gives one. A
// relative link, such as ../usr/share/zoneinfo/Etc/UTC on Ubuntu, is relative
// to /etc. A name that is not IANA-shaped, such as a path, is not a name.
func resolveName(
	readlink func(string) (string, error),
	fallbacks ...func() (string, error),
) string {
	if target, err := readlink(localtimePath); err == nil {
		abs := target
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(filepath.Dir(localtimePath), target)
		}
		if name := nameFromPath(abs); name != "" {
			return name
		}
		if ianaName.MatchString(target) {
			return target
		}
	}
	for _, fallback := range fallbacks {
		name, err := fallback()
		if err != nil {
			continue
		}
		if name = strings.TrimSpace(name); ianaName.MatchString(name) {
			return name
		}
	}
	return ""
}
