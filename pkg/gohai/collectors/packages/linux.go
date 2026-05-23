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

package packages

import (
	"bufio"
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/internal/platform"
)

// Linux collects installed packages on Linux hosts. On Debian-family
// systems it uses dpkg-query; on RHEL-family systems it uses rpm.
// Detection is based on platform.Detect() result. Embeds base for the
// static Name / DefaultEnabled / Dependencies methods.
type Linux struct {
	base

	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: executor.New()}
}

// Collect queries the installed-package database and returns the
// package list. Returns an empty list (not an error) if the query tool
// is absent or fails — mirrors Ohai's no-panic stance.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if l.Exec == nil {
		return &Info{Packages: []Package{}}, nil
	}
	switch platform.Detect() {
	case "debian":
		return collectDpkg(ctx, l.Exec)
	default:
		return collectRPM(ctx, l.Exec)
	}
}

// collectDpkg queries dpkg-query on Debian/Ubuntu hosts. Format
// mirrors Ohai's packages.rb debian branch:
// ${Package}\t${Version}\t${Architecture}\t${db:Status-Status}
// Only "installed" status packages are included.
func collectDpkg(
	ctx context.Context,
	exec executor.Executor,
) (*Info, error) {
	format := "${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\n"
	out, err := exec.Execute(ctx, "dpkg-query", "-W", "-f="+format)
	if err != nil {
		return &Info{Packages: []Package{}}, nil
	}
	info := &Info{Packages: []Package{}}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		parts := strings.Split(sc.Text(), "\t")
		if len(parts) < 4 {
			continue
		}
		name, version, arch, status := parts[0], parts[1], parts[2], strings.TrimSpace(parts[3])
		if name == "" || status != "installed" {
			continue
		}
		info.Packages = append(info.Packages, Package{
			Name:    name,
			Version: version,
			Arch:    arch,
			Source:  "dpkg",
		})
	}
	return info, nil
}

// collectRPM queries rpm on RHEL/Fedora/SUSE hosts. Format mirrors
// Ohai's packages.rb rhel branch: %{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}
func collectRPM(
	ctx context.Context,
	exec executor.Executor,
) (*Info, error) {
	format := "%{NAME}\\t%{VERSION}-%{RELEASE}\\t%{ARCH}\\n"
	out, err := exec.Execute(ctx, "rpm", "-qa", "--qf", format)
	if err != nil {
		return &Info{Packages: []Package{}}, nil
	}
	info := &Info{Packages: []Package{}}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		parts := strings.Split(sc.Text(), "\t")
		if len(parts) < 3 {
			continue
		}
		name, version, arch := parts[0], parts[1], strings.TrimSpace(parts[2])
		if name == "" {
			continue
		}
		info.Packages = append(info.Packages, Package{
			Name:    name,
			Version: version,
			Arch:    arch,
			Source:  "rpm",
		})
	}
	return info, nil
}
