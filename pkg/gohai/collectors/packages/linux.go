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
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Linux collects installed packages on Linux hosts. On Debian-family
// systems it uses dpkg-query; on RHEL-family systems it uses rpm.
// The database is Manager, which NewWith picks from the family. Embeds base for the
// static Name / DefaultEnabled / Dependencies methods.
type Linux struct {
	base

	Exec source.Runner
	// Manager is the package database: dpkg, rpm, or "" for a family
	// with neither. NewWith picks it from the family of its sources.
	Manager string
}

// NewLinux returns a Linux variant wired to the production Executor.
func NewLinux() *Linux {
	return &Linux{Exec: source.Commands(), Manager: "dpkg"}
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
	switch l.Manager {
	case "rpm":
		return collectRPM(ctx, l.Exec)
	case "dpkg":
		return collectDpkg(ctx, l.Exec)
	default:
		return &Info{Packages: []Package{}}, nil
	}
}

// collectDpkg queries dpkg-query on Debian/Ubuntu hosts. Format
// mirrors Ohai's packages.rb debian branch:
// ${Package}\t${Version}\t${Architecture}\t${db:Status-Status}
// Only "installed" status packages are included.
func collectDpkg(
	ctx context.Context,
	exec source.Runner,
) (*Info, error) {
	format := "${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n"
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
		p := Package{Name: name, Version: version, Architecture: arch, PackageManager: "dpkg"}
		if len(parts) >= 6 {
			p.Description = parts[4]
			kb, _ := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64)
			p.Size = kb * 1024
		}
		info.Packages = append(info.Packages, p)
	}
	return info, nil
}

// collectRPM queries rpm on RHEL/Fedora/SUSE hosts. Format mirrors
// Ohai's packages.rb rhel branch: %{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}
func collectRPM(
	ctx context.Context,
	exec source.Runner,
) (*Info, error) {
	format := "%{NAME}\\t%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n"
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
		p := Package{Name: name, Version: version, Architecture: arch, PackageManager: "rpm"}
		if len(parts) >= 5 {
			p.Description = parts[3]
			p.Size, _ = strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		}
		info.Packages = append(info.Packages, p)
	}
	return info, nil
}
