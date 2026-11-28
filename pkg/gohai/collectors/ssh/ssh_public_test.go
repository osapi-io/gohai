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
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	gohaisssh "github.com/osapi-io/gohai/pkg/gohai/collectors/ssh"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// dsclHomes is the command line that lists the home of each macOS user.
const dsclHomes = "dscl . -list /Users NFSHomeDirectory"

var (
	_ collector.Collector = (*gohaisssh.Linux)(nil)
	_ collector.Collector = (*gohaisssh.Darwin)(nil)
)

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
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
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
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c gohaisssh.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := gohaisssh.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = gohaisssh.New()
			}
			s.Equal("ssh", c.Name())
			s.Equal("security", string(c.Category()))
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
	// Wire layout: algo-name(ssh-rsa) | exponent | (no modulus: the blob
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
		commands     sourcetest.Commands
		nilExec      bool
		cancel       bool
		validateFunc func(any, error)
	}{
		{
			name:    "linux: authorized keys of each user with a home",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd": "root:x:0:0::/root:/bin/sh\n" +
						"john:x:1000:1000::/home/john/:/bin/bash\n" +
						"nobody:x:65534:65534::/:/usr/sbin/nologin\n" +
						"nohome:x:2:2:::/bin/sh\n" +
						"jane:x:1001:1001::/home/jane:/bin/bash\n" +
						"short\n",
					"/root/.ssh/authorized_keys": "# a comment\n\n" +
						"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 admin@lab\n",
					"/home/john/.ssh/authorized_keys": "no-pty,from=\"10.0.0.0/8\" ecdsa-sha2-nistp256 AAAAE2VjZHNh john laptop\n" +
						"ssh-rsa !!!not-base64 bad\n" +
						"just some words\n",
					"/home/jane/.ssh/authorized_keys": "# empty but for a comment\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				// The user jane has no key, and nobody and nohome are skipped.
				s.Require().
					Len(info.AuthorizedKeys, 2)
				root := info.AuthorizedKeys["root"]
				s.Require().Len(root, 1)
				s.Equal("ssh-ed25519", root[0].Type)
				s.Equal("admin@lab", root[0].Comment)
				s.Empty(root[0].Options)
				s.True(strings.HasPrefix(root[0].FingerprintSHA256, "SHA256:"))
				john := info.AuthorizedKeys["john"]
				// The bad base64 line and the line with no key are left out.
				s.Require().
					Len(john, 1)
				s.Equal("ecdsa-sha2-nistp256", john[0].Type)
				s.Equal(`no-pty,from="10.0.0.0/8"`, john[0].Options)
				s.Equal("john laptop", john[0].Comment)
			},
		},
		{
			name:    "linux: an LDAP user from getent passwd",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                      "root:x:0:0::/root:/bin/sh\n",
					"/home/alice/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 alice@ldap\n",
				})
			},
			commands: sourcetest.Commands{Out: map[string]string{
				"getent passwd": "root:x:0:0::/root:/bin/sh\nalice:*:5001:5001::/home/alice:/bin/bash\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Require().Len(info.AuthorizedKeys["alice"], 1)
				s.Equal("alice@ldap", info.AuthorizedKeys["alice"][0].Comment)
			},
		},
		{
			name:    "linux: getent with no users falls back to /etc/passwd",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                "root:x:0:0::/root:/bin/sh\n",
					"/root/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 root@lab\n",
				})
			},
			commands: sourcetest.Commands{Out: map[string]string{"getent passwd": "\n"}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.AuthorizedKeys["root"], 1)
			},
		},
		{
			name:    "linux: the AuthorizedKeysFile of sshd -T, with its tokens",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd": "svc:x:900:900::/:/usr/sbin/nologin\n" +
						"john:x:1000:1000::/home/john:/bin/bash\n",
					"/etc/ssh/authorized_keys/svc":     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 svc@ci\n",
					"/etc/ssh/keys/1000%/john":         "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@uid\n",
					"/home/john/.ssh/authorized_keys":  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@home\n",
					"/home/john/.ssh/authorized_keys2": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@two\n",
				})
			},
			commands: sourcetest.Commands{Out: map[string]string{
				"sshd -T": "port 22\nauthorizedkeysfile /etc/ssh/authorized_keys/%u " +
					"/etc/ssh/keys/%U%%/%u .ssh/authorized_keys %h/.ssh/authorized_keys %x bad%\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				svc := info.AuthorizedKeys["svc"]
				// A home of / reads an absolute pattern only.
				s.Require().Len(svc, 1)
				s.Equal("svc@ci", svc[0].Comment)
				john := info.AuthorizedKeys["john"]
				// The same file twice is read once, and authorized_keys2 is not named.
				s.Require().
					Len(john, 2)
				s.Equal("john@uid", john[0].Comment)
				s.Equal("john@home", john[1].Comment)
			},
		},
		{
			name:    "linux: authorized_keys2 is read when sshd -T does not run",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                      "john:x:1000:1000::/home/john:/bin/bash\n",
					"/home/john/.ssh/authorized_keys2": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@two\n",
				})
			},
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.AuthorizedKeys["john"], 1)
			},
		},
		{
			name:    "linux: an AuthorizedKeysFile of none, and sshd -T with no such line",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                     "john:x:1000:1000::/home/john:/bin/bash\n",
					"/home/john/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@home\n",
				})
			},
			commands: sourcetest.Commands{
				Out: map[string]string{"sshd -T": "authorizedkeysfile none\n"},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Nil(info.AuthorizedKeys)
			},
		},
		{
			name:    "linux: sshd -T with no AuthorizedKeysFile line has the default",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                     "john:x:1000:1000::/home/john:/bin/bash\n",
					"/home/john/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@home\n",
				})
			},
			commands: sourcetest.Commands{Out: map[string]string{"sshd -T": "port 22\n"}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Len(info.AuthorizedKeys["john"], 1)
			},
		},
		{
			name:    "linux: quoted options keep their text, and command and environment are redacted",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd": "john:x:1000:1000::/home/john:/bin/bash\n",
					"/home/john/.ssh/authorized_keys": `command="/bin/deploy  --token s3cret ssh-rsa x",` +
						`environment="API_KEY=abc \"q\"",no-pty ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john  laptop` + "\n" +
						`"quoted first" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 q` + "\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				john := info.AuthorizedKeys["john"]
				s.Require().Len(john, 2)
				s.Equal(`"quoted first"`, john[1].Options)
				// A key type inside a quoted option is not the key.
				s.Equal(
					"ssh-ed25519",
					john[0].Type,
				)
				s.Equal(`command="[redacted]",environment="[redacted]",no-pty`, john[0].Options)
				s.Equal("john  laptop", john[0].Comment)
				s.NotContains(john[0].Line, "s3cret")
				s.NotContains(john[0].Line, "API_KEY")
				s.True(
					strings.HasSuffix(
						john[0].Line,
						"AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john  laptop",
					),
				)
			},
		},
		{
			// Most macOS users are in the directory service, not in
			// /etc/passwd. dscl gives the home of each one.
			name:    "darwin: authorized keys of a directory service user",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                          "root:*:0:0::/var/root:/bin/sh\n",
					"/Users/john/.ssh/authorized_keys":     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 john@mac\n",
					"/var/root/.ssh/authorized_keys":       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 root@mac\n",
					"/Users/Jo Smith/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 jo@mac\n",
				})
			},
			commands: sourcetest.Commands{Out: map[string]string{
				dsclHomes: "_www                     /Library/WebServer\n" +
					"daemon                   /var/root\n" +
					"jo                       /Users/Jo Smith\n" +
					"john                     /Users/john/\n" +
					"nobody                   /var/empty\n" +
					"nohome\n" +
					"slash                    /\n" +
					"root                     /var/root /private/var/root\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				// The users john, jo, root and daemon, which shares the home
				// of root. jo's home has a space.
				s.Require().
					Len(info.AuthorizedKeys, 4)
				s.Require().Len(info.AuthorizedKeys["jo"], 1)
				s.Equal("jo@mac", info.AuthorizedKeys["jo"][0].Comment)
				s.Require().Len(info.AuthorizedKeys["john"], 1)
				s.Equal("john@mac", info.AuthorizedKeys["john"][0].Comment)
				s.Require().Len(info.AuthorizedKeys["root"], 1)
				s.Equal("root@mac", info.AuthorizedKeys["root"][0].Comment)
			},
		},
		{
			name:    "darwin: dscl fails, so /etc/passwd gives the homes",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                    "root:*:0:0::/var/root:/bin/sh\n",
					"/var/root/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 root@mac\n",
				})
			},
			commands: sourcetest.Commands{
				Err: map[string]error{dsclHomes: errors.New("exec dscl: not found")},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Require().Len(info.AuthorizedKeys["root"], 1)
			},
		},
		{
			name:    "darwin: nil Exec, so /etc/passwd gives the homes",
			variant: "darwin",
			nilExec: true,
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                    "root:*:0:0::/var/root:/bin/sh\n",
					"/var/root/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 root@mac\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Require().Len(info.AuthorizedKeys["root"], 1)
			},
		},
		{
			name:    "linux: a host key file that does not read is an error",
			variant: "linux",
			nilExec: true,
			setupFS: func() avfs.VFS {
				return sourcetest.FailRead(s.T(), memfs.New(), "/etc/ssh/ssh_host_rsa_key.pub")
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/ssh/ssh_host_rsa_key.pub")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an /etc/passwd that does not read is an error",
			variant: "linux",
			nilExec: true,
			setupFS: func() avfs.VFS {
				return sourcetest.FailRead(s.T(), memfs.New(), "/etc/passwd")
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/passwd")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: dscl fails and /etc/passwd does not read: an error",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return sourcetest.FailRead(s.T(), memfs.New(), "/etc/passwd")
			},
			commands: sourcetest.Commands{
				Err: map[string]error{dsclHomes: errors.New("exec dscl: not found")},
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/passwd")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an authorized_keys file that does not read is an error",
			variant: "linux",
			nilExec: true,
			setupFS: func() avfs.VFS {
				fs := sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd": "root:x:0:0::/root:/bin/sh\n",
				})
				return sourcetest.FailRead(s.T(), fs, "/root/.ssh/authorized_keys")
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /root/.ssh/authorized_keys")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: a host key file that does not read is an error",
			variant: "darwin",
			nilExec: true,
			setupFS: func() avfs.VFS {
				return sourcetest.FailRead(s.T(), memfs.New(), "/etc/ssh/ssh_host_ed25519_key.pub")
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /etc/ssh/ssh_host_ed25519_key.pub")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: an authorized_keys file that does not read is an error",
			variant: "darwin",
			nilExec: true,
			setupFS: func() avfs.VFS {
				fs := sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd": "root:*:0:0::/var/root:/bin/sh\n",
				})
				return sourcetest.FailRead(s.T(), fs, "/var/root/.ssh/authorized_keys")
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /var/root/.ssh/authorized_keys")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: a cancelled collection is an error",
			variant: "darwin",
			nilExec: true,
			cancel:  true,
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                    "root:*:0:0::/var/root:/bin/sh\n",
					"/var/root/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 root@mac\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a cancelled collection is an error",
			variant: "linux",
			nilExec: true,
			cancel:  true,
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/passwd":                "root:x:0:0::/root:/bin/sh\n",
					"/root/.ssh/authorized_keys": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 root@lab\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			// A host key that does not parse does not hide the
			// authorized keys, which do not depend on it.
			name:    "linux: a bad host key keeps the authorized keys",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub":     "ssh-rsa !!!not-base64 bad\n",
					"/etc/ssh/ssh_host_ed25519_key.pub": string(generateEd25519Key(s.T())),
					"/etc/passwd":                       "root:x:0:0::/root:/bin/sh\n",
					"/root/.ssh/authorized_keys":        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG9zYXBpIHRlc3Qga2V5IG1hdGVyaWFsIDEyMzQ1 admin@lab\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				// The bad rsa key is left out.
				s.Require().Len(info.Keys, 1)
				s.Equal("ssh-ed25519", info.Keys[0].Type)
				s.Require().Len(info.AuthorizedKeys["root"], 1)
			},
		},
		{
			name:    "darwin: users with no authorized_keys file",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(
					s.T(),
					map[string]string{"/etc/passwd": "root:*:0:0::/var/root:/bin/sh\n"},
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Nil(info.AuthorizedKeys)
			},
		},
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
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(generateRSAKey(s.T(), 2048)),
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
					s.Equal(2048, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: rsa-4096 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(generateRSAKey(s.T(), 4096)),
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
					s.Equal(4096, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: a truncated rsa wire blob has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(pubLineFromBlob(truncatedRSABlob)),
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
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ecdsa-p256 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_ecdsa_key.pub": string(
						generateECDSAKey(s.T(), elliptic.P256()),
					),
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
					s.Equal(256, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ecdsa-p384 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_ecdsa_key.pub": string(
						generateECDSAKey(s.T(), elliptic.P384()),
					),
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
					s.Equal(384, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ecdsa-p521 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_ecdsa_key.pub": string(
						generateECDSAKey(s.T(), elliptic.P521()),
					),
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
					s.Equal(521, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: ed25519 key",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_ed25519_key.pub": string(generateEd25519Key(s.T())),
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
					s.Equal(256, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: all three key types",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(generateRSAKey(s.T(), 2048)),
					"/etc/ssh/ssh_host_ecdsa_key.pub": string(
						generateECDSAKey(s.T(), elliptic.P256()),
					),
					"/etc/ssh/ssh_host_ed25519_key.pub": string(generateEd25519Key(s.T())),
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
			name:    "linux: malformed base64 is left out",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": "ssh-rsa !!!notbase64!!! comment\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Empty(info.Keys)
			},
		},
		{
			name:    "linux: too few fields is left out",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": "ssh-rsa\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Empty(info.Keys)
			},
		},
		{
			name:    "darwin: ed25519 key",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_ed25519_key.pub": string(generateEd25519Key(s.T())),
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
			name:    "darwin: malformed base64 is left out",
			variant: "darwin",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": "ssh-rsa !!!notbase64!!! comment\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*gohaisssh.Info)
				s.Require().True(ok)
				s.Empty(info.Keys)
			},
		},
		{
			name:    "linux: an unknown key type has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// Build a blob whose type field is not rsa/ecdsa/ed25519 to
				// exercise the default branch in deriveKeyLength.
				unknownBlob := marshalWireString([]byte("ssh-unknown"))
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(pubLineFromBlob(unknownBlob)),
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
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: a wire blob too short for its length header has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// 3-byte blob: too short for even the first 4-byte length
				// header. skipWireString returns nil (len < 4 branch).
				shortBlob := []byte{0x00, 0x01, 0x02}
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(pubLineFromBlob(shortBlob)),
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
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			name:    "linux: a wire blob whose length field is more than its bytes has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// The length header claims 100 bytes but only 2 follow. That is the
				// len(b) < 4+n branch in skipWireString.
				tooShort := []byte{0, 0, 0, 100, 0x01, 0x02}
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(pubLineFromBlob(tooShort)),
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
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			// Exercise readWireString's len(b) < 4+n branch: blob has two
			// valid fields (algo + exponent) so skipWireString succeeds twice,
			// then the modulus field header claims 100 bytes but only 2 remain.
			name:    "linux: a modulus longer than the bytes left has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				algoField := marshalWireString([]byte("ssh-rsa"))
				expField := marshalWireString([]byte{1, 0, 1})
				// Modulus header claims 100 bytes but only 2 bytes follow.
				modHeader := []byte{0, 0, 0, 100, 0x00, 0x00}
				blob := append(append(algoField, expField...), modHeader...)
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(pubLineFromBlob(blob)),
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
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			// Exercise deriveKeyLength default branch: a fake type that is
			// not rsa, ecdsa, or ed25519. The file field[0] is "ssh-dss"
			// (deprecated) so keyType hits the default return 0.
			name:    "linux: a dsa key has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				blob := marshalWireString([]byte("ssh-dss"))
				line := append(
					[]byte("ssh-dss "),
					[]byte(base64.StdEncoding.EncodeToString(blob)+"\n")...,
				)
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_rsa_key.pub": string(line),
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
					s.Equal(0, info.Keys[0].KeyLength)
				}
			},
		},
		{
			// Exercise deriveKeyLength ecdsa inner-switch default: key type
			// has "ecdsa" prefix but not a recognized nistp* curve label.
			// This hits the `return 0` after the inner ecdsa switch.
			name:    "linux: an ecdsa key with an unknown curve has key length 0",
			variant: "linux",
			setupFS: func() avfs.VFS {
				blob := marshalWireString([]byte("ecdsa-sha2-unknown"))
				line := append(
					[]byte("ecdsa-sha2-unknown "),
					[]byte(base64.StdEncoding.EncodeToString(blob)+"\n")...,
				)
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/ssh/ssh_host_ecdsa_key.pub": string(line),
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
				var exec source.Runner = tt.commands.Mock(s.T())
				if tt.nilExec {
					exec = nil
				}
				c = &gohaisssh.Linux{FS: tt.setupFS(), Exec: exec}
			case "darwin":
				var exec source.Runner = tt.commands.Mock(s.T())
				if tt.nilExec {
					exec = nil
				}
				c = &gohaisssh.Darwin{FS: tt.setupFS(), Exec: exec}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			tt.validateFunc(c.Collect(ctx, nil))
		})
	}
}

func (s *SSHPublicTestSuite) TestHostKeyJSON() {
	tests := []struct {
		name         string
		key          gohaisssh.HostKey
		validateFunc func(fields map[string]any)
	}{
		{
			name: "has no MD5 fingerprint, which FIPS 140-3 does not approve",
			key: gohaisssh.HostKey{
				Type:              "ssh-ed25519",
				FingerprintSHA256: "SHA256:abc",
				KeyLength:         256,
			},
			validateFunc: func(fields map[string]any) {
				s.NotContains(fields, "fingerprint_md5")
				s.Equal("SHA256:abc", fields["fingerprint_sha256"])
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			b, err := json.Marshal(tc.key)
			s.Require().NoError(err)
			var fields map[string]any
			s.Require().NoError(json.Unmarshal(b, &fields))
			tc.validateFunc(fields)
		})
	}
}
