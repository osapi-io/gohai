# DigitalOcean

> **Status:** Implemented ✅

## Description

Collects DigitalOcean droplet metadata by hitting the link-local server at
`http://169.254.169.254/metadata/v1.json`. Returns nil with no error when the
endpoint isn't reachable or the DMI signature doesn't match —
`facts.DigitalOcean != nil` is the detection signal.

Detection is gated on `dmi.BIOS.Vendor == "DigitalOcean"` (exact match, matches
Ohai's `has_do_dmi?`). `vendor_data` is dropped from the response because it
commonly contains cloud-init user scripts with credentials — matches Ohai's
explicit scrub.

## Collected Fields

| Field              | Type          | Description                                     | Schema mapping            |
| ------------------ | ------------- | ----------------------------------------------- | ------------------------- |
| `droplet_id`       | `int64`       | DigitalOcean droplet numeric ID.                | OTel `cloud.resource_id`  |
| `hostname`         | `string`      | Droplet hostname.                               | OCSF `device.hostname`    |
| `region`           | `string`      | DigitalOcean region (e.g. `nyc3`).              | OTel `cloud.region`       |
| `public_keys`      | `[]string`    | SSH public keys attached to the droplet.        | No direct schema mapping. |
| `tags`             | `[]string`    | User-defined tags.                              | No direct schema mapping. |
| `features`         | `[]string`    | DO feature flags enabled on this droplet.       | No direct schema mapping. |
| `floating_ip`      | `string`      | Floating IPv4 attached to the droplet (if any). | No direct schema mapping. |
| `reserved_ip`      | `string`      | DO's newer replacement for floating_ip (IPv4).  | No direct schema mapping. |
| `auth_key`         | `string`      | DO internal auth token; often empty.            | No direct schema mapping. |
| `user_data`        | `string`      | User-supplied cloud-init data.                  | No direct schema mapping. |
| `ipv4_nameservers` | `[]string`    | DNS resolvers.                                  | No direct schema mapping. |
| `interfaces`       | `[]Interface` | Attached VNICs — see below.                     | OCSF `network_interface`  |

### Interface

| Field          | Type     | Description                          |
| -------------- | -------- | ------------------------------------ |
| `scope`        | `string` | `"public"` or `"private"`.           |
| `mac`          | `string` | MAC address.                         |
| `type`         | `string` | Interface type.                      |
| `ipv4`         | `string` | Primary IPv4 address.                |
| `ipv4_netmask` | `string` | IPv4 netmask.                        |
| `ipv4_gateway` | `string` | IPv4 gateway.                        |
| `ipv6`         | `string` | Primary IPv6 address.                |
| `ipv6_cidr`    | `int`    | IPv6 CIDR prefix length.             |
| `ipv6_gateway` | `string` | IPv6 gateway.                        |
| `anchor_ipv4`  | `string` | DO anchor IP (inter-droplet fabric). |

## Platform Support

| Platform | Supported                            |
| -------- | ------------------------------------ |
| Linux    | ✅                                   |
| macOS    | ✅ (only meaningful on a DO droplet) |
| Other    | ✅ (only meaningful on a DO droplet) |

## Example Output

```json
{
  "digital_ocean": {
    "droplet_id": 123456,
    "hostname": "web-1",
    "region": "nyc3",
    "tags": ["prod"],
    "floating_ip": "138.1.2.3",
    "interfaces": [
      { "scope": "public", "ipv4": "138.4.5.6", "mac": "f6:a0:..." },
      { "scope": "private", "ipv4": "10.132.0.2", "mac": "f6:a0:..." }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("digital_ocean"))
facts, _ := g.Collect(context.Background())

if facts.DigitalOcean != nil {
    fmt.Println("running on DO,", facts.DigitalOcean.Region)
}
```

## Enable/Disable

```bash
gohai --collector.digital_ocean      # enable (opt-in)
gohai --no-collector.digital_ocean   # disable (default)
gohai --category=cloud               # pulls this + all cloud collectors
```

## Dependencies

`dmi`. DO writes `"DigitalOcean"` as `bios_vendor`; the collector gates the HTTP
call on an exact match. When `dmi` is absent from prior (user disabled it), the
collector fails open and tries the endpoint anyway.

## Data Sources

1. **DMI gate:** `dmi.BIOS.Vendor == "DigitalOcean"` (exact). Matches Ohai's
   `has_do_dmi?`.
2. **Endpoint:** `http://169.254.169.254/metadata/v1.json` — single JSON.
3. **User-Agent:** `gohai` (the cloudmetadata default).
4. **Timeout:** 6 seconds — matches Ohai's `read_timeout` in
   `mixin/do_metadata.rb`.
5. **Failure handling:** transport / 404 / other errors → `(nil, nil)`.
6. **Security scrub:** `vendor_data` dropped from the output (matches Ohai). May
   contain cloud-init scripts + credentials.

Mirrors Ohai's `Ohai::Mixin::DoMetadata` methodology — same endpoint, same DMI
gate, same `vendor_data` scrub, same 6s timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
