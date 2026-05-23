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
	"crypto/md5" //nolint:gosec // MD5 is standard for SSH key fingerprints (RFC 4716)
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// sshDir is the standard directory containing SSH host public keys.
const sshDir = "/etc/ssh"

// hostKeyGlob lists the well-known host public key file names in the
// order Ohai checks them: rsa, ecdsa, ed25519. DSA keys are omitted —
// OpenSSH deprecated DSA keys in 7.0 (2015) and removed them in 9.8.
var hostKeyFiles = []string{
	sshDir + "/ssh_host_rsa_key.pub",
	sshDir + "/ssh_host_ecdsa_key.pub",
	sshDir + "/ssh_host_ed25519_key.pub",
}

// Info holds the SSH host keys found on this host.
type Info struct {
	Keys []HostKey `json:"keys"`
}

// HostKey describes a single SSH host public key.
type HostKey struct {
	// Type is the key algorithm as reported in the public key file
	// (e.g. "ssh-rsa", "ecdsa-sha2-nistp256", "ssh-ed25519").
	Type string `json:"type"`
	// FingerprintSHA256 is the SHA-256 fingerprint in OpenSSH format
	// ("SHA256:<base64>"), matching `ssh-keygen -l -E sha256`.
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	// FingerprintMD5 is the MD5 fingerprint in colon-hex format
	// ("xx:xx:..."), matching `ssh-keygen -l -E md5`.
	FingerprintMD5 string `json:"fingerprint_md5"`
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
func (base) Category() string { return collector.CategorySecurity }

// DefaultEnabled returns false — host key collection is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the ssh collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// collectHostKeys reads the well-known SSH host public key files from
// fs and returns a slice of HostKey structs. Files that don't exist are
// silently skipped — a host without a particular key type is normal
// (e.g. RSA disabled in the sshd config). Shared by Linux and Darwin
// because /etc/ssh is the canonical path on both platforms.
func collectHostKeys(
	fs avfs.VFS,
) ([]HostKey, error) {
	keys := []HostKey{}
	for _, path := range hostKeyFiles {
		b, err := fs.ReadFile(path)
		if err != nil {
			// Missing file is not an error — key type simply not present.
			continue
		}
		key, err := parseHostKeyPub(b)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
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
	// Strip any trailing comment / options — split on whitespace, take
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

	sha256sum := sha256.Sum256(raw) //nolint:gosec
	fingerprintSHA256 := "SHA256:" + base64.StdEncoding.EncodeToString(sha256sum[:])
	fingerprintSHA256 = strings.TrimRight(fingerprintSHA256, "=")

	md5sum := md5.Sum(raw) //nolint:gosec
	hexParts := make([]string, len(md5sum))
	for i, b := range md5sum {
		hexParts[i] = fmt.Sprintf("%02x", b)
	}
	fingerprintMD5 := strings.Join(hexParts, ":")

	keyLength := deriveKeyLength(keyType, raw)

	return HostKey{
		Type:              keyType,
		FingerprintSHA256: fingerprintSHA256,
		FingerprintMD5:    fingerprintMD5,
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
