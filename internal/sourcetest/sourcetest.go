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

// Package sourcetest builds the sources a collector test reads: a
// filesystem that holds given files, and a runner that answers given
// command lines.
package sourcetest

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
)

// FS is an in-memory filesystem that holds files, by path.
func FS(
	t testing.TB,
	files map[string]string,
) avfs.VFS {
	t.Helper()
	fs := memfs.New()
	for path, content := range files {
		if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := fs.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return fs
}

// Runner answers each command line, its name and arguments joined by
// spaces, with Out, or fails it with Err. A line it does not know fails,
// as a program that is not installed does.
type Runner struct {
	Out map[string]string
	Err map[string]error
}

// Execute is the answer for name and args.
func (r Runner) Execute(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	line := strings.Join(append([]string{name}, args...), " ")
	if err, ok := r.Err[line]; ok {
		return []byte(r.Out[line]), err
	}
	if out, ok := r.Out[line]; ok {
		return []byte(out), nil
	}

	return nil, fmt.Errorf("exec %s: executable file not found", name)
}

// Links adds symbolic links to fs, each path to its target, as sysfs
// links a device to its driver, a link's directory before the link. It
// returns fs.
func Links(
	t testing.TB,
	fs avfs.VFS,
	links map[string]string,
) avfs.VFS {
	t.Helper()
	paths := make([]string, 0, len(links))
	for path := range links {
		paths = append(paths, path)
	}
	// A link inside a linked directory comes after the directory's link.
	sort.Strings(paths)
	for _, path := range paths {
		target := links[path]
		if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := fs.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}

	return fs
}
