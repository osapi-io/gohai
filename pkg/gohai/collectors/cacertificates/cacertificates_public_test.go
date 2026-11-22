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

package cacertificates_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cacertificates"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*cacertificates.Linux)(nil)
	_ collector.Collector = (*cacertificates.Darwin)(nil)
)

type CACertificatesPublicTestSuite struct {
	suite.Suite
}

// cert is a self-signed certificate with the subject's common name cn, as
// PEM, its fingerprint, and when it expires.
type cert struct {
	pem, fingerprint, notAfter string
}

func (s *CACertificatesPublicTestSuite) cert(
	cn string,
	isCA bool,
) cert {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	s.Require().NoError(err)
	notAfter := time.Date(2036, 1, 2, 3, 4, 5, 0, time.UTC)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn, Organization: []string{"Test"}},
		NotBefore:             notAfter.AddDate(-10, 0, 0),
		NotAfter:              notAfter,
		IsCA:                  isCA,
		BasicConstraintsValid: true,
	}
	if cn == "" {
		tmpl.Subject = pkix.Name{Organization: []string{"No CN"}}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	s.Require().NoError(err)
	sum := sha256.Sum256(der)

	return cert{
		pem:         string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		fingerprint: hex.EncodeToString(sum[:]),
		notAfter:    notAfter.Format(time.RFC3339),
	}
}

func (s *CACertificatesPublicTestSuite) TestNew() {
	tests := []struct {
		name       string
		family     string
		wantDarwin bool
		wantLayout cacertificates.Layout
	}{
		{name: "when the host is macOS", family: "darwin", wantDarwin: true},
		{
			name:       "when the host is Debian",
			family:     "debian",
			wantLayout: cacertificates.LayoutFor("debian"),
		},
		{
			name:       "when the host is Red Hat",
			family:     "rhel",
			wantLayout: cacertificates.LayoutFor("rhel"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := cacertificates.NewWith(source.Sources{Family: tt.family})
			s.Equal("ca_certificates", c.Name())
			s.Equal(collector.CategorySecurity, c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			if tt.wantDarwin {
				s.IsType(&cacertificates.Darwin{}, c)

				return
			}
			s.Require().IsType(&cacertificates.Linux{}, c)
			s.Equal(tt.wantLayout, c.(*cacertificates.Linux).Layout)
		})
	}
	s.NotNil(cacertificates.New())
	s.Equal(cacertificates.LayoutFor("rhel"), cacertificates.NewLinux().Layout)
	s.NotNil(cacertificates.NewDarwin().Exec)
}

func (s *CACertificatesPublicTestSuite) TestLayoutFor() {
	tests := []struct {
		family    string
		wantFirst string
		wantConf  string
	}{
		{
			family:    "debian",
			wantFirst: "/usr/share/ca-certificates",
			wantConf:  "/etc/ca-certificates.conf",
		},
		{
			family:    "alpine",
			wantFirst: "/usr/share/ca-certificates",
			wantConf:  "/etc/ca-certificates.conf",
		},
		{family: "suse", wantFirst: "/usr/share/pki/trust"},
		{family: "arch", wantFirst: "/usr/share/ca-certificates/trust-source"},
		{family: "fedora", wantFirst: "/usr/share/pki/ca-trust-source"},
	}

	for _, tt := range tests {
		s.Run(tt.family, func() {
			l := cacertificates.LayoutFor(tt.family)
			s.Equal(tt.wantFirst, l.Dirs[0].Path)
			s.Equal(cacertificates.SourceSystem, l.Dirs[0].Source)
			s.Equal(cacertificates.SourceLocal, l.Dirs[1].Source)
			s.Equal(tt.wantConf, l.Conf)
		})
	}
}

func (s *CACertificatesPublicTestSuite) TestCollect() {
	root := s.cert("Root CA", true)
	other := s.cert("Other CA", true)
	noCN := s.cert("", true)
	leaf := s.cert("leaf.example.com", false)
	corrupt := "-----BEGIN CERTIFICATE-----\nbm90IGEgY2VydA==\n-----END CERTIFICATE-----\n"
	key := "-----BEGIN PRIVATE KEY-----\nbm90IGEga2V5\n-----END PRIVATE KEY-----\n"

	tests := []struct {
		name    string
		variant string
		layout  string
		files   map[string]string
		links   map[string]string
		runner  sourcetest.Commands
		want    []cacertificates.Certificate
	}{
		{
			name:    "when a Debian host has system and local CAs, and turns one off",
			variant: "linux",
			layout:  "debian",
			files: map[string]string{
				"/usr/share/ca-certificates/mozilla/Root_CA.crt":  root.pem,
				"/usr/share/ca-certificates/mozilla/Other_CA.crt": other.pem,
				"/usr/share/ca-certificates/mozilla/README":       "not a CA",
				"/usr/local/share/ca-certificates/osapi-ours.crt": noCN.pem,
				"/etc/ca-certificates.conf":                       "# comment\nmozilla/Root_CA.crt\n!mozilla/Other_CA.crt\n",
			},
			want: []cacertificates.Certificate{
				{
					Name:        "mozilla/Other_CA",
					Path:        "/usr/share/ca-certificates/mozilla/Other_CA.crt",
					Source:      "system",
					Enabled:     false,
					Subject:     "CN=Other CA,O=Test",
					Fingerprint: other.fingerprint,
					NotAfter:    other.notAfter,
				},
				{
					Name: "mozilla/Root_CA", Path: "/usr/share/ca-certificates/mozilla/Root_CA.crt",
					Source: "system", Enabled: true, Subject: "CN=Root CA,O=Test",
					Fingerprint: root.fingerprint, NotAfter: root.notAfter,
				},
				{
					Name: "osapi-ours", Path: "/usr/local/share/ca-certificates/osapi-ours.crt",
					Source: "local", Enabled: true, Subject: "O=No CN",
					Fingerprint: noCN.fingerprint, NotAfter: noCN.notAfter,
				},
			},
		},
		{
			name:    "when a Red Hat bundle holds many CAs, each is named by its subject",
			variant: "linux",
			layout:  "rhel",
			files: map[string]string{
				"/usr/share/pki/ca-trust-source/ca-bundle.trust.p11-kit": "[p11-kit-object-v1]\n" + root.pem + key + corrupt + noCN.pem,
				"/etc/pki/ca-trust/source/anchors/broken.pem":            "not PEM",
			},
			want: []cacertificates.Certificate{
				{
					Name: "Root CA", Path: "/usr/share/pki/ca-trust-source/ca-bundle.trust.p11-kit",
					Source: "system", Enabled: true, Subject: "CN=Root CA,O=Test",
					Fingerprint: root.fingerprint, NotAfter: root.notAfter,
				},
				{
					Name: "O=No CN", Path: "/usr/share/pki/ca-trust-source/ca-bundle.trust.p11-kit",
					Source: "system", Enabled: true, Subject: "O=No CN",
					Fingerprint: noCN.fingerprint, NotAfter: noCN.notAfter,
				},
				{
					Name: "broken", Path: "/etc/pki/ca-trust/source/anchors/broken.pem",
					Source: "local", Enabled: true,
				},
			},
		},
		{
			name:    "when the host has no CA directories, no conf, and an unreadable file",
			variant: "linux",
			layout:  "debian",
			files:   map[string]string{"/usr/local/share/ca-certificates/README": "x"},
			links:   map[string]string{"/usr/local/share/ca-certificates/gone.crt": "/nowhere"},
			want:    []cacertificates.Certificate{},
		},
		{
			name:    "when macOS has roots and a local CA beside a certificate that is not a CA",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"security find-certificate -a -p /System/Library/Keychains/SystemRootCertificates.keychain": root.pem,
				"security find-certificate -a -p /Library/Keychains/System.keychain":                        leaf.pem + other.pem,
			}},
			want: []cacertificates.Certificate{
				{
					Name:        "Root CA",
					Path:        "/System/Library/Keychains/SystemRootCertificates.keychain",
					Source:      "system",
					Enabled:     true,
					Subject:     "CN=Root CA,O=Test",
					Fingerprint: root.fingerprint,
					NotAfter:    root.notAfter,
				},
				{
					Name: "Other CA", Path: "/Library/Keychains/System.keychain",
					Source: "local", Enabled: true, Subject: "CN=Other CA,O=Test",
					Fingerprint: other.fingerprint, NotAfter: other.notAfter,
				},
			},
		},
		{
			name:    "when security cannot read the keychains",
			variant: "darwin",
			runner: sourcetest.Commands{Err: map[string]error{
				"security find-certificate -a -p /System/Library/Keychains/SystemRootCertificates.keychain": errors.New(
					"denied",
				),
			}},
			want: []cacertificates.Certificate{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c collector.Collector
			switch tt.variant {
			case "linux":
				fs := sourcetest.FS(s.T(), tt.files)
				if tt.links != nil {
					fs = sourcetest.Links(s.T(), fs, tt.links)
				}
				c = &cacertificates.Linux{FS: fs, Layout: cacertificates.LayoutFor(tt.layout)}
			case "darwin":
				c = &cacertificates.Darwin{Exec: tt.runner.Mock(s.T())}
			}

			got, err := c.Collect(context.Background(), collector.PriorResults{})
			s.Require().NoError(err)
			s.Equal(tt.want, got.(*cacertificates.Info).Certificates)
		})
	}
}

func TestCACertificatesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CACertificatesPublicTestSuite))
}
