# SSH

> **Status:** Implemented ✅

## Description

The collector reports the fingerprints of the SSH host public keys in
`/etc/ssh/`. For each key type on the host (RSA, ECDSA, Ed25519), it reports:

- The algorithm type.
- The SHA-256 fingerprint, in OpenSSH format.
- The MD5 fingerprint, in colon-hex format.
- The key length in bits.

Consumers use this data to:

- Make a list of the host key fingerprints across a fleet, for TOFU
  (trust-on-first-use) or to track key rotation.
- Make sure that the expected key types are there, and that deprecated types,
  for example DSA, are removed.
- Compare fingerprints with known-host databases or SIEM events.
- Find changes to the key material after an OS reinstall or after a cloud image
  is baked.

The collector does not report DSA keys. This is intentional. OpenSSH deprecated
DSA in 7.0 (2015) and removed support in 9.8. gohai does not report DSA keys. If
it reports them, consumers can think that DSA keys are still a valid security
signal.

## Signals

The collector reports these signals for each key type:

- `keys[].fingerprint_sha256`. The SHA-256 fingerprint in `SHA256:<base64>`
  format. It is the same as the output of `ssh-keygen -l -E sha256`. Use it to
  compare keys and to correlate SIEM events.
- `keys[].fingerprint_md5`. The MD5 fingerprint in `xx:xx:...` colon-hex format.
  It is the same as the output of `ssh-keygen -l -E md5`. It is there for older
  `known_hosts` tools. For new work, use SHA-256.
- `keys[].key_length`. The effective key strength in bits. For RSA, this is the
  bit length of the modulus. For ECDSA, it is the curve size (256/384/521). For
  Ed25519, it is always 256. Use it to find keys that are below the minimum
  strength of a policy.

## Collected fields

| Field                       | Type     | Description                                                           | Schema mapping                                                                                                                                    |
| --------------------------- | -------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `keys[].type`               | `string` | The key algorithm as the .pub file gives it, for example `ssh-rsa`.   | No direct OCSF mapping. OTel has no SSH host key object. gohai convention: the key type field, unchanged, from the public key file.               |
| `keys[].fingerprint_sha256` | `string` | The SHA-256 fingerprint in `SHA256:<base64>` format.                  | No direct OCSF or OTel mapping. gohai convention: `fingerprint_sha256`. It is the same as the output format of OpenSSH `-E sha256`.               |
| `keys[].fingerprint_md5`    | `string` | The MD5 fingerprint in `xx:xx:...` colon-hex format.                  | No direct OCSF or OTel mapping. gohai convention: `fingerprint_md5`. It is the same as the output format of OpenSSH `-E md5`.                     |
| `keys[].key_length`         | `int`    | The effective key length in bits (RSA modulus, ECDSA curve, Ed25519). | No direct OCSF or OTel mapping. gohai convention: `key_length`. The unit is bits, the same as OpenSSH's `-b` flag and the `ssh-keygen -l` output. |

## Platform support

| Platform | Supported                                                   |
| -------- | ----------------------------------------------------------- |
| Linux    | ✅                                                          |
| macOS    | ✅ (same `/etc/ssh/` path. macOS has OpenSSH)               |
| Other    | Empty key list (no files found means no keys in the output) |

## Example output

### RHEL/Fedora host with all three key types

```json
{
  "ssh": {
    "keys": [
      {
        "type": "ssh-rsa",
        "fingerprint_sha256": "SHA256:abc123...",
        "fingerprint_md5": "a1:b2:c3:d4:e5:f6:a7:b8:c9:d0:e1:f2:a3:b4:c5:d6",
        "key_length": 3072
      },
      {
        "type": "ecdsa-sha2-nistp256",
        "fingerprint_sha256": "SHA256:xyz789...",
        "fingerprint_md5": "11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff:00",
        "key_length": 256
      },
      {
        "type": "ssh-ed25519",
        "fingerprint_sha256": "SHA256:uvw456...",
        "fingerprint_md5": "00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff",
        "key_length": 256
      }
    ]
  }
}
```

### Host with no host keys present

```json
{
  "ssh": {
    "keys": []
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

g, _ := gohai.New(gohai.WithCollectors("ssh"))
facts, _ := g.Collect(context.Background())

for _, key := range facts.SSH.Keys {
    fmt.Printf("%s  %s  (%d bits)\n", key.Type, key.FingerprintSHA256, key.KeyLength)
}
```

## Enable/disable

```bash
gohai collect --collector.ssh       # enable (opt-in)
gohai collect --no-collector.ssh    # disable
```

This collector is opt-in (`DefaultEnabled: false`). Host key material is
security-sensitive, and a general inventory does not need it.

## Dependencies

None.

## Data sources

On Linux and macOS, the steps are the same. `/etc/ssh/` is the standard path on
both platforms. Ohai's `ssh_host_key.rb` checks `/etc/sshd_config` first, and
then uses the well-known paths.

1. The collector checks the three well-known host public key files in this
   order: `/etc/ssh/ssh_host_rsa_key.pub`, `/etc/ssh/ssh_host_ecdsa_key.pub`,
   `/etc/ssh/ssh_host_ed25519_key.pub`. It skips a file that does not exist, and
   gives no message. A host without one key type is normal, for example when RSA
   is disabled in the sshd config.
2. For each file that it finds, the collector parses the single-line format:
   `<type> <base64-wire-blob> [comment]`. It splits the line at the whitespace.
   The line must have at least two fields. If a file has fewer fields, or the
   base64 does not decode, the collector returns an error.
3. The collector calculates the SHA-256 and MD5 fingerprints over the raw wire
   bytes. These bytes are in RFC 4251 format: the blob decoded from the base64
   field. The SHA-256 format is `SHA256:<base64-without-trailing-=>`. The MD5
   format is hex pairs with colons between them. Both formats are exactly the
   same as the `ssh-keygen -l` output.
4. The collector gets the key length from the key type:
   - `ssh-rsa`: it parses the wire blob (algorithm name, exponent, modulus). It
     removes leading zero-byte sign padding and returns `len(modulus) * 8`.
   - `ecdsa-sha2-nistp256/384/521`: it reads the curve label in the type string
     (256, 384 or 521).
   - `ssh-ed25519`: always 256.
   - Unknown types: key length 0.

Note: Ohai reads `sshd_config` first to find the configured `HostKey` paths.
Then it uses the well-known paths. gohai reads only the well-known paths, which
is correct for all standard deployments. If a `HostKey` directive points to a
path that is not standard, gohai does not report that key.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read the files in
  `/etc/ssh/`. Production uses `osfs`. Tests use `memfs`.
- The Go standard library `crypto/sha256`, `crypto/md5` and `encoding/binary` to
  calculate the fingerprints and parse the wire format.
- [`golang.org/x/crypto/ssh`](https://pkg.go.dev/golang.org/x/crypto/ssh), in
  tests only, to make real key fixtures.
