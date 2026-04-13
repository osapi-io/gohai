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

// Package lsb reports Linux Standard Base identification fields via
// the `lsb_release` CLI — matches Ohai's linux/lsb plugin. The legacy
// /etc/lsb-release file fallback was deliberately removed by Ohai in
// chef/ohai#1562; we match that stance.
package lsb

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds LSB release data.
type Info struct {
	ID          string `json:"id,omitempty"`          // Distributor ID (e.g., "Ubuntu")
	Release     string `json:"release,omitempty"`     // Release number (e.g., "24.04")
	Codename    string `json:"codename,omitempty"`    // Release codename (e.g., "noble")
	Description string `json:"description,omitempty"` // Human-readable description
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

// parseLsbReleaseCLI parses `lsb_release -a` output. The CLI emits one
// labelled field per line — matches Ohai's regex set in linux/lsb.rb.
func parseLsbReleaseCLI(
	content string,
) *Info {
	info := &Info{}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		switch key {
		case "Distributor ID":
			info.ID = val
		case "Release":
			info.Release = val
		case "Codename":
			info.Codename = val
		case "Description":
			info.Description = val
		}
	}
	return info
}
