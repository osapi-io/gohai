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

// Package sysctl collects kernel parameters by running `sysctl -a`.
// Matches Ohai's linux/sysctl plugin: runs `sysctl -a`, splits each
// output line on "=" and stores key→value pairs. DefaultEnabled is
// false — the full sysctl table is large and most consumers don't need
// it.
package sysctl

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the complete sysctl parameter table.
type Info struct {
	Params map[string]string `json:"params"`
}

// Collector is the public interface every sysctl variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "sysctl" }
func (base) Category() string       { return collector.CategoryLinux }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the sysctl variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// parseSysctl parses `sysctl -a` output (one "key = value" or
// "key: value" pair per line). Linux emits "key = value"; macOS emits
// "key: value". We try ": " first — on Linux lines with " = " the key
// never contains ": ", and on macOS values may contain " = " which
// would corrupt a key if we split on " = " first. Lines with neither
// separator are skipped.
func parseSysctl(
	output string,
) *Info {
	info := &Info{Params: make(map[string]string)}
	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := sc.Text()
		// Try ": " first (macOS native; also present in some Linux lines).
		// Then fall back to " = " (Linux standard format).
		var key, val string
		if idx := strings.Index(line, ": "); idx >= 0 {
			key = strings.TrimSpace(line[:idx])
			val = strings.TrimSpace(line[idx+2:])
		} else if idx := strings.Index(line, " = "); idx >= 0 {
			key = strings.TrimSpace(line[:idx])
			val = strings.TrimSpace(line[idx+3:])
		} else {
			continue
		}
		if key == "" {
			continue
		}
		info.Params[key] = val
	}
	return info
}
