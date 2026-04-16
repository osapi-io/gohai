# FIPS

> **Status:** Implemented ✅

## Description

Reports whether the system is running in FIPS 140 mode. FIPS 140 is the U.S.
federal cryptographic module standard (current revision: FIPS 140-3, which
superseded FIPS 140-2 in 2019). When enabled, the system restricts cryptographic
primitives to the approved set and fails closed rather than falling back to
non-compliant algorithms.

The collector reports two related signals:

- `kernel.enabled` — the kernel-level flag from `/proc/sys/crypto/fips_enabled`,
  set by the `fips=1` kernel command-line parameter at boot. This is the classic
  140-2-era signal.
- `policy` — the user-space crypto policy on hosts that ship the
  `crypto-policies` framework (RHEL 8+, Fedora 30+, CentOS Stream, Amazon Linux
  2023). On FIPS 140-3 systems, an operator can flip the policy post-boot with
  `update-crypto-policies --set DEFAULT` **without** changing the kernel flag,
  leaving the kernel in FIPS mode while OpenSSL, GnuTLS, etc. no longer enforce
  FIPS-approved algorithms. This field catches that drift. On hosts without the
  framework the field is omitted.

Consumers use this to:

- Confirm FIPS posture across a fleet that is supposed to be compliant (FedRAMP,
  DoD SRG, HIPAA).
- Gate features that behave differently under FIPS — for example, TLS libraries
  and SSH clients that disable non-approved ciphers when the kernel flag is set.
- Detect drift: a FIPS-provisioned host whose crypto policy got toggled off is a
  real incident (kernel says "FIPS", user-space says "not really"), and
  `kernel.enabled=true` + `policy.fips_effective=false` is the signal that flags
  it.

The runtime flag does **not** distinguish which revision of FIPS 140 the
kernel's crypto module was validated against — that's a property of the module
build (RHEL 8 targets 140-2; RHEL 9 targets 140-3). Consumers needing the
validated revision should correlate with `platform`/`kernel`.

## Collected Fields

| Field                   | Type   | Description                                                                    | Schema mapping            |
| ----------------------- | ------ | ------------------------------------------------------------------------------ | ------------------------- |
| `kernel.enabled`        | `bool` | `true` if the kernel flag `/proc/sys/crypto/fips_enabled` is `1`.              | No direct schema mapping. |
| `policy.name`           | string | Active crypto policy (e.g. `FIPS`, `FIPS:OSPP`, `DEFAULT`). Omitted if absent. | No direct schema mapping. |
| `policy.fips_effective` | `bool` | `true` if the policy name starts with `FIPS`.                                  | No direct schema mapping. |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | `nil`     |
| Other    | `nil`     |

macOS is not covered because Apple's CoreCrypto module has no runtime toggle
equivalent to Linux's kernel flag; FIPS 140 validation there is a property of
the module and must be looked up via Apple's certificate list separately.

## Example Output

### FIPS-enabled RHEL 9 host (kernel + policy both on)

```json
{
  "fips": {
    "kernel": { "enabled": true },
    "policy": { "name": "FIPS", "fips_effective": true }
  }
}
```

### Drift: kernel boots FIPS, policy switched to DEFAULT

```json
{
  "fips": {
    "kernel": { "enabled": true },
    "policy": { "name": "DEFAULT", "fips_effective": false }
  }
}
```

### Non-FIPS Debian/Ubuntu (no crypto-policies framework)

```json
{
  "fips": {
    "kernel": { "enabled": false }
  }
}
```

### macOS

`facts.Fips` is `nil`.

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("fips"))
facts, _ := g.Collect(context.Background())

f := facts.Fips
if f.Kernel.Enabled && (f.Policy == nil || f.Policy.FIPSEffective) {
    fmt.Println("FIPS mode effective")
}
```

## Enable/Disable

```bash
gohai --collector.fips      # enable (default)
gohai --no-collector.fips   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. Read `/proc/sys/crypto/fips_enabled` via the injected `avfs.VFS`. Trimmed
   content `"1"` → `kernel.enabled = true`; anything else → `false`. Missing
   file (kernel without the flag) leaves `kernel` nil.
2. Read `/etc/crypto-policies/config` via the same VFS. The trimmed first line
   is the policy name. `policy.fips_effective` is set when the name starts with
   `FIPS` (case-sensitive — matches the file's convention). Missing file leaves
   `policy` nil.

Mirrors Ohai's `fips.rb` kernel-flag signal and extends it with the
`crypto-policies` probe — which catches FIPS 140-3 post-boot drift that Ohai's
OpenSSL-binding approach would miss.

On macOS the collector returns no data. Apple's CoreCrypto module has no runtime
FIPS toggle equivalent to Linux's kernel flag; FIPS 140 validation is a property
of the compiled module and must be looked up via Apple's certificate list
separately. Matches Ohai's decision to skip macOS.

## Backing library

- Go stdlib (`os`, `io`) — no third-party dependency.
