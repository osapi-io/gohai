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

// Package rootgroup reports the primary group name for the root user.
package rootgroup

import (
	"fmt"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds the root group data.
type Info struct {
	Name string `json:"name"` // primary group name for root — typically "root" on Linux, "wheel" on macOS/BSD
}

// Collector is the public interface every rootgroup variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields common to every OS variant.
type base struct{}

// Name returns "root_group".
func (base) Name() string { return "root_group" }

// Category returns "system".
func (base) Category() collector.Category { return collector.CategorySystem }

// DefaultEnabled returns true — root_group is on by default.
func (base) DefaultEnabled() bool { return true }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the root_group collector variant appropriate to the host.
// The files are the same on Linux and macOS, but the expected result
// differs (Linux → "root", macOS → "wheel").
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

// resolveRootGroup is the name of root's primary group: the gid of root
// in /etc/passwd, and the name of that gid in /etc/group. Ohai asks
// Etc.getgrgid(Etc.getpwnam("root").gid), which reads the same files on
// Linux, and on macOS finds root and wheel there too.
func resolveRootGroup(
	fs avfs.VFS,
) (*Info, error) {
	gid, err := field(fs, "/etc/passwd", 0, "root", 3)
	if err != nil {
		return nil, fmt.Errorf("lookup user root: %w", err)
	}
	name, err := field(fs, "/etc/group", 2, gid, 0)
	if err != nil {
		return nil, fmt.Errorf("lookup gid %s: %w", gid, err)
	}

	return &Info{Name: name}, nil
}

// field is field want of the first line of the colon-separated file path
// whose field key is value.
func field(
	fs avfs.VFS,
	path string,
	key int,
	value string,
	want int,
) (string, error) {
	b, err := fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, ":")
		if len(f) > max(key, want) && f[key] == value {
			return f[want], nil
		}
	}

	return "", fmt.Errorf("%s has no %q", path, value)
}
