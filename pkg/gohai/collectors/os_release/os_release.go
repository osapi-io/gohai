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

// Package osrelease parses /etc/os-release into typed fields. Matches
// Ohai's linux/os_release plugin — every field documented in os-release(5).
package osrelease

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

const osReleasePath = "/etc/os-release"

// Info holds the typed fields parsed from /etc/os-release. Unknown
// keys are preserved in Extra.
type Info struct {
	ID              string            `json:"id,omitempty"`
	IDLike          []string          `json:"id_like,omitempty"`
	Name            string            `json:"name,omitempty"`
	PrettyName      string            `json:"pretty_name,omitempty"`
	Version         string            `json:"version,omitempty"`
	VersionID       string            `json:"version_id,omitempty"`
	VersionCodename string            `json:"version_codename,omitempty"`
	BuildID         string            `json:"build_id,omitempty"`
	Variant         string            `json:"variant,omitempty"`
	VariantID       string            `json:"variant_id,omitempty"`
	HomeURL         string            `json:"home_url,omitempty"`
	SupportURL      string            `json:"support_url,omitempty"`
	BugReportURL    string            `json:"bug_report_url,omitempty"`
	Extra           map[string]string `json:"extra,omitempty"` // any keys we don't parse explicitly
}

// Collector is the public interface every os_release variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "os_release" }
func (base) Category() string       { return collector.CategorySystem }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the os_release variant for the host OS. Linux only —
// macOS has no equivalent (sw_vers is already wrapped by platform).
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// parseOSRelease parses os-release(5) format. Each non-comment, non-blank
// line is KEY=VALUE; values may be quoted. ID_LIKE is space-separated.
func parseOSRelease(
	content string,
) *Info {
	info := &Info{Extra: map[string]string{}}
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
		case "ID":
			info.ID = val
		case "ID_LIKE":
			info.IDLike = strings.Fields(val)
		case "NAME":
			info.Name = val
		case "PRETTY_NAME":
			info.PrettyName = val
		case "VERSION":
			info.Version = val
		case "VERSION_ID":
			info.VersionID = val
		case "VERSION_CODENAME":
			info.VersionCodename = val
		case "BUILD_ID":
			info.BuildID = val
		case "VARIANT":
			info.Variant = val
		case "VARIANT_ID":
			info.VariantID = val
		case "HOME_URL":
			info.HomeURL = val
		case "SUPPORT_URL":
			info.SupportURL = val
		case "BUG_REPORT_URL":
			info.BugReportURL = val
		default:
			info.Extra[key] = val
		}
	}
	if len(info.Extra) == 0 {
		info.Extra = nil
	}
	return info
}
