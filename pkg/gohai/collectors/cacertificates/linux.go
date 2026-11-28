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

package cacertificates

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

// Dir is a directory of CAs and the source of the CAs in it.
type Dir struct {
	Path   string
	Source string
}

// Layout is where a family keeps its CAs: the directories, the file that
// turns system CAs off, if the family has one, and the p11-kit blocklists.
type Layout struct {
	Dirs []Dir
	// Conf is the file whose "!name" lines turn a system CA off, or "".
	Conf string
	// Blocklists are the p11-kit directories whose certificates the host
	// distrusts. A CA in one of them is not enabled, and a file in one of
	// them is not a CA that the host trusts.
	Blocklists []string
}

// The layouts of the Linux families.
var (
	debianLayout = Layout{
		Dirs: []Dir{
			{Path: "/usr/share/ca-certificates", Source: SourceSystem},
			{Path: "/usr/local/share/ca-certificates", Source: SourceLocal},
		},
		Conf: "/etc/ca-certificates.conf",
	}
	rhelLayout = Layout{
		Dirs: []Dir{
			{Path: "/usr/share/pki/ca-trust-source", Source: SourceSystem},
			{Path: "/etc/pki/ca-trust/source/anchors", Source: SourceLocal},
		},
		Blocklists: p11kitBlocklists(
			"/usr/share/pki/ca-trust-source",
			"/etc/pki/ca-trust/source",
		),
	}
	suseLayout = Layout{
		Dirs: []Dir{
			{Path: "/usr/share/pki/trust", Source: SourceSystem},
			{Path: "/etc/pki/trust/anchors", Source: SourceLocal},
		},
		Blocklists: p11kitBlocklists("/usr/share/pki/trust", "/etc/pki/trust"),
	}
	archLayout = Layout{
		Dirs: []Dir{
			{Path: "/usr/share/ca-certificates/trust-source", Source: SourceSystem},
			{Path: "/etc/ca-certificates/trust-source/anchors", Source: SourceLocal},
		},
		Blocklists: p11kitBlocklists(
			"/usr/share/ca-certificates/trust-source",
			"/etc/ca-certificates/trust-source",
		),
	}
)

// p11kitBlocklists are the blocklist directories of the p11-kit trust
// sources roots: "blocklist", and "blacklist", the name that p11-kit
// still reads from older hosts.
func p11kitBlocklists(
	roots ...string,
) []string {
	out := make([]string, 0, 2*len(roots))
	for _, r := range roots {
		out = append(out, r+"/blocklist", r+"/blacklist")
	}

	return out
}

// LayoutFor is where the family keeps its CAs. Debian and Alpine keep them
// in /usr/share/ca-certificates, SUSE and Arch in their own trees, and the
// Red Hat families, as each other, in /usr/share/pki/ca-trust-source.
func LayoutFor(
	family string,
) Layout {
	switch family {
	case "debian", "alpine":
		return debianLayout
	case "suse":
		return suseLayout
	case "arch":
		return archLayout
	default:
		return rhelLayout
	}
}

// Linux collects the CAs of a Linux host from the files of its family.
type Linux struct {
	base

	FS     avfs.VFS
	Layout Layout
}

// NewLinux returns a Linux variant on the real filesystem, with the Red Hat
// layout until NewWith sets the host's.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Layout: rhelLayout}
}

// Collect reads each CA file under the layout's directories. A directory
// or a file that the host does not have has no CAs. One that exists and
// cannot be read is an error, so a host does not look as if it has fewer
// CAs. A CA that a blocklist or a p11-kit bundle distrusts is not enabled.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	off, err := l.disabled()
	if err != nil {
		return nil, err
	}
	distrusted, err := l.blocklisted()
	if err != nil {
		return nil, err
	}
	info := &Info{Certificates: []Certificate{}}
	for _, d := range l.Layout.Dirs {
		werr := l.FS.WalkDir(d.Path, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return walkErr(p, err)
			}
			if e.IsDir() {
				if slices.Contains(l.Layout.Blocklists, p) {
					return fs.SkipDir
				}

				return nil
			}
			if !caFile(p) {
				return nil
			}
			data, rerr := l.FS.ReadFile(p)
			if collector.Absent(rerr) {
				return nil
			}
			if rerr != nil {
				return fmt.Errorf("read %s: %w", p, rerr)
			}
			info.Certificates = append(info.Certificates, entries(d, p, data, off)...)
			if path.Ext(p) == ".p11-kit" {
				for _, c := range p11kitDistrusted(data) {
					distrusted[fingerprint(c)] = true
				}
			}

			return nil
		})
		if werr != nil {
			return nil, werr
		}
	}
	for i, c := range info.Certificates {
		if distrusted[c.Fingerprint] {
			info.Certificates[i].Enabled = false
		}
	}

	return info, nil
}

// blocklisted are the fingerprints of the certificates in the layout's
// blocklists. A blocklist the host lacks distrusts nothing. One that
// exists and cannot be read is an error.
func (l *Linux) blocklisted() (map[string]bool, error) {
	out := map[string]bool{}
	for _, dir := range l.Layout.Blocklists {
		werr := l.FS.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return walkErr(p, err)
			}
			if e.IsDir() || !caFile(p) {
				return nil
			}
			data, rerr := l.FS.ReadFile(p)
			if collector.Absent(rerr) {
				return nil
			}
			if rerr != nil {
				return fmt.Errorf("read %s: %w", p, rerr)
			}
			for _, c := range certs(data) {
				out[fingerprint(c)] = true
			}

			return nil
		})
		if werr != nil {
			return nil, werr
		}
	}

	return out, nil
}

// walkErr is what a WalkDir function returns for the error of the path
// p: a path the host lacks is skipped, and any other error stops the walk.
func walkErr(
	p string,
	err error,
) error {
	if collector.Absent(err) {
		return fs.SkipDir
	}

	return fmt.Errorf("read %s: %w", p, err)
}

// p11kitDistrusted are the certificates of the objects of a p11-kit file
// that have the attribute "x-distrusted: true", such as Fedora's
// "Explicitly Distrust DigiNotar Root CA".
func p11kitDistrusted(
	data []byte,
) []*x509.Certificate {
	var out []*x509.Certificate
	for _, obj := range strings.Split(string(data), "[p11-kit-object-v1]") {
		attrs, _, _ := strings.Cut(obj, "-----BEGIN")
		for _, line := range strings.Split(attrs, "\n") {
			k, v, ok := strings.Cut(line, ":")
			if ok && strings.TrimSpace(k) == "x-distrusted" && strings.TrimSpace(v) == "true" {
				out = append(out, certs([]byte(obj))...)

				break
			}
		}
	}

	return out
}

// disabled are the names of the system CAs that the layout's conf file
// turns off. A conf file the host lacks turns off nothing. One that exists
// and cannot be read is an error.
func (l *Linux) disabled() (map[string]bool, error) {
	off := map[string]bool{}
	if l.Layout.Conf == "" {
		return off, nil
	}
	data, err := l.FS.ReadFile(l.Layout.Conf)
	if collector.Absent(err) {
		return off, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", l.Layout.Conf, err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if name, ok := strings.CutPrefix(line, "!"); ok {
			off[strings.TrimSuffix(name, ".crt")] = true
		}
	}

	return off, nil
}

// caFile reports whether p is a file that holds CAs.
func caFile(
	p string,
) bool {
	return slices.Contains([]string{".crt", ".pem", ".cer", ".p11-kit"}, path.Ext(p))
}

// entries are the CAs of the file p in d. One CA in a file is named by the
// file; each of many is named by its subject.
func entries(
	d Dir,
	p string,
	data []byte,
	off map[string]bool,
) []Certificate {
	name := strings.TrimSuffix(strings.TrimPrefix(p, d.Path+"/"), path.Ext(p))
	base := Certificate{
		Name:    name,
		Path:    p,
		Source:  d.Source,
		Enabled: d.Source == SourceLocal || !off[name],
	}
	cs := certs(data)
	switch len(cs) {
	case 0:
		return []Certificate{base}
	case 1:
		return []Certificate{describe(base, cs[0])}
	}
	out := make([]Certificate, 0, len(cs))
	for _, c := range cs {
		e := base
		e.Name = nameOf(c)
		out = append(out, describe(e, c))
	}

	return out
}
