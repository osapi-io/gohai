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

// Package lsb parses /etc/lsb-release — the Debian/Ubuntu-specific LSB
// metadata file. Matches Ohai's linux/lsb plugin. Mostly a legacy
// data source; os-release has superseded it on modern distros but
// Debian derivatives still populate /etc/lsb-release.
package lsb

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

const lsbReleasePath = "/etc/lsb-release"

// Info holds LSB release data.
type Info struct {
	ID          string `json:"id,omitempty"`          // DISTRIB_ID    (e.g., "Ubuntu")
	Release     string `json:"release,omitempty"`     // DISTRIB_RELEASE (e.g., "24.04")
	Codename    string `json:"codename,omitempty"`    // DISTRIB_CODENAME (e.g., "noble")
	Description string `json:"description,omitempty"` // DISTRIB_DESCRIPTION
}

// Collector is the public interface every lsb variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "lsb" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the lsb variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// parseLSBRelease parses lsb-release format: KEY=VALUE, values may
// be quoted with double quotes.
func parseLSBRelease(
	content string,
) *Info {
	info := &Info{}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		key := line[:i]
		val := strings.Trim(line[i+1:], `"'`)
		switch key {
		case "DISTRIB_ID":
			info.ID = val
		case "DISTRIB_RELEASE":
			info.Release = val
		case "DISTRIB_CODENAME":
			info.Codename = val
		case "DISTRIB_DESCRIPTION":
			info.Description = val
		}
	}
	return info
}
