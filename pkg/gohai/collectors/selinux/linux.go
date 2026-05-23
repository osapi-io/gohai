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

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects SELinux status on Linux hosts. FS provides file
// access (production: real OS FS; tests: avfs memfs). Exec runs
// sestatus (production: real executor; tests: gomock mock).
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the real OS filesystem and
// the production Executor.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: executor.New(),
	}
}

// Collect gathers SELinux status. If sestatus is unavailable (not
// installed or returns an error), the collector falls back to
// /etc/selinux/config for the config mode. This matches Ohai's
// linux/selinux.rb which also uses sestatus as its primary source.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}

	// Primary: read /etc/selinux/config for the configured mode and
	// policy type. If the file is absent, SELinux is disabled or not
	// installed.
	configMode, policyName := parseConfigFile(l.FS)
	if configMode == "" {
		info.Status = "disabled"
		return info, nil
	}
	info.ConfigMode = configMode
	info.LoadedPolicyName = policyName

	if configMode == "disabled" {
		info.Status = "disabled"
		return info, nil
	}

	// Secondary: sestatus for runtime mode and version numbers.
	if l.Exec != nil {
		out, err := l.Exec.Execute(ctx, "sestatus")
		if err == nil {
			parseSestatus(out, info)
		}
	}

	// If sestatus didn't set status, derive it from the config mode.
	if info.Status == "" {
		info.Status = "enabled"
	}

	return info, nil
}
