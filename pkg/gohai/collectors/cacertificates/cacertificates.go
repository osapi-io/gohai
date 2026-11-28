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

// Package cacertificates reports the certificate authorities that the host
// trusts: the CAs that its distribution ships and the ones that an
// administrator added, each with its subject, its SHA-256 fingerprint and
// when it expires.
package cacertificates

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// The sources of a CA: one the distribution ships, or one an administrator
// added.
const (
	SourceSystem = "system"
	SourceLocal  = "local"
)

// Certificate is one CA that the host trusts.
type Certificate struct {
	// Name names the CA: its file's path under its directory, without the
	// extension, or its subject's common name when one file holds many.
	Name string `json:"name"`
	// Path is the file that holds it, or the keychain on macOS.
	Path string `json:"path"`
	// Source is system or local.
	Source string `json:"source"`
	// Enabled is false for a system CA that the host's config turns off,
	// such as a "!" line in /etc/ca-certificates.conf, and for a CA that
	// the host distrusts, such as one in a p11-kit blocklist. On macOS it
	// is always true, because the collector does not read the trust
	// settings.
	Enabled bool `json:"enabled"`
	// Subject is the certificate's subject, when the file parses.
	Subject string `json:"subject,omitempty"`
	// Fingerprint is the SHA-256 of the certificate's DER, in hex.
	Fingerprint string `json:"fingerprint,omitempty"`
	// NotAfter is when it expires, in RFC 3339.
	NotAfter string `json:"not_after,omitempty"`
}

// Info is the CAs that the host trusts.
type Info struct {
	Certificates []Certificate `json:"certificates"`
}

// Collector is the public interface every variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "ca_certificates".
func (base) Name() string { return "ca_certificates" }

// Category returns "security".
func (base) Category() collector.Category { return collector.CategorySecurity }

// DefaultEnabled returns false: it reads and parses each CA file, more than
// a hundred on a typical host.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the variant for the host's OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src. A Linux host's family decides where its CAs are.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}
	l := NewLinux()
	l.Layout = LayoutFor(collector.Family(src))

	return collector.Wire(l, src)
}

// certs are the certificates of the PEM blocks in data. A block that is
// not a certificate, or does not parse, is left out.
func certs(
	data []byte,
) []*x509.Certificate {
	var out []*x509.Certificate
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return out
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
}

// describe fills the fields of e that come from c.
func describe(
	e Certificate,
	c *x509.Certificate,
) Certificate {
	e.Subject = c.Subject.String()
	e.Fingerprint = fingerprint(c)
	e.NotAfter = c.NotAfter.UTC().Format(time.RFC3339)

	return e
}

// fingerprint is the SHA-256 of c's DER, in hex.
func fingerprint(
	c *x509.Certificate,
) string {
	sum := sha256.Sum256(c.Raw)

	return hex.EncodeToString(sum[:])
}

// nameOf is the name of a CA that one file of many holds: its subject's
// common name, else its subject.
func nameOf(
	c *x509.Certificate,
) string {
	if c.Subject.CommonName != "" {
		return c.Subject.CommonName
	}

	return c.Subject.String()
}
