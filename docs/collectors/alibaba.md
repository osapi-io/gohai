# Alibaba

> **Status:** Implemented ✅

## Description

Collects Alibaba Cloud ECS instance metadata from the link-local server at
`http://100.100.100.200/2016-01-01/` — Alibaba uses `100.100.100.200` rather
than the more common `169.254.169.254` link-local range. `facts.Alibaba != nil`
is the detection signal.

Detection is gated on DMI `sys_vendor` containing `"Alibaba"` — matches Ohai's
`has_ali_dmi?`.

## Collected Fields

| Field                   | Type             | Description                                                                                                                                                | Schema mapping                 |
| ----------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `instance_id`           | `string`         | ECS instance ID.                                                                                                                                           | OTel `cloud.resource_id`       |
| `instance_name`         | `string`         | Instance display name.                                                                                                                                     | OTel `host.name`               |
| `instance_type`         | `string`         | ECS instance type (e.g. `ecs.g6.large`).                                                                                                                   | OTel `host.type`               |
| `hostname`              | `string`         | Hostname.                                                                                                                                                  | OCSF `device.hostname`         |
| `image_id`              | `string`         | Source image ID.                                                                                                                                           | OTel `host.image.id`           |
| `serial_number`         | `string`         | Instance serial.                                                                                                                                           | No direct schema mapping.      |
| `network_type`          | `string`         | `vpc` / `classic`.                                                                                                                                         | No direct schema mapping.      |
| `region`                | `string`         | Region (e.g. `cn-hangzhou`).                                                                                                                               | OTel `cloud.region`            |
| `zone`                  | `string`         | Availability zone (e.g. `cn-hangzhou-b`).                                                                                                                  | OTel `cloud.availability_zone` |
| `mac`                   | `string`         | Primary interface MAC.                                                                                                                                     | OCSF `network_interface.mac`   |
| `private_ipv4`          | `string`         | Primary private IPv4.                                                                                                                                      | No direct schema mapping.      |
| `public_ipv4`           | `string`         | EIP (elastic IP) when attached.                                                                                                                            | No direct schema mapping.      |
| `vpc_id`                | `string`         | VPC ID.                                                                                                                                                    | No direct schema mapping.      |
| `vpc_cidr_block`        | `string`         | VPC CIDR.                                                                                                                                                  | No direct schema mapping.      |
| `vswitch_id`            | `string`         | vSwitch ID (VPC subnet).                                                                                                                                   | No direct schema mapping.      |
| `vswitch_cidr_block`    | `string`         | vSwitch CIDR.                                                                                                                                              | No direct schema mapping.      |
| `dns_nameservers`       | `[]string`       | Configured DNS resolvers.                                                                                                                                  | No direct schema mapping.      |
| `ntp_servers`           | `[]string`       | Configured NTP servers.                                                                                                                                    | No direct schema mapping.      |
| `max_bandwidth_ingress` | `int64`          | Ingress bandwidth cap (bytes/sec).                                                                                                                         | No direct schema mapping.      |
| `max_bandwidth_egress`  | `int64`          | Egress bandwidth cap (bytes/sec).                                                                                                                          | No direct schema mapping.      |
| `ram_role_name`         | `string`         | Attached RAM (role) name.                                                                                                                                  | No direct schema mapping.      |
| `raw`                   | `map[string]any` | Full metadata tree as walked recursively. Keys follow Ohai's sanitization (dashes/slashes → `_`). Use when you need a field not exposed as a typed member. | No direct schema mapping.      |

## Platform Support

| Platform | Supported                              |
| -------- | -------------------------------------- |
| Linux    | ✅                                     |
| macOS    | ✅ (only meaningful on an Alibaba ECS) |
| Other    | ✅ (only meaningful on an Alibaba ECS) |

## Example Output

```json
{
  "alibaba": {
    "instance_id": "i-abc",
    "region": "cn-hangzhou",
    "zone": "cn-hangzhou-b",
    "instance_type": "ecs.g6.large",
    "private_ipv4": "172.16.0.5",
    "public_ipv4": "47.1.2.3",
    "vpc_id": "vpc-1",
    "ram_role_name": "ecs-default"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("alibaba"))
facts, _ := g.Collect(context.Background())

if facts.Alibaba != nil {
    fmt.Println(facts.Alibaba.Region, facts.Alibaba.InstanceType)
}
```

## Enable/Disable

```bash
gohai --collector.alibaba      # enable (opt-in)
gohai --no-collector.alibaba   # disable (default)
gohai --category=cloud         # pulls this + all cloud collectors
```

## Dependencies

`dmi`. Alibaba writes `"Alibaba Cloud"` as `sys_vendor`. Fails open when dmi
wasn't run.

## Data Sources

1. **DMI gate:** `dmi.Product.Vendor` contains `"Alibaba"`. ghw reads
   `/sys/class/dmi/id/sys_vendor` into `Product.Vendor`, so this check is
   equivalent to Ohai's `has_ali_dmi?`.
2. **Endpoint:** `http://100.100.100.200/2016-01-01/` — note the non-standard
   `100.100.100.200` link-local address, not `169.254.169.254`.
3. **Recursive walk:** starting at `/`, the collector fetches each directory
   listing (newline-separated entries), recurses into entries ending in `/`, and
   fetches leaves as values. Matches Ohai's `fetch_metadata` algorithm in
   `mixin/alibaba_metadata.rb`. Forward-compatible — fields Alibaba adds later
   surface automatically under `Raw`.
4. **`/user-data` excluded** (root-level only) — matches Ohai's explicit skip to
   avoid surfacing cloud-init scripts that may contain credentials.
5. **Leaf parsing:** each leaf is tried as JSON first; on parse failure the raw
   text is kept. Matches Ohai's `parse_json` fallback pattern.
6. **Key sanitization:** dashes and slashes in path segments become underscores
   in map keys; trailing underscores are stripped. Matches Ohai's
   `sanitize_key`.
7. **User-Agent:** `gohai` (the cloudmetadata default) — Alibaba's metadata
   service filters some requests by UA.
8. **Timeout:** 6 seconds per request, matching Ohai's `read_timeout` +
   `keep_alive_timeout`.
9. **Failure handling:** first call (the root listing) failing → `(nil, nil)`.
   Per-path failures during the walk are tolerated and leave those keys absent
   from `Raw`.

Mirrors Ohai's `Ohai::Mixin::AlibabaMetadata` methodology: same endpoint, same
recursive walk, same key sanitization, same `/user-data` scrub, same User-Agent
pattern, same 6s timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
