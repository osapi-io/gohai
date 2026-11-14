# FIPS

> **Status:** Implemented ✅

## Description

The collector reports if the system runs in FIPS 140 mode. FIPS 140 is the U.S.
federal standard for cryptographic modules. The current revision is FIPS 140-3.
It replaced FIPS 140-2 in 2019. In FIPS mode, the system permits only the
approved cryptographic primitives. If an operation needs an algorithm that is
not approved, the operation fails. The system does not use the algorithm.

The collector reports two related signals:

- `kernel.enabled`. The kernel flag from `/proc/sys/crypto/fips_enabled`. The
  `fips=1` kernel command-line parameter sets it at boot. This is the signal
  from the time of FIPS 140-2.
- `policy`. The user-space crypto policy, on hosts that have the
  `crypto-policies` framework (RHEL 8+, Fedora 30+, CentOS Stream, Amazon Linux
  2023). On FIPS 140-3 systems, an operator can change the policy after boot
  with `update-crypto-policies --set DEFAULT` **without** a change to the kernel
  flag. Then the kernel stays in FIPS mode, but OpenSSL, GnuTLS and other
  libraries do not enforce the FIPS-approved algorithms. This field shows that
  drift. On hosts without the framework, the field is not in the output.

Consumers use this data to:

- Confirm the FIPS state of a fleet that must be compliant (FedRAMP, DoD SRG,
  HIPAA).
- Control features that operate differently in FIPS mode. For example, TLS
  libraries and SSH clients disable the ciphers that are not approved when the
  kernel flag is set.
- Find drift. If someone turns off the crypto policy on a FIPS host, that is a
  real incident: the kernel is in FIPS mode, but user space is not. The signal
  for this is `kernel.enabled=true` with `policy.fips_effective=false`.

The runtime flag does **not** show the revision of FIPS 140 that the kernel's
crypto module is validated against. That is a property of the module build. RHEL
8 targets 140-2. RHEL 9 targets 140-3. If a consumer needs the validated
revision, it must compare this data with `platform` and `kernel`.

## Collected fields

| Field                   | Type   | Description                                                                                                         | Schema mapping            |
| ----------------------- | ------ | ------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `kernel.enabled`        | `bool` | `true` if the kernel flag `/proc/sys/crypto/fips_enabled` is `1`.                                                   | No direct schema mapping. |
| `policy.name`           | string | The active crypto policy, for example `FIPS`, `FIPS:OSPP` or `DEFAULT`. Not in the output if the policy is missing. | No direct schema mapping. |
| `policy.fips_effective` | `bool` | `true` if the policy name starts with `FIPS`.                                                                       | No direct schema mapping. |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | `nil`     |
| Other    | `nil`     |

The collector does not support macOS. Apple's CoreCrypto module has no runtime
switch that is equivalent to the Linux kernel flag. On macOS, FIPS 140
validation is a property of the module. You must find it in Apple's certificate
list.

## Example output

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

## SDK usage

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

## Enable/disable

```bash
gohai --collector.fips      # enable (default)
gohai --no-collector.fips   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. The collector reads `/proc/sys/crypto/fips_enabled` through the injected
   `avfs.VFS`. If the trimmed content is `"1"`, `kernel.enabled` is `true`. For
   all other content, it is `false`. If the file is missing (a kernel without
   the flag), `kernel` stays nil.
2. The collector reads `/etc/crypto-policies/config` through the same VFS. The
   trimmed first line is the policy name. `policy.fips_effective` is `true` when
   the name starts with `FIPS`. The check is case-sensitive, the same as the
   convention of the file. If the file is missing, `policy` stays nil.

The kernel-flag signal is the same as in Ohai's `fips.rb`. gohai adds the
`crypto-policies` check. This check finds the FIPS 140-3 drift after boot. Ohai
reads the OpenSSL binding, so it does not find this drift.

On macOS, the collector returns no data. Apple's CoreCrypto module has no
runtime FIPS switch that is equivalent to the Linux kernel flag. FIPS 140
validation is a property of the compiled module. You must find it in Apple's
certificate list. Ohai also does not support macOS.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for the file reads,
  through the injected `avfs.VFS`, and the Go standard library.
