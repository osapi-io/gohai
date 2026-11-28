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

// Package ssh collects SSH host key fingerprints from /etc/ssh/. Each
// host key type (rsa, ecdsa, ed25519) is reported with its key type,
// fingerprint, and key length. The collector is opt-in (DefaultEnabled
// false) because reading host key material is security-sensitive.
package ssh

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	iofs "io/fs"
	"regexp"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// sshDir is the standard directory containing SSH host public keys.
const sshDir = "/etc/ssh"

// hostKeyGlob lists the well-known host public key file names in the
// order Ohai checks them: rsa, ecdsa, ed25519. DSA keys are left out, because
// OpenSSH deprecated DSA keys in 7.0 (2015) and removed them in 9.8.
var hostKeyFiles = []string{
	sshDir + "/ssh_host_rsa_key.pub",
	sshDir + "/ssh_host_ecdsa_key.pub",
	sshDir + "/ssh_host_ed25519_key.pub",
}

// Info holds the SSH host keys found on this host.
type Info struct {
	Keys []HostKey `json:"keys"`
	// AuthorizedKeys is the keys of each user's authorized_keys files, by
	// user name: the AuthorizedKeysFile of `sshd -T`, else
	// ~/.ssh/authorized_keys and ~/.ssh/authorized_keys2. A user with no
	// file is not in it.
	AuthorizedKeys map[string][]AuthorizedKey `json:"authorized_keys,omitempty"`
}

// AuthorizedKey is one line of an authorized_keys file.
type AuthorizedKey struct {
	Type              string `json:"type"`
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	Comment           string `json:"comment,omitempty"`
	// Options are the options before the key, as the file has them, such
	// as `no-pty,from="10.0.0.0/8"`. The value of a command= or an
	// environment= option is redacted, because it can hold a secret.
	Options string `json:"options,omitempty"`
	// Line is the line as the file has it, with the same redaction.
	Line string `json:"line"`
}

// HostKey describes a single SSH host public key.
type HostKey struct {
	// Type is the key algorithm as reported in the public key file
	// (e.g. "ssh-rsa", "ecdsa-sha2-nistp256", "ssh-ed25519").
	Type string `json:"type"`
	// FingerprintSHA256 is the SHA-256 fingerprint in OpenSSH format
	// ("SHA256:<base64>"), matching `ssh-keygen -l -E sha256`.
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	// KeyLength is the effective key length in bits. For RSA/DSA it is
	// the modulus bit-length; for ECDSA it is the curve size; for
	// Ed25519 it is always 256.
	KeyLength int `json:"key_length"`
}

// Collector is the public interface every ssh variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "ssh".
func (base) Name() string { return "ssh" }

// Category returns "security".
func (base) Category() collector.Category { return collector.CategorySecurity }

// DefaultEnabled returns false, because host key collection is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the ssh collector variant appropriate to the detected
// host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// collectHostKeys reads the well-known SSH host public key files from
// fs and returns a slice of HostKey structs. A file that does not exist
// or does not parse is left out: a host without a particular key type is
// normal (e.g. RSA disabled in the sshd config), and one bad file must
// not hide the other keys or the authorized keys. A file that exists and
// does not read is an error, since the public keys are readable by every
// user. Shared by Linux and Darwin because /etc/ssh is the canonical path
// on both platforms.
func collectHostKeys(
	fs avfs.VFS,
) ([]HostKey, error) {
	keys := []HostKey{}
	for _, path := range hostKeyFiles {
		b, err := fs.ReadFile(path)
		if collector.Absent(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		key, err := parseHostKeyPub(b)
		if err != nil {
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// parseHostKeyPub parses a single line from a .pub file
// ("<type> <base64-blob> [comment]") and returns a HostKey.
func parseHostKeyPub(
	content []byte,
) (HostKey, error) {
	line := strings.TrimSpace(string(content))
	// Strip any trailing comment / options: split on whitespace, take
	// the first two fields.
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return HostKey{}, fmt.Errorf("malformed public key line: %q", line)
	}
	keyType := fields[0]
	rawB64 := fields[1]

	raw, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return HostKey{}, fmt.Errorf("base64 decode: %w", err)
	}

	sha256sum := sha256.Sum256(raw)
	fingerprintSHA256 := "SHA256:" + base64.StdEncoding.EncodeToString(sha256sum[:])
	fingerprintSHA256 = strings.TrimRight(fingerprintSHA256, "=")

	keyLength := deriveKeyLength(keyType, raw)

	return HostKey{
		Type:              keyType,
		FingerprintSHA256: fingerprintSHA256,
		KeyLength:         keyLength,
	}, nil
}

// deriveKeyLength returns the effective key length in bits for the
// given key type and raw wire-format bytes.
//
// Wire format (RFC 4251 §5): each component is a length-prefixed
// big-endian uint32 followed by that many bytes. The first component
// is always the algorithm name.
//
//   - RSA: second component = public exponent (e), third = modulus (n).
//     Key length = bit-length of n.
//   - ECDSA (nistp256/384/521): second component = curve name,
//     third = Q (uncompressed point). Key length derived from curve name.
//   - Ed25519: always 256 bits.
func deriveKeyLength(
	keyType string,
	raw []byte,
) int {
	switch {
	case strings.HasPrefix(keyType, "ecdsa"):
		// Derive from curve name embedded in the key type string.
		// ecdsa-sha2-nistp256 → 256, nistp384 → 384, nistp521 → 521.
		switch {
		case strings.Contains(keyType, "nistp256"):
			return 256
		case strings.Contains(keyType, "nistp384"):
			return 384
		case strings.Contains(keyType, "nistp521"):
			return 521
		}
		return 0
	case keyType == "ssh-ed25519":
		return 256
	case keyType == "ssh-rsa":
		// Skip algorithm name field, skip exponent field, read modulus.
		blob := raw
		// Skip algorithm name (field 0).
		blob = skipWireString(blob)
		// Skip public exponent (field 1).
		blob = skipWireString(blob)
		// Read modulus (field 2).
		modBytes := readWireString(blob)
		if modBytes == nil {
			return 0
		}
		// Strip leading zero byte (sign bit padding) if present.
		for len(modBytes) > 0 && modBytes[0] == 0 {
			modBytes = modBytes[1:]
		}
		return len(modBytes) * 8
	default:
		return 0
	}
}

// skipWireString advances past one length-prefixed wire string and
// returns the remaining bytes. Returns nil on underflow.
func skipWireString(
	b []byte,
) []byte {
	if len(b) < 4 {
		return nil
	}
	n := int(binary.BigEndian.Uint32(b[:4]))
	if len(b) < 4+n {
		return nil
	}
	return b[4+n:]
}

// readWireString returns the bytes of the first length-prefixed wire
// string in b. Returns nil on underflow.
func readWireString(
	b []byte,
) []byte {
	if len(b) < 4 {
		return nil
	}
	n := int(binary.BigEndian.Uint32(b[:4]))
	if len(b) < 4+n {
		return nil
	}
	return b[4 : 4+n]
}

// home is the home directory of one user, and its uid when the source
// gives one.
type home struct {
	user string
	dir  string
	uid  string
}

// passwdHomes is the home of each user of /etc/passwd. A host with no
// /etc/passwd has no users; an /etc/passwd that does not read is an error.
func passwdHomes(
	fs avfs.VFS,
) ([]home, error) {
	b, err := fs.ReadFile("/etc/passwd")
	if collector.Absent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read /etc/passwd: %w", err)
	}

	return parsePasswd(b), nil
}

// parsePasswd is the home of each line of passwd(5) format, as
// /etc/passwd and `getent passwd` give it.
func parsePasswd(
	b []byte,
) []home {
	var out []home
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 6 {
			continue
		}
		out = append(out, home{user: f[0], dir: f[5], uid: f[2]})
	}

	return out
}

// defaultKeyFiles is sshd's AuthorizedKeysFile when it does not say.
var defaultKeyFiles = []string{".ssh/authorized_keys", ".ssh/authorized_keys2"}

// keyFiles is the AuthorizedKeysFile patterns of `sshd -T`. A host where
// sshd -T does not run, such as an agent that is not root, has the
// default. "none" is no file.
func keyFiles(
	ctx context.Context,
	exec source.Runner,
) []string {
	if exec == nil {
		return defaultKeyFiles
	}
	out, err := exec.Execute(ctx, "sshd", "-T")
	if err != nil {
		return defaultKeyFiles
	}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || !strings.EqualFold(f[0], "authorizedkeysfile") {
			continue
		}
		if len(f) == 2 && f[1] == "none" {
			return nil
		}

		return f[1:]
	}

	return defaultKeyFiles
}

// keyPath is pattern for h, with sshd's tokens: %% is %, %h the home,
// %u the user and %U the uid. A relative path is in the home. It is
// false when h has no value for a token, or no home for a relative path.
func keyPath(
	pattern string,
	h home,
) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		if pattern[i] != '%' {
			b.WriteByte(pattern[i])
			continue
		}
		if i+1 == len(pattern) {
			return "", false
		}
		i++
		var v string
		switch pattern[i] {
		case '%':
			v = "%"
		case 'h':
			v = h.dir
		case 'u':
			v = h.user
		case 'U':
			v = h.uid
		}
		if v == "" {
			return "", false
		}
		b.WriteString(v)
	}
	p := b.String()
	if strings.HasPrefix(p, "/") {
		return p, true
	}
	if h.dir == "" || h.dir == "/" {
		return "", false
	}

	return strings.TrimRight(h.dir, "/") + "/" + p, true
}

// dsclHomes is the home of each user of `dscl . -list /Users
// NFSHomeDirectory`, one "name home [home...]" line per user. The first
// home of a user is the one that it logs in to.
func dsclHomes(
	out []byte,
) []home {
	var homes []home
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		homes = append(homes, home{user: f[0], dir: f[1]})
	}

	return homes
}

// collectAuthorizedKeys is the keys of each file of patterns for each
// user of homes. A line that is not a key is left out. A file that does
// not exist is no keys. A file that the collector may not read, as each
// other user's file is to a collector that is not root, is left out, as
// the users collector leaves out /etc/shadow. Any other read failure is
// an error.
func collectAuthorizedKeys(
	fs avfs.VFS,
	homes []home,
	patterns []string,
) (map[string][]AuthorizedKey, error) {
	out := map[string][]AuthorizedKey{}
	for _, h := range homes {
		read := map[string]bool{}
		for _, pattern := range patterns {
			path, ok := keyPath(pattern, h)
			if !ok || read[path] {
				continue
			}
			read[path] = true
			keys, err := fs.ReadFile(path)
			if collector.Absent(err) || errors.Is(err, iofs.ErrPermission) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			for _, l := range strings.Split(string(keys), "\n") {
				if k, ok := parseAuthorizedKey(strings.TrimSpace(l)); ok {
					out[h.user] = append(out[h.user], k)
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

// parseAuthorizedKey is one authorized_keys line: options, then the key
// type and the key, then a comment. The key type is the first field that
// names one, so options may come before it.
func parseAuthorizedKey(
	line string,
) (AuthorizedKey, bool) {
	if line == "" || strings.HasPrefix(line, "#") {
		return AuthorizedKey{}, false
	}
	f := fields(line)
	for i := 0; i+1 < len(f); i++ {
		if !isKeyType(f[i].text) {
			continue
		}
		hk, err := parseHostKeyPub([]byte(f[i].text + " " + f[i+1].text))
		if err != nil {
			return AuthorizedKey{}, false
		}
		comment := ""
		if i+2 < len(f) {
			comment = line[f[i+2].start:]
		}

		return AuthorizedKey{
			Type:              hk.Type,
			FingerprintSHA256: hk.FingerprintSHA256,
			Comment:           comment,
			Options:           redactOptions(strings.TrimSpace(line[:f[i].start])),
			Line:              redactOptions(line[:f[i].start]) + line[f[i].start:],
		}, true
	}

	return AuthorizedKey{}, false
}

// field is one field of an authorized_keys line, and where it starts.
type field struct {
	text  string
	start int
}

// fields splits line at the spaces and tabs that are not in double
// quotes, as sshd reads the options. A space in an option value, such as
// command="a b", is then not a field of its own, and a key type in one,
// such as command="echo ssh-rsa", is not the key.
func fields(
	line string,
) []field {
	var out []field
	start, quoted := -1, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '"' && (i == 0 || line[i-1] != '\\'):
			quoted = !quoted
			if start < 0 {
				start = i
			}
		case (c == ' ' || c == '\t') && !quoted:
			if start >= 0 {
				out = append(out, field{text: line[start:i], start: start})
				start = -1
			}
		default:
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		out = append(out, field{text: line[start:], start: start})
	}

	return out
}

// secretOptionRE is a command= or environment= option with its quoted
// value, which sshd lets hold escaped quotes.
var secretOptionRE = regexp.MustCompile(`(?i)\b(command|environment)="(?:[^"\\]|\\.)*"`)

// redactOptions is options with the value of each command= and
// environment= option replaced, because a forced command or a variable
// can hold a token.
func redactOptions(
	options string,
) string {
	return secretOptionRE.ReplaceAllString(options, `$1="[redacted]"`)
}

// isKeyType reports whether s names an OpenSSH public key type.
func isKeyType(
	s string,
) bool {
	for _, p := range []string{"ssh-", "ecdsa-", "sk-"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}

	return false
}
