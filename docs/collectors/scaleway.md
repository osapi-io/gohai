# Scaleway

> **Status:** Implemented ✅

## Description

Collects Scaleway instance metadata by hitting the link-local server at
`http://169.254.42.42/conf?format=json` (Scaleway uses `169.254.42.42`, not the
more common `169.254.169.254`). `facts.Scaleway != nil` is the detection signal.
Non-Scaleway hosts drop silently.

Detection uses `/proc/cmdline` for the substring `"scaleway"` — Scaleway's boot
environment always announces itself in the kernel command line.

## Collected Fields

| Field               | Type          | Description                                                                          | Schema mapping                 |
| ------------------- | ------------- | ------------------------------------------------------------------------------------ | ------------------------------ |
| `id`                | `string`      | Instance UUID.                                                                       | OTel `cloud.resource_id`       |
| `name`              | `string`      | Instance display name.                                                               | OTel `host.name`               |
| `hostname`          | `string`      | Instance hostname.                                                                   | OCSF `device.hostname`         |
| `organization`      | `string`      | Scaleway organization UUID.                                                          | OTel `cloud.account.id`        |
| `project`           | `string`      | Scaleway project UUID.                                                               | No direct schema mapping.      |
| `commercial_type`   | `string`      | Instance type (e.g. `DEV1-S`).                                                       | OTel `host.type`               |
| `tags`              | `[]string`    | Instance tags.                                                                       | No direct schema mapping.      |
| `state_detail`      | `string`      | Instance lifecycle state.                                                            | No direct schema mapping.      |
| `public_ip`         | `string`      | Public IPv4.                                                                         | No direct schema mapping.      |
| `public_ip_id`      | `string`      | Public IP resource ID.                                                               | No direct schema mapping.      |
| `public_ip_dynamic` | `bool`        | Whether the public IP is dynamic.                                                    | No direct schema mapping.      |
| `private_ip`        | `string`      | Private IPv4.                                                                        | No direct schema mapping.      |
| `ipv6_address`      | `string`      | Public IPv6.                                                                         | No direct schema mapping.      |
| `ipv6_netmask`      | `string`      | IPv6 prefix.                                                                         | No direct schema mapping.      |
| `ipv6_gateway`      | `string`      | IPv6 gateway.                                                                        | No direct schema mapping.      |
| `zone`              | `string`      | Availability zone (e.g. `fr-par-1`).                                                 | OTel `cloud.availability_zone` |
| `platform_id`       | `string`      | Scaleway hardware platform identifier.                                               | No direct schema mapping.      |
| `ssh_public_keys`   | `[]string`    | SSH public keys attached to the account.                                             | No direct schema mapping.      |
| `volumes`           | `[]Volume`    | Attached volumes.                                                                    | No direct schema mapping.      |
| `timezone`          | `string`      | Instance timezone from the IMDS (e.g. `Europe/Paris`).                               | No direct schema mapping.      |
| `bootscript`        | `*Bootscript` | Legacy boot configuration (deprecated — modern instances use local boot). See below. | No direct schema mapping.      |

### Volume

| Field         | Type     | Description                           |
| ------------- | -------- | ------------------------------------- |
| `id`          | `string` | Volume UUID.                          |
| `name`        | `string` | Volume name.                          |
| `volume_type` | `string` | Volume class (e.g. `b_ssd`, `l_ssd`). |
| `size`        | `int64`  | Volume size in bytes.                 |
| `export_uri`  | `string` | NBD export URI (Scaleway-internal).   |

### Bootscript

| Field          | Type     | Description                            |
| -------------- | -------- | -------------------------------------- |
| `id`           | `string` | Bootscript ID.                         |
| `title`        | `string` | Bootscript display name.               |
| `architecture` | `string` | Target architecture (e.g. `x86_64`).   |
| `kernel`       | `string` | Kernel download URL.                   |
| `initrd`       | `string` | Initrd download URL.                   |
| `bootcmdargs`  | `string` | Boot command-line arguments.           |
| `organization` | `string` | Organization that owns the bootscript. |
| `public`       | `bool`   | Whether the bootscript is public.      |

## Platform Support

| Platform | Supported                             |
| -------- | ------------------------------------- |
| Linux    | ✅                                    |
| macOS    | ✅ (only meaningful on a Scaleway VM) |
| Other    | ✅ (only meaningful on a Scaleway VM) |

## Example Output

```json
{
  "scaleway": {
    "id": "sc-abc",
    "name": "prod-1",
    "hostname": "prod-1",
    "commercial_type": "DEV1-S",
    "public_ip": "51.0.0.1",
    "private_ip": "10.64.0.1",
    "zone": "fr-par-1"
  }
}
```

## SDK Usage

```go
import (
    "context"
    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("scaleway"))
facts, _ := g.Collect(context.Background())

if facts.Scaleway != nil {
    fmt.Println("running in", facts.Scaleway.Zone)
}
```

## Enable/Disable

```bash
gohai --collector.scaleway      # enable (opt-in)
gohai --no-collector.scaleway   # disable (default)
gohai --category=cloud          # pulls this + all cloud collectors
```

## Dependencies

None. Detection uses `/proc/cmdline` rather than DMI — Scaleway exposes no
stable DMI signature, but their kernel cmdline always contains `"scaleway"`.

## Data Sources

1. **cmdline gate:** read `/proc/cmdline` and check for the substring
   `"scaleway"` (case-insensitive). Missing file or absent substring
   short-circuits with `(nil, nil)`. Matches Ohai's `has_scaleway_cmdline?`.
2. **Endpoint:** `http://169.254.42.42/conf?format=json` — a single JSON
   document.
3. **User-Agent:** `gohai` (the cloudmetadata default).
4. **Timeout:** 6 seconds — matches Ohai's `read_timeout` in
   `mixin/scaleway_metadata.rb`.
5. **Failure handling:** any fetch failure (transport, non-2xx, body read)
   returns `(nil, nil)` so `Facts.Scaleway` drops silently. Only malformed JSON
   in the response surfaces as an error.
6. **Transformation:** nested `public_ip`, `ipv6`, and `location` objects are
   flattened onto the top-level Info; `ssh_public_keys` is flattened from an
   array of `{key}` objects to a plain string array (blank keys skipped); the
   oddly-shaped index-keyed `volumes` map is converted to a slice; `bootscript`
   is retained as a typed sub-struct when present (deprecated on modern
   instances — may be null); `timezone` is lifted as a simple string. All
   surfaced fields are typed — no `Raw` escape hatch.

Mirrors Ohai's `Ohai::Mixin::ScalewayMetadata` methodology — same endpoint, same
single JSON GET, same 6s timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
