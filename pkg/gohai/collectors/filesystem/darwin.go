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

package filesystem

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// mountLineRE is a line of macOS mount: "/dev/disk3s1s1 on / (apfs,
// sealed, local)". The device and the mountpoint can hold spaces.
var mountLineRE = regexp.MustCompile(`^(.*) on (.*) \(([^,)]+)(?:, ([^)]*))?\)$`)

// Darwin collects mounted filesystems on macOS: the mount table from
// mount, and the sizes and inodes from df -ki.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's mount and df.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns filesystem Info. With no Exec it gives no fact, as
// mount and df give every value.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	out, err := d.Exec.Execute(ctx, "mount")
	if err != nil {
		return nil, fmt.Errorf("mount: %w", err)
	}
	mounts := []Mount{}
	for _, line := range strings.Split(string(out), "\n") {
		m := mountLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		mount := Mount{Device: m[1], Mountpoint: m[2], Type: m[3]}
		if m[4] != "" {
			mount.Opts = strings.Split(m[4], ", ")
		}
		mounts = append(mounts, mount)
	}
	rows := df(ctx, d.Exec, mounts, 7, "-ki")
	for i := range mounts {
		if r, ok := rows[mounts[i].Mountpoint]; ok {
			usage(&mounts[i], r[0], r[1], r[2])
			inodes(&mounts[i], r[4], r[5])
		}
	}

	return &Info{Mounts: mounts}, nil
}
