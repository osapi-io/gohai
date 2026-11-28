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
	"fmt"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin collects platform identification on macOS. The name and the
// family are darwin, and it runs `sw_vers` through the Executor and
// parses:
//
//   - ProductVersion → Version (e.g., "14.4.1").
//   - ProductVersionExtra → VersionExtra (Apple Rapid Security
//     Response patch suffix, e.g., "(a)"; empty when no RSR is
//     applied).
//   - BuildVersion → Build (e.g., "23E224").
//
// When sw_vers fails, the version and the build are empty.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns platform Info.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	arch, err := darwinArch(ctx, d.Exec)
	if err != nil {
		return nil, err
	}
	info := &Info{
		OS: family.Darwin, Name: family.Darwin, Family: family.Darwin,
		CPUArchitecture: arch,
	}
	if d.Exec != nil {
		if err := applySwVers(ctx, d.Exec, info); err != nil {
			return nil, err
		}
	}
	return info, nil
}

// applySwVers parses `sw_vers` output (key: value lines) and
// supplements info with Build + VersionExtra. A host with no sw_vers
// leaves them empty; a sw_vers that runs and fails is an error.
func applySwVers(
	ctx context.Context,
	exec source.Runner,
	info *Info,
) error {
	out, err := exec.Execute(ctx, "sw_vers")
	if collector.Absent(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("sw_vers: %w", err)
	}
	for rawLine := range strings.Lines(string(out)) {
		line := strings.TrimRight(rawLine, "\r\n")
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		switch key {
		case "ProductVersion":
			info.Version = val
		case "BuildVersion":
			if val != "" {
				info.Build = val
			}
		case "ProductVersionExtra":
			if val != "" {
				info.VersionExtra = val
			}
		}
	}
	return nil
}
