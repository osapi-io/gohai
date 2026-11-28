# CA certificates

> **Status:** Implemented ✅

## Description

The collector reports the certificate authorities (CAs) that the host trusts.
These are the CAs that the distribution ships, and the CAs that an administrator
added. For each CA, it gives the file that holds it, its subject, the SHA-256
fingerprint of the certificate, and the time at which it expires.

Consumers use this data to:

- Find the CAs that an administrator added to a host, such as a company's own
  CA, across a fleet.
- Find a CA that expires soon, or a CA that the host config turns off.
- Compare the trust store of hosts of one distribution, to find drift.

The collector is off by default. It reads and parses each CA file, and a typical
host has more than one hundred.

## Collected fields

| Field                        | Type     | Description                                                                                                                                     | Schema mapping                                                               |
| ---------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `certificates[].name`        | `string` | The name of the CA: the path of its file under its directory, with no extension, or its subject's common name when one file holds many CAs.     | No direct schema mapping. gohai treats the trust store as a gohai extension. |
| `certificates[].path`        | `string` | The file that holds the CA, or the keychain on macOS.                                                                                           | `file.path`                                                                  |
| `certificates[].source`      | `string` | `system` for a CA that the distribution ships, `local` for a CA that an administrator added.                                                    | No direct schema mapping.                                                    |
| `certificates[].enabled`     | `bool`   | `false` for a system CA that the host config turns off, such as a `!` line in `/etc/ca-certificates.conf`, or for a CA that the host distrusts. | No direct schema mapping.                                                    |
| `certificates[].subject`     | `string` | The subject of the certificate, when the file parses.                                                                                           | `certificate.subject`                                                        |
| `certificates[].fingerprint` | `string` | The SHA-256 of the certificate's DER, in hex.                                                                                                   | `certificate.fingerprints[].value`, with `algorithm` SHA-256                 |
| `certificates[].not_after`   | `string` | The time at which the certificate expires, in RFC 3339.                                                                                         | `certificate.expiration_time`                                                |

## Platform support

| Platform                   | Supported | Where it reads                                                                                |
| -------------------------- | --------- | --------------------------------------------------------------------------------------------- |
| Debian, Ubuntu, Alpine     | ✅        | `/usr/share/ca-certificates`, `/usr/local/share/ca-certificates`, `/etc/ca-certificates.conf` |
| RHEL, Fedora, Amazon Linux | ✅        | `/usr/share/pki/ca-trust-source`, `/etc/pki/ca-trust/source/anchors`                          |
| SUSE                       | ✅        | `/usr/share/pki/trust`, `/etc/pki/trust/anchors`                                              |
| Arch                       | ✅        | `/usr/share/ca-certificates/trust-source`, `/etc/ca-certificates/trust-source/anchors`        |
| macOS                      | ✅        | the SystemRootCertificates and System keychains, with `security find-certificate -a -p`       |

A directory that the host does not have gives no CAs, not an error.

## Example output

### Typical Linux (Debian)

```json
{
  "ca_certificates": {
    "certificates": [
      {
        "name": "mozilla/ACCVRAIZ1",
        "path": "/usr/share/ca-certificates/mozilla/ACCVRAIZ1.crt",
        "source": "system",
        "enabled": true,
        "subject": "CN=ACCVRAIZ1,OU=PKIACCV,O=ACCV,C=ES",
        "fingerprint": "9a6ec012e1a7da9dbe34194d478ad7c0db1822fb071df12981496ed104384113",
        "not_after": "2030-12-31T09:37:37Z"
      }
    ]
  }
}
```

### Typical macOS

```json
{
  "ca_certificates": {
    "certificates": [
      {
        "name": "DigiCert TLS ECC P384 Root G5",
        "path": "/System/Library/Keychains/SystemRootCertificates.keychain",
        "source": "system",
        "enabled": true,
        "subject": "CN=DigiCert TLS ECC P384 Root G5,O=DigiCert\\, Inc.,C=US",
        "fingerprint": "018e13f0772532cf809bd1b17281867283fc48c6e13be9c69812854a490c1b05",
        "not_after": "2046-01-14T23:59:59Z"
      }
    ]
  }
}
```

## SDK usage

```go
import (
    "context"
    "fmt"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("ca_certificates"))
facts, _ := g.Collect(context.Background())

for _, c := range facts.CACertificates.Certificates {
    fmt.Println(c.Source, c.Name, c.NotAfter)
}
```

## Enable/disable

```bash
gohai collect --collector.ca_certificates      # enable
gohai collect --no-collector.ca_certificates   # disable (default)
```

## Dependencies

None.

## Data sources

### Linux

1. The collector selects the directories of the host's family (the table above).
   It walks each directory, and reads each file with the extension `.crt`,
   `.pem`, `.cer` or `.p11-kit`. A directory or a file that the host does not
   have, such as a link that points at nothing, has no CAs. A directory, a file,
   a blocklist or the conf file that is there and cannot be read is an error, so
   that the host does not look as if it has fewer CAs.
2. It decodes each PEM block of type `CERTIFICATE`, and parses it with
   `crypto/x509`. A block of a different type, or a block that does not parse,
   is left out. A file that holds no certificate that parses gives one entry
   with its name, path and source only.
3. A file with one certificate gives one entry named by the file. A file with
   many, such as Fedora's `ca-bundle.trust.p11-kit`, gives one entry for each
   certificate, named by its subject's common name.
4. On the Debian family, it reads `/etc/ca-certificates.conf`. A line `!name`
   turns the system CA `name` off, and its entry has `enabled: false`.
5. On the Red Hat, SUSE and Arch families, p11-kit sets the trust. The collector
   reads the certificates in the `blocklist` and `blacklist` directories of each
   trust source, for example `/usr/share/pki/ca-trust-source/blocklist` and
   `/etc/pki/ca-trust/source/blocklist` on the Red Hat family. A file in a
   blocklist is not a CA that the host trusts. Thus it gives no entry.
6. In a `.p11-kit` file, an object with `x-distrusted: true` marks its
   certificate as distrusted. Fedora's bundle has one: "Explicitly Distrust
   DigiNotar Root CA".
7. A CA whose SHA-256 fingerprint is the same as the fingerprint of a
   certificate in step 5 or step 6 has `enabled: false`. This applies to a
   system CA and to a local CA.

The collector does not read the other trust attributes of p11-kit, such as the
extensions that limit the purposes of a CA, or `nss-server-distrust-after`.

### macOS

1. The collector runs `security find-certificate -a -p` on
   `/System/Library/Keychains/SystemRootCertificates.keychain` (the roots that
   Apple ships, `system`) and on `/Library/Keychains/System.keychain` (the CAs
   that an administrator added, `local`).
2. It parses each PEM certificate, and keeps only the certificates that are CAs.
   The System keychain can also hold certificates that are not CAs, such as a
   certificate of the device.
3. A host without `security` gives no CAs. A keychain that `security` fails to
   read is an error.

## Backing library

The Go standard library: `encoding/pem`, `crypto/x509` and `crypto/sha256`.
