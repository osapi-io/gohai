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

package ssh_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	gohaisssh "github.com/osapi-io/gohai/pkg/gohai/collectors/ssh"
)

var (
	_ collector.Collector = (*gohaisssh.Linux)(nil)
	_ collector.Collector = (*gohaisssh.Darwin)(nil)
)

// malformFS returns a file with malformed base64 content.
type malformFS struct {
	avfs.VFS
}

func (m malformFS) ReadFile(
	_ string,
) ([]byte, error) {
	return []byte("ssh-rsa !!!notbase64!!! comment\n"), nil
}

// badFieldsFS returns a file with only one whitespace-separated field.
type badFieldsFS struct {
	avfs.VFS
}

func (b badFieldsFS) ReadFile(
	_ string,
) ([]byte, error) {
	return []byte("ssh-rsa\n"), nil
}

// fsWith builds a memfs with the given path→content entries. The
// /etc/ssh directory is created once regardless of the supplied paths.
func fsWith(
	t require.TestingT,
	files map[string][]byte,
) avfs.VFS {
	f := memfs.New()
	_ = f.MkdirAll("/etc/ssh", 0o755)
	for path, content := range files {
		require.NoError(t, f.WriteFile(path, content, fs.FileMode(0o644)))
	}
	return f
}

// encodeSSHPublicKey encodes a crypto public key as an OpenSSH
// authorized_keys line.
func encodeSSHPublicKey(
	t require.TestingT,
	pub interface{},
) []byte {
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	return ssh.MarshalAuthorizedKey(sshPub)
}

func generateRSAKey(
	t require.TestingT,
	bits int,
) []byte {
	k, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)
	return encodeSSHPublicKey(t, &k.PublicKey)
}

func generateECDSAKey(
	t require.TestingT,
	curve elliptic.Curve,
) []byte {
	k, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return encodeSSHPublicKey(t, &k.PublicKey)
}

func generateEd25519Key(
	t require.TestingT,
) []byte {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return encodeSSHPublicKey(t, pub)
}

// marshalWireString returns the RFC 4251 length-prefixed wire encoding
// of b.
func marshalWireString(
	b []byte,
) []byte {
	hdr := make([]byte, 4, 4+len(b))
	binary.BigEndian.PutUint32(hdr, uint32(len(b)))
	return append(hdr, b...)
}

// pubLineFromBlob wraps raw wire bytes as a fake ssh-rsa public key
// line that parseHostKeyPub can consume.
func pubLineFromBlob(
	blob []byte,
) []byte {
	return []byte("ssh-rsa " + base64.StdEncoding.EncodeToString(blob) + "\n")
}

type SSHPublicTestSuite struct {
	suite.Suite
}

func TestSSHPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SSHPublicTestSuite))
}

func (s *SSHPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(gohaisssh.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c gohaisssh.Collector) {
				_, ok := c.(*gohaisssh.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c gohaisssh.Collector) {
				_, ok := c.(*gohaisssh.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c gohaisssh.Collector) {
				_, ok := c.(*gohaisssh.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c gohaisssh.Collector) {
				_, ok := c.(*gohaisssh.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c gohaisssh.Collector) {
				_, ok := c.(*gohaisssh.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := gohaisssh.New()
			s.Equal("ssh", c.Name())
			s.Equal("security", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SSHPublicTestSuite) TestCollect() {
	// truncatedRSABlob is a valid-looking ssh-rsa public key line whose
	// wire bytes are too short to contain a modulus. deriveKeyLength
	// must return 0 without panicking. We use this for both the Linux
	// and Darwin truncated-modulus rows.
	//
	// Wire layout: algo-name(ssh-rsa) | exponent | (no modulus — blob
	// ends after exponent field). skipWireString returns nil on the
	// second call; readWireString therefore gets nil → returns 0.
	truncatedRSABlob := func() []byte {
		algoField := marshalWireString([]byte("ssh-rsa"))
		expField := marshalWireString([]byte{1, 0, 1}) // exponent 65537 only
		return append(algoField, expField...)
	}()

	tests := []struct {
		name         string
		variant      string
		setupFS      func() avfs.VFS
		validateFunc func(any, error)
	}{
		{
			name:    "linux: no key files",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc/ssh", 0o755)
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 0)
			},
		},
		{
			name:    "linux: rsa-2048 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": generateRSAKey(s.T(), 2048),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(2048, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: rsa-4096 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": generateRSAKey(s.T(), 4096),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(4096, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: truncated rsa wire blob — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": pubLineFromBlob(truncatedRSABlob),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ecdsa-p256 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_ecdsa_key.pub": generateECDSAKey(s.T(), elliptic.P256()),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ecdsa-sha2-nistp256" != "" && len(info.Keys) > 0 {
					s.Equal("ecdsa-sha2-nistp256", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(256, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ecdsa-p384 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_ecdsa_key.pub": generateECDSAKey(s.T(), elliptic.P384()),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ecdsa-sha2-nistp384" != "" && len(info.Keys) > 0 {
					s.Equal("ecdsa-sha2-nistp384", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(384, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ecdsa-p521 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_ecdsa_key.pub": generateECDSAKey(s.T(), elliptic.P521()),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ecdsa-sha2-nistp521" != "" && len(info.Keys) > 0 {
					s.Equal("ecdsa-sha2-nistp521", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(521, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ed25519 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_ed25519_key.pub": generateEd25519Key(s.T()),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-ed25519" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-ed25519", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(256, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: all three key types",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub":     generateRSAKey(s.T(), 2048),
					"/etc/ssh/ssh_host_ecdsa_key.pub":   generateECDSAKey(s.T(), elliptic.P256()),
					"/etc/ssh/ssh_host_ed25519_key.pub": generateEd25519Key(s.T()),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 3)
			},
		},
		{
			name:    "linux: malformed base64 returns error",
			variant: "linux",
			setupFS: func() avfs.VFS { return malformFS{memfs.New()} },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: too few fields returns error",
			variant: "linux",
			setupFS: func() avfs.VFS { return badFieldsFS{memfs.New()} },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: ed25519 key",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_ed25519_key.pub": generateEd25519Key(s.T()),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-ed25519" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-ed25519", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(256, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "darwin: no key files",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc/ssh", 0o755)
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 0)
			},
		},
		{
			name:    "darwin: malformed base64 returns error",
			variant: "darwin",
			setupFS: func() avfs.VFS { return malformFS{memfs.New()} },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: unknown key type — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// Build a blob whose type field is not rsa/ecdsa/ed25519 to
				// exercise the default branch in deriveKeyLength.
				unknownBlob := marshalWireString([]byte("ssh-unknown"))
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": pubLineFromBlob(unknownBlob),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: wire blob too short for length header — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// 3-byte blob: too short for even the first 4-byte length
				// header. skipWireString returns nil (len < 4 branch).
				shortBlob := []byte{0x00, 0x01, 0x02}
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": pubLineFromBlob(shortBlob),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: wire blob length field exceeds bytes — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// Length header claims 100 bytes but only 2 follow — the
				// len(b) < 4+n branch in skipWireString.
				tooShort := []byte{0, 0, 0, 100, 0x01, 0x02}
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": pubLineFromBlob(tooShort),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			// Exercise readWireString's len(b) < 4+n branch: blob has two
			// valid fields (algo + exponent) so skipWireString succeeds twice,
			// then the modulus field header claims 100 bytes but only 2 remain.
			name:    "linux: modulus length exceeds remaining bytes — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				algoField := marshalWireString([]byte("ssh-rsa"))
				expField := marshalWireString([]byte{1, 0, 1})
				// Modulus header claims 100 bytes but only 2 bytes follow.
				modHeader := []byte{0, 0, 0, 100, 0x00, 0x00}
				blob := append(append(algoField, expField...), modHeader...)
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": pubLineFromBlob(blob),
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-rsa" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-rsa", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			// Exercise deriveKeyLength default branch: a fake type that is
			// not rsa, ecdsa, or ed25519. The file field[0] is "ssh-dss"
			// (deprecated) so keyType hits the default return 0.
			name:    "linux: dsa key type — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				blob := marshalWireString([]byte("ssh-dss"))
				line := append(
					[]byte("ssh-dss "),
					[]byte(base64.StdEncoding.EncodeToString(blob)+"\n")...,
				)
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_rsa_key.pub": line,
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ssh-dss" != "" && len(info.Keys) > 0 {
					s.Equal("ssh-dss", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			// Exercise deriveKeyLength ecdsa inner-switch default: key type
			// has "ecdsa" prefix but not a recognized nistp* curve label.
			// This hits the `return 0` after the inner ecdsa switch.
			name:    "linux: ecdsa unknown curve — key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				blob := marshalWireString([]byte("ecdsa-sha2-unknown"))
				line := append(
					[]byte("ecdsa-sha2-unknown "),
					[]byte(base64.StdEncoding.EncodeToString(blob)+"\n")...,
				)
				return fsWith(s.T(), map[string][]byte{
					"/etc/ssh/ssh_host_ecdsa_key.pub": line,
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.Keys, 1)
				if "ecdsa-sha2-unknown" != "" && len(info.Keys) > 0 {
					s.Equal("ecdsa-sha2-unknown", info.Keys[0].Type)
					s.NotEmpty(info.Keys[0].FingerprintSHA256)
					s.NotEmpty(info.Keys[0].FingerprintMD5)
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c gohaisssh.Collector
			switch tt.variant {
			case "linux":
				c = &gohaisssh.Linux{FS: tt.setupFS()}
			case "darwin":
				c = &gohaisssh.Darwin{FS: tt.setupFS()}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
