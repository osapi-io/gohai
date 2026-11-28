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

package selinux

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	osexec "os/exec"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects SELinux status on Linux hosts. FS provides file
// access (production: real OS FS; tests: avfs memfs). Exec runs
// sestatus (production: real executor; tests: gomock mock).
type Linux struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewLinux returns a Linux variant wired to the real OS filesystem and
// the production Executor.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: source.Commands(),
	}
}

// enforcePath is selinuxfs's enforcing switch: present when the running
// kernel has SELinux on, with "1" for enforcing and "0" for permissive.
const enforcePath = "/sys/fs/selinux/enforce"

// Collect gathers SELinux status. The running kernel decides Status and
// CurrentMode: SELinux is enabled when selinuxfs has its enforce file,
// whatever /etc/selinux/config says, and disabled when it has none.
// /etc/selinux/config gives the configured mode and the policy, and
// sestatus, when it is installed, gives the rest. A sestatus that is
// installed and fails is an error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{Status: "disabled"}
	info.ConfigMode, info.LoadedPolicyName = parseConfigFile(l.FS)

	enforce, err := l.FS.ReadFile(enforcePath)
	switch {
	case err == nil:
		info.Status = "enabled"
		info.CurrentMode = "permissive"
		if strings.TrimSpace(string(enforce)) == "1" {
			info.CurrentMode = "enforcing"
		}
	case errors.Is(err, fs.ErrNotExist):
		return info, nil
	default:
		return nil, fmt.Errorf("read %s: %w", enforcePath, err)
	}

	if l.Exec != nil {
		out, err := l.Exec.Execute(ctx, "sestatus")
		switch {
		case err == nil:
			parseSestatus(out, info)
		case !errors.Is(err, osexec.ErrNotFound):
			return nil, fmt.Errorf("sestatus: %w", err)
		}
	}

	return info, nil
}
