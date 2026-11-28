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

// Package readlog records what a collector reads from a host: each file and
// directory it opens or looks at, and each command it runs. Sources gives a
// collector a host with nothing on it, so a collector reads what it would
// try first on a real host, and the log says what that was.
package readlog

import (
	"context"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"

	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Log is what one collector read.
type Log struct {
	mu       sync.Mutex
	files    map[string]bool
	commands map[string]bool
}

// Files are the paths that the collector read or looked at, sorted.
func (l *Log) Files() []string {
	return l.sorted(l.files)
}

// Commands are the commands that the collector ran, each as its program
// and its arguments, sorted.
func (l *Log) Commands() []string {
	return l.sorted(l.commands)
}

func (l *Log) sorted(
	m map[string]bool,
) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)

	return out
}

func (l *Log) file(
	name string,
) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.files[name] = true
}

// Sources is an empty host of family, and the log of what is read from it.
func Sources(
	family string,
) (source.Sources, *Log) {
	l := &Log{files: map[string]bool{}, commands: map[string]bool{}}

	return source.Sources{
		FS:       &vfs{VFS: memfs.New(), log: l},
		Exec:     runner{log: l},
		Now:      func() time.Time { return time.Unix(0, 0).UTC() },
		Family:   family,
		Resolver: resolver{},
	}, l
}

// vfs is an empty file system that logs each path that is read.
type vfs struct {
	avfs.VFS
	log *Log
}

func (v *vfs) Open(
	name string,
) (avfs.File, error) {
	v.log.file(name)

	return v.VFS.Open(name)
}

func (v *vfs) OpenFile(
	name string,
	flag int,
	perm fs.FileMode,
) (avfs.File, error) {
	v.log.file(name)

	return v.VFS.OpenFile(name, flag, perm)
}

func (v *vfs) ReadFile(
	name string,
) ([]byte, error) {
	v.log.file(name)

	return v.VFS.ReadFile(name)
}

func (v *vfs) ReadDir(
	name string,
) ([]fs.DirEntry, error) {
	v.log.file(name)

	return v.VFS.ReadDir(name)
}

func (v *vfs) Stat(
	name string,
) (fs.FileInfo, error) {
	v.log.file(name)

	return v.VFS.Stat(name)
}

func (v *vfs) Lstat(
	name string,
) (fs.FileInfo, error) {
	v.log.file(name)

	return v.VFS.Lstat(name)
}

func (v *vfs) Readlink(
	name string,
) (string, error) {
	v.log.file(name)

	return v.VFS.Readlink(name)
}

func (v *vfs) EvalSymlinks(
	path string,
) (string, error) {
	v.log.file(path)

	return v.VFS.EvalSymlinks(path)
}

func (v *vfs) Glob(
	pattern string,
) ([]string, error) {
	v.log.file(pattern)

	return v.VFS.Glob(pattern)
}

func (v *vfs) WalkDir(
	root string,
	fn fs.WalkDirFunc,
) error {
	v.log.file(root)

	return v.VFS.WalkDir(root, fn)
}

// runner logs each command, and answers with no output.
type runner struct {
	log *Log
}

func (r runner) Execute(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	r.log.mu.Lock()
	defer r.log.mu.Unlock()
	r.log.commands[strings.TrimSpace(name+" "+strings.Join(args, " "))] = true

	return nil, nil
}

// resolver resolves nothing.
type resolver struct{}

func (resolver) LookupHost(
	context.Context,
	string,
) ([]string, error) {
	return nil, nil
}

func (resolver) LookupAddr(
	context.Context,
	string,
) ([]string, error) {
	return nil, nil
}
