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

package fips

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
)

const (
	fipsPath   = "/proc/sys/crypto/fips_enabled"
	policyPath = "/etc/crypto-policies/config"
)

// Linux collects FIPS mode on Linux hosts. Reads the kernel flag
// directly from /proc (equivalent to Ohai's OpenSSL.fips_mode call on
// Linux) and additionally probes /etc/crypto-policies/config for the
// 140-3 user-space policy signal Ohai doesn't capture.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect reads both the kernel flag and (if present) the crypto-policies
// file. Missing policy file is expected on non-RHEL/Fedora hosts and
// omits the Policy field rather than erroring.
func (l *Linux) Collect(
	_ context.Context,
) (any, error) {
	info := &Info{}

	kernel, err := readKernelFlag(l.FS)
	if err != nil {
		return nil, err
	}
	info.Kernel = kernel

	info.Policy = readCryptoPolicy(l.FS)

	return info, nil
}

func readKernelFlag(
	fs avfs.VFS,
) (Kernel, error) {
	b, err := fs.ReadFile(fipsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Kernel{}, nil
		}
		return Kernel{}, fmt.Errorf("read %s: %w", fipsPath, err)
	}
	return Kernel{Enabled: strings.TrimSpace(string(b)) == "1"}, nil
}

// readCryptoPolicy returns nil on hosts without /etc/crypto-policies
// (Debian/Ubuntu, older RHEL, Alpine, etc.). Missing or unreadable
// file → nil (we'd rather omit the field than fail the whole
// collector).
func readCryptoPolicy(
	fs avfs.VFS,
) *Policy {
	b, err := fs.ReadFile(policyPath)
	if err != nil {
		return nil
	}
	return parsePolicy(string(b))
}

func parsePolicy(
	raw string,
) *Policy {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return &Policy{
			Name:          line,
			FIPSEffective: strings.HasPrefix(line, "FIPS"),
		}
	}
	return nil
}
