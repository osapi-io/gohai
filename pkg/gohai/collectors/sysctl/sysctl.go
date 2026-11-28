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
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
	"strings"

	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/osapi-io/gohai/internal/collector"
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

func (base) Name() string                 { return "sysctl" }
func (base) Category() collector.Category { return collector.CategoryLinux }
func (base) DefaultEnabled() bool         { return false }
func (base) Dependencies() []string       { return nil }

// New returns the sysctl variant for the host OS.
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

// run runs `sysctl -a` through exec and parses it with sep, the separator
// of the host's format. A host with no sysctl gives an empty table. A run
// that fails but prints part of the table, as `sysctl -a` does for a user
// who cannot read some keys, gives that part. A run that fails and prints
// nothing is an error.
func run(
	ctx context.Context,
	exec source.Runner,
	sep string,
) (*Info, error) {
	if exec == nil {
		return &Info{Params: make(map[string]string)}, nil
	}
	out, err := exec.Execute(ctx, "sysctl", "-a")
	switch {
	case err == nil, len(out) > 0:
		return parseSysctl(string(out), sep), nil
	case errors.Is(err, osexec.ErrNotFound):
		return &Info{Params: make(map[string]string)}, nil
	default:
		return nil, fmt.Errorf("sysctl -a: %w", err)
	}
}

// Separators of `sysctl -a`: Linux prints "key = value", macOS prints
// "key: value". Each host's lines split at the first of its own separator,
// so a Linux value that holds ": ", such as dev.cdrom.info's, and a macOS
// value that holds " = ", keep their text.
const (
	linuxSep  = " = "
	darwinSep = ": "
)

// parseSysctl parses `sysctl -a` output, one pair for each line, split at
// the first sep. Lines without sep, or with an empty key, are skipped.
func parseSysctl(
	output string,
	sep string,
) *Info {
	info := &Info{Params: make(map[string]string)}
	sc := bufio.NewScanner(strings.NewReader(output))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		key, val, ok := strings.Cut(sc.Text(), sep)
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		info.Params[key] = strings.TrimSpace(val)
	}
	return info
}
