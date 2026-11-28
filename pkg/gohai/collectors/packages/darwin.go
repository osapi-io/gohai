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
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// brewPaths are the paths where Homebrew installs brew: Apple silicon,
// then Intel. A launchd daemon has neither on its PATH.
var brewPaths = []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}

// Darwin collects Homebrew-installed packages on macOS hosts via
// `brew list --versions`. Embeds base for static collector metadata.
type Darwin struct {
	base

	FS   avfs.VFS
	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to the host.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm(), Exec: source.Commands()}
}

// Collect runs `brew list --versions` and parses its output. Returns
// an empty list (not an error) if Homebrew is absent, and the error if
// brew fails. With no Exec there is no fact.
//
// brew is the first of brewPaths that exists, else brew from the PATH.
// brew refuses to run as root, so a root gohai runs it as the owner of
// the brew executable, as Ohai does, with sudo -u.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	name, args := d.brewCommand(ctx)
	out, err := d.Exec.Execute(ctx, name, args...)
	if err != nil {
		if collector.Absent(err) {
			return &Info{Packages: []Package{}}, nil
		}
		return nil, fmt.Errorf("brew list: %w", err)
	}

	return parseBrew(string(out)), nil
}

// brewCommand returns the command line that lists the brew packages.
func (d *Darwin) brewCommand(
	ctx context.Context,
) (string, []string) {
	list := []string{"list", "--versions"}
	brew := d.findBrew()
	if brew == "brew" || d.uid(ctx, "id", "-u") != 0 {
		return brew, list
	}
	owner := d.uid(ctx, "stat", "-L", "-f", "%u", brew)
	if owner <= 0 {
		return brew, list
	}

	return "sudo", append([]string{"-u", "#" + strconv.Itoa(owner), "-H", brew}, list...)
}

// findBrew returns the first of brewPaths that exists, or brew, which the
// PATH finds.
func (d *Darwin) findBrew() string {
	if d.FS != nil {
		for _, path := range brewPaths {
			if _, err := d.FS.Stat(path); err == nil {
				return path
			}
		}
	}

	return "brew"
}

// uid runs a command that prints a user ID and returns it. It returns -1
// when the command fails or prints no number.
func (d *Darwin) uid(
	ctx context.Context,
	name string,
	args ...string,
) int {
	out, err := d.Exec.Execute(ctx, name, args...)
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return -1
	}

	return n
}

// parseBrew parses `brew list --versions` output: one line for each keg,
// "<name> <version> [<version> ...]". Version is the newest of the
// installed versions, and Versions lists each of them.
func parseBrew(
	out string,
) *Info {
	info := &Info{Packages: []Package{}}
	for rawLine := range strings.Lines(out) {
		parts := strings.Fields(strings.TrimRight(rawLine, "\r\n"))
		if len(parts) < 2 {
			continue
		}
		versions := parts[1:]
		newest := versions[0]
		for _, v := range versions[1:] {
			if compareVersions(v, newest) > 0 {
				newest = v
			}
		}
		info.Packages = append(info.Packages, Package{
			Name:           parts[0],
			Version:        newest,
			Versions:       versions,
			PackageManager: "brew",
		})
	}

	return info
}

// compareVersions compares two Homebrew versions, as Homebrew's Version
// does in outline: each version splits into runs of digits and runs of
// letters, and the runs compare in order. Digit runs compare as numbers,
// and a digit run is newer than a letter run. When one version is a
// prefix of the other, the longer one is newer, unless its next run is
// letters, which marks a pre-release such as 1.2.0rc1. It returns -1, 0
// or 1.
func compareVersions(
	a string,
	b string,
) int {
	ra, rb := versionRuns(a), versionRuns(b)
	for i := 0; i < len(ra) && i < len(rb); i++ {
		if c := compareRun(ra[i], rb[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(ra) > len(rb):
		return tailOrder(ra[len(rb)])
	case len(ra) < len(rb):
		return -tailOrder(rb[len(ra)])
	}

	return 0
}

// tailOrder is the order of a version that has run left over against a
// version that ends: newer for digits, older for a pre-release of letters.
func tailOrder(
	run string,
) int {
	if isDigits(run) {
		return 1
	}

	return -1
}

// compareRun compares two runs of a version.
func compareRun(
	a string,
	b string,
) int {
	da, db := isDigits(a), isDigits(b)
	switch {
	case da && db:
		na, _ := strconv.ParseUint(a, 10, 64)
		nb, _ := strconv.ParseUint(b, 10, 64)
		switch {
		case na > nb:
			return 1
		case na < nb:
			return -1
		}
		return 0
	case da:
		return 1
	case db:
		return -1
	}

	return strings.Compare(a, b)
}

// versionRuns splits a version into its runs of digits and of letters,
// and drops the separators, such as dots, dashes and underscores.
func versionRuns(
	v string,
) []string {
	var runs []string
	var cur strings.Builder
	digits := false
	flush := func() {
		if cur.Len() > 0 {
			runs = append(runs, cur.String())
			cur.Reset()
		}
	}
	for _, r := range v {
		switch {
		case unicode.IsDigit(r):
			if !digits {
				flush()
			}
			digits = true
			cur.WriteRune(r)
		case unicode.IsLetter(r):
			if digits {
				flush()
			}
			digits = false
			cur.WriteRune(r)
		default:
			flush()
		}
	}
	flush()

	return runs
}

// isDigits reports whether run is a run of digits.
func isDigits(
	run string,
) bool {
	return run != "" && unicode.IsDigit(rune(run[0]))
}
