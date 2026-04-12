# FIPS

> **Status:** Implemented ✅

## Description

Reports whether the kernel is running in FIPS mode. FIPS 140 is the U.S. federal
cryptographic module standard (current revision: FIPS 140-3, which superseded
FIPS 140-2 in 2019). When enabled, the kernel restricts cryptographic primitives
to the approved set and fails closed rather than falling back to non-compliant
algorithms. On Linux the state is exposed as a single 0/1 flag at
`/proc/sys/crypto/fips_enabled`, controlled by the `fips=1` kernel command-line
parameter at boot.

The runtime flag does **not** distinguish which revision of FIPS 140 the
kernel's crypto module was validated against — that's a property of the module
build (RHEL 8 targets 140-2; RHEL 9 targets 140-3; Ubuntu Pro FIPS, SUSE, and
others each ship their own validated modules). This collector reports mode
on/off only; consumers who need the validated revision must correlate with the
OS/kernel version reported by `platform`/`kernel`.

Consumers use this to:

- Confirm FIPS posture across a fleet that is supposed to be compliant (e.g.
  systems running under FedRAMP, DoD SRG, or HIPAA controls).
- Gate features that behave differently under FIPS — for example, TLS libraries
  and SSH clients that disable non-approved ciphers when `fips_enabled=1`.
- Detect drift: a host that was provisioned FIPS-on but came up with it off
  (missing kernel parameter, wrong kernel package) is worth paging on.

macOS is reported as `enabled: false` in all cases. Apple's CoreCrypto module is
FIPS 140-validated by Apple, but there is no equivalent of Linux's runtime
toggle — FIPS is a property of the module, not something the operator turns on
at boot — so a simple boolean isn't a useful signal there. Consumers who need
macOS FIPS posture should consult Apple's certificate list separately.

## Collected Fields

Top-level: `kernel` (object, matches Ohai's `fips.kernel` shape).

| Field            | Type   | Description                                   |
| ---------------- | ------ | --------------------------------------------- |
| `kernel.enabled` | `bool` | `true` if the kernel is running in FIPS mode. |

## Platform Support

| Platform | Source                          | Supported |
| -------- | ------------------------------- | --------- |
| Linux    | `/proc/sys/crypto/fips_enabled` | ✅        |
| macOS    | —                               | `nil`     |
| Other    | —                               | `nil`     |

On Linux, if `/proc/sys/crypto/fips_enabled` is missing (very old or custom
kernel without the crypto API compiled in), the collector reports
`enabled: false` rather than erroring — the file's absence is itself evidence
FIPS mode is not active.

## Example Output

### FIPS-enabled Linux host

```json
{
  "fips": {
    "kernel": {
      "enabled": true
    }
  }
}
```

### Standard Linux host

```json
{
  "fips": {
    "kernel": {
      "enabled": false
    }
  }
}
```

macOS: `facts.Fips` is `nil` (matches Ohai — no `:darwin` platform in Ohai's
plugin).

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("fips"))
facts, _ := g.Collect(context.Background())

if facts.Fips.Enabled {
    fmt.Println("kernel is in FIPS 140 mode")
}
```

## Enable/Disable

```bash
gohai --collector.fips      # enable (default)
gohai --no-collector.fips   # disable
```

## Dependencies

None — Tier 1 core collector with no upstream collector dependencies.

## Backing library

- Go stdlib (`os`, `io`) — no third-party dependency.

## Ohai parity

- Ohai plugin:
  [`lib/ohai/plugins/fips.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/fips.rb)
- Output shape: **identical** — `fips.kernel.enabled` (bool).
- Platform coverage: **identical** — Ohai provides on `:linux` and `:windows`
  only; we provide on Linux only (Windows planned) and return `nil` on macOS.
- Source divergence: Ohai reads `OpenSSL.fips_mode` from the Ruby OpenSSL
  binding; we read `/proc/sys/crypto/fips_enabled` directly. On Linux these
  track the same kernel flag (the OpenSSL FIPS mode is initialized from it), so
  the reported value matches in practice. We prefer the `/proc` read because it
  avoids linking or shelling to OpenSSL and doesn't depend on which OpenSSL the
  host has installed.
