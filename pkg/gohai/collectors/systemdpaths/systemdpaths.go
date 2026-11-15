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

// Package systemdpaths reports standard systemd directory paths by
// running `systemd-path` and parsing its "name: /path" output. Matches
// Ohai's linux/systemd_paths plugin. Darwin returns nil — systemd is
// Linux-only. DefaultEnabled is false.
package systemdpaths

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the systemd path map. Keys are the path names reported
// by `systemd-path` (e.g. "systemd", "systemd-search-system-unit",
// "user-configuration"), values are the absolute directory paths.
type Info struct {
	Paths map[string]string `json:"paths"`
}

// Collector is the public interface every systemd_paths variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "systemd_paths" }
func (base) Category() string       { return collector.CategoryLinux }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the systemd_paths variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// parseSystemdPath parses `systemd-path` output. Each line is
// "name: /absolute/path". Matches Ohai's split-on-":" approach.
func parseSystemdPath(
	output string,
) *Info {
	info := &Info{Paths: make(map[string]string)}
	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := sc.Text()
		idx := strings.Index(line, ": ")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+2:])
		if key == "" {
			continue
		}
		info.Paths[key] = val
	}
	return info
}
