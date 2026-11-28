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
	"errors"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

// Linux collects IPC kernel parameters from /proc/sys/kernel/. FS is
// the virtual filesystem the collector reads from. Production uses the
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
// an Info struct. Missing files yield empty strings for their fields,
// not an error, since not all kernels expose all parameters. A file that
// is there and can't be read is an error.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	r := &sysctlReader{fs: l.FS}
	info := &Info{
		Sem: parseSem(r.read("/proc/sys/kernel/sem")),
		Msg: MsgLimits{
			MSGMNB: r.read("/proc/sys/kernel/msgmnb"),
			MSGMNI: r.read("/proc/sys/kernel/msgmni"),
			MSGMAX: r.read("/proc/sys/kernel/msgmax"),
		},
		Shm: ShmLimits{
			SHMALL: r.read("/proc/sys/kernel/shmall"),
			SHMMAX: r.read("/proc/sys/kernel/shmmax"),
			SHMMNI: r.read("/proc/sys/kernel/shmmni"),
		},
	}
	if err := errors.Join(r.errs...); err != nil {
		return nil, err
	}
	return info, nil
}
