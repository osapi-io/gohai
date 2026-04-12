# Shard

> **Status:** Implemented ✅

## Description

Derives a deterministic shard seed from the host's stable identity. Matches
Ohai's `shard` plugin semantics: a SHA-256 hash combining machine_id + hostname
so the same host always maps to the same shard, but different hosts distribute
evenly across any number of buckets.

Consumers use this to:

- Stagger scheduled work across a fleet (`shard % 60` for minute-of-hour,
  `shard % 7` for day-of-week, etc.) without a per-host schedule file.
- Distribute work across parallel pipelines without central coordination.
- Pick a canary host deterministically (`shard == 0 mod N`).

## Collected Fields

| Field  | Type   | Description                                                                                                                     | OCSF mapping                                           |
| ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| `seed` | string | Hex-encoded SHA-256 of `"<machine_id>:<hostname>"`. 64 characters. Stable across reboots when the host has a stable machine-id. | No OCSF equivalent — shard is a gohai/Ohai convention. |

## Platform Support

| Platform | Supported                                                               |
| -------- | ----------------------------------------------------------------------- |
| Linux    | ✅ (inputs: `/etc/machine-id` or `/var/lib/dbus/machine-id` + hostname) |
| macOS    | ✅ (inputs: IOPlatformUUID + hostname)                                  |

## Example Output

```json
{
  "shard": {
    "seed": "7e1a9d5c94b0b3a6b2e3f8c7d1a9e5b4c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7"
  }
}
```

## SDK Usage

```go
import (
    "context"
    "encoding/hex"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("shard"))
facts, _ := g.Collect(context.Background())

// Use first 8 hex chars as a uint32 for modular arithmetic.
raw, _ := hex.DecodeString(facts.Shard.Seed[:8])
bucket := (uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])) % 60
fmt.Printf("this host runs at minute %d of every hour\n", bucket)
```

## Enable/Disable

```bash
gohai --collector.shard      # enable (default)
gohai --no-collector.shard   # disable
```

## Dependencies

None at the collector-registry level. **Conceptually** depends on stable
machine_id + hostname — consumers on hosts with neither should treat the derived
seed as unreliable.

## Data Sources

| Platform | What we read                                                                              | Ohai plugin                                                                                                  | Alignment                                                                                                                                           |
| -------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `/etc/machine-id` (fallback `/var/lib/dbus/machine-id`) + `os.Hostname`; SHA-256 of both. | [`shard.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/shard.rb) — SHA-256 of a similar input. | **Equivalent concept.** Ohai's input combination has varied across versions; ours pins it to a clear `machine_id:hostname` spec for predictability. |
| macOS    | IOPlatformUUID (via gopsutil host.Info) + `os.Hostname`; SHA-256 of both.                 | Ohai's darwin path uses `dmi`/`hardware` identifiers.                                                        | **Equivalent outcome** — stable hardware UUID + hostname; different exact source.                                                                   |

**Known gaps:** None. The seed format (hex-encoded SHA-256) is a deliberate
public contract — consumers can split, hash, modulo, etc.

## Backing library

- Go stdlib (`crypto/sha256`, `encoding/hex`, `os`) — no third-party dependency.
