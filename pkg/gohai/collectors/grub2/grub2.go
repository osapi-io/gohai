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

// Package grub2 reads the GRUB2 environment block and reports the
// key=value pairs it contains. On Darwin the collector returns nil
// gracefully — GRUB2 is a Linux/BSD bootloader. The collector is
// opt-in (DefaultEnabled false) because it requires a GRUB2 install.
package grub2

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// grubenvPaths lists the canonical grubenv locations in priority order.
// Ohai checks for grub2-editenv on PATH first, then falls back to
// directly reading the file. We read the file directly because it is
// simpler and avoids a fork — grubenv is a fixed-format binary-safe
// file, not an exec artifact.
//
//   - GRUB2 path (RHEL/Fedora): /boot/grub2/grubenv
//   - GRUB path (Debian/Ubuntu): /boot/grub/grubenv
var grubenvPaths = []string{
	"/boot/grub2/grubenv",
	"/boot/grub/grubenv",
}

// Info holds the GRUB2 environment variables.
type Info struct {
	// Environment is the map of key=value pairs from the grubenv file.
	// An empty map means the grubenv file was found but contained no
	// parseable key=value pairs. A nil map means no grubenv file was
	// found on any of the checked paths.
	Environment map[string]string `json:"environment"`
}

// Collector is the public interface every grub2 variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "grub2".
func (base) Name() string { return "grub2" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — grub2 is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the grub2 collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// readGrubenv tries each path in grubenvPaths in order and returns the
// parsed environment from the first file found. Returns nil when no
// grubenv file exists on any path.
func readGrubenv(
	fs avfs.VFS,
) map[string]string {
	for _, path := range grubenvPaths {
		b, err := fs.ReadFile(path)
		if err != nil {
			continue
		}
		return parseGrubenv(b)
	}
	return nil
}

// parseGrubenv parses a grubenv file. The format is:
//
//	# GRUB Environment Block
//	key=value
//	key2=value2
//
// Lines starting with '#' are comments and skipped. Lines without '='
// are also skipped. Values are taken verbatim — GRUB does not quote or
// escape values in the environment block.
func parseGrubenv(
	content []byte,
) map[string]string {
	env := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[key] = val
	}
	return env
}
