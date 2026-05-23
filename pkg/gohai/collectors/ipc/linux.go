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

package ipc

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

// Linux collects IPC kernel parameters from /proc/sys/kernel/. FS is
// the virtual filesystem the collector reads from — production is the
// real OS FS via avfs/osfs; tests inject an avfs memfs with canned
// content.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect reads the IPC sysctl files under /proc/sys/kernel and returns
// an Info struct. Missing files yield empty strings for their fields —
// not an error, since not all kernels expose all parameters.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	readFn := func(path string) ([]byte, error) {
		return l.FS.ReadFile(path)
	}

	semRaw := readTrimmed(readFn, "/proc/sys/kernel/sem")

	return &Info{
		Sem: parseSem(semRaw),
		Msg: MsgLimits{
			MSGMNB: readTrimmed(readFn, "/proc/sys/kernel/msgmnb"),
			MSGMNI: readTrimmed(readFn, "/proc/sys/kernel/msgmni"),
			MSGMAX: readTrimmed(readFn, "/proc/sys/kernel/msgmax"),
		},
		Shm: ShmLimits{
			SHMALL: readTrimmed(readFn, "/proc/sys/kernel/shmall"),
			SHMMAX: readTrimmed(readFn, "/proc/sys/kernel/shmmax"),
			SHMMNI: readTrimmed(readFn, "/proc/sys/kernel/shmmni"),
		},
	}, nil
}
