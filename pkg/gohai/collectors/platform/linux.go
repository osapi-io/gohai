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
)

// Linux collects platform identification on Linux. gopsutil reads
// /etc/os-release for the modern systemd path; we then layer two
// extensions:
//
//   - /etc/redhat-release (or /etc/debian_version) supplements
//     `Version` when gopsutil reports a major-only or empty value.
//   - When gopsutil produces no name at all (legacy / appliance
//     distros without /etc/os-release), we cascade through the Ohai
//     legacy_platform_detection chain (redhat-release, SuSE-release,
//     f5-release, system-release, debian_version, arch-release,
//     gentoo-release, slackware-version, enterprise-release,
//     exherbo-release).
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns platform Info.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info, _, err := readPlatform(ctx)
	if err != nil {
		return nil, err
	}
	if l.FS != nil {
		applyRedhatReleaseSupplement(l.FS, info)
		applyLegacyReleaseFallback(l.FS, info)
	}
	applyFamilyFallback(info)
	return info, nil
}
