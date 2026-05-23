# SSH

> **Status:** Implemented ✅

## Description

Reports SSH host public key fingerprints from `/etc/ssh/`. For each key type
present on the host (RSA, ECDSA, Ed25519), the collector reports the algorithm
type, SHA-256 fingerprint (OpenSSH format), MD5 fingerprint (colon-hex format),
and key length in bits.

Consumers use this to:

- Inventory host key fingerprints across a fleet for TOFU (trust-on-first-use)
  or key rotation tracking.
- Verify that expected key types are present and that deprecated types (e.g.
  DSA) have been removed.
- Cross-reference fingerprints against known-host databases or SIEM events.
- Detect key material drift after OS reinstalls or cloud image baking.

DSA keys are intentionally omitted — OpenSSH deprecated DSA in 7.0 (2015) and
removed support in 9.8. Collecting them would mislead consumers into thinking
DSA keys are still a valid security signal.

## Signals

The collector reports per-key-type signals:

- `keys[].fingerprint_sha256` — SHA-256 fingerprint in `SHA256:<base64>` format,
  matching `ssh-keygen -l -E sha256`. Use this for modern key comparisons and
  SIEM correlation.
- `keys[].fingerprint_md5` — MD5 fingerprint in `xx:xx:...` colon-hex format,
  matching `ssh-keygen -l -E md5`. Provided for legacy compatibility with older
  `known_hosts` tooling; prefer SHA-256 for new work.
- `keys[].key_length` — effective key strength in bits. For RSA this is the
  modulus bit-length; for ECDSA it is the curve size (256/384/521); for Ed25519
  it is always 256. Use this to flag keys below a minimum strength policy.

## Collected Fields

| Field                       | Type     | Description                                                       | Schema mapping                                                                                                                                           |
| --------------------------- | -------- | ----------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `keys[].type`               | `string` | Key algorithm as reported in the .pub file (e.g. `ssh-rsa`).      | No direct OCSF mapping. OTel has no SSH host key object. gohai convention: verbatim key type field from the public key file.                             |
| `keys[].fingerprint_sha256` | `string` | SHA-256 fingerprint in `SHA256:<base64>` format.                  | No direct OCSF or OTel mapping. gohai convention: `fingerprint_sha256`, matches OpenSSH `-E sha256` output format.                                       |
| `keys[].fingerprint_md5`    | `string` | MD5 fingerprint in `xx:xx:...` colon-hex format.                  | No direct OCSF or OTel mapping. gohai convention: `fingerprint_md5`, matches OpenSSH `-E md5` output format.                                             |
| `keys[].key_length`         | `int`    | Effective key length in bits (RSA modulus; ECDSA curve; Ed25519). | No direct OCSF or OTel mapping. gohai convention: `key_length` with implicit unit bits — consistent with OpenSSH's `-b` flag and `ssh-keygen -l` output. |

## Platform Support

| Platform | Supported                                           |
| -------- | --------------------------------------------------- |
| Linux    | ✅                                                  |
| macOS    | ✅ (same `/etc/ssh/` path — macOS ships OpenSSH)    |
| Other    | Empty key list (no files found = no keys in output) |

## Example Output

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

## SDK Usage

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

## Enable/Disable

```bash
gohai --collector.ssh       # enable (opt-in)
gohai --no-collector.ssh    # disable
```

This collector is opt-in (`DefaultEnabled: false`) because reading host key
material is security-sensitive and not needed for general-purpose inventory.

## Dependencies

None.

## Data Sources

On Linux and macOS (identical — `/etc/ssh/` is the canonical path on both
platforms; Ohai's `ssh_host_key.rb` checks `/etc/sshd_config` then falls back to
well-known paths):

1. Check the three well-known host public key files in order:
   `/etc/ssh/ssh_host_rsa_key.pub`, `/etc/ssh/ssh_host_ecdsa_key.pub`,
   `/etc/ssh/ssh_host_ed25519_key.pub`. Files that do not exist are silently
   skipped — a host without a particular key type (e.g. RSA disabled in sshd
   config) is normal.
2. For each file found, parse the single-line format:
   `<type> <base64-wire-blob> [comment]`. Split on whitespace; require at least
   two fields. A file with fewer fields or non-decodable base64 propagates as an
   error.
3. Compute SHA-256 and MD5 fingerprints over the raw wire bytes (RFC 4251 format
   — the blob decoded from the base64 field). Format SHA-256 as
   `SHA256:<base64-without-trailing-=>`; format MD5 as colon-separated hex
   pairs. Both formats match `ssh-keygen -l` output exactly.
4. Derive key length from the key type:
   - `ssh-rsa`: parse the wire blob (algorithm name, exponent, modulus) and
     return `len(modulus) * 8`, stripping any leading zero-byte sign padding.
   - `ecdsa-sha2-nistp256/384/521`: derive from curve label in the type string
     (256, 384, or 521).
   - `ssh-ed25519`: always 256.
   - Unknown types: key length 0.

Note: Ohai reads `sshd_config` first to discover which `HostKey` paths are
configured, then falls back to the well-known paths. gohai reads only the
well-known paths directly, which covers all standard deployments. Hosts with
`HostKey` directives pointing to non-standard paths will not have those keys
reported.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for the `/etc/ssh/` file reads.
- Go stdlib `crypto/sha256`, `crypto/md5`, `encoding/binary` for fingerprint
  computation and wire-format parsing.
- [`golang.org/x/crypto/ssh`](https://pkg.go.dev/golang.org/x/crypto/ssh) in
  tests only, for generating real key fixtures.
