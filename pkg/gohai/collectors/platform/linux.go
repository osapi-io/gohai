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

package platform

import (
	"context"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects platform identification on Linux. It reads the ID and
// VERSION_ID of /etc/os-release, as Ohai does, and then layers two
// extensions:
//
//   - /etc/redhat-release (or /etc/debian_version) supplements
//     `Version` when os-release has a major-only or empty value.
//   - When os-release gives no name at all (legacy / appliance
//     distros without /etc/os-release), we cascade through the Ohai
//     legacy_platform_detection chain (redhat-release, SuSE-release,
//     f5-release, system-release, debian_version, arch-release,
//     gentoo-release, slackware-version, enterprise-release,
//     exherbo-release).
type Linux struct {
	base

	FS avfs.VFS
	// Exec runs `uname -m` for the host's architecture. When it is nil or
	// uname fails, the architecture is the binary's.
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect returns platform Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{OS: "linux", CPUArchitecture: unameArch(ctx, l.Exec)}
	if l.FS == nil {
		return info, nil
	}
	for _, step := range []func(avfs.VFS, *Info) error{
		readOSRelease,
		applyRedhatReleaseSupplement,
		applyLegacyReleaseFallback,
	} {
		if err := step(l.FS, info); err != nil {
			return nil, err
		}
	}
	return info, nil
}
