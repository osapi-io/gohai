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

| Field                             | Type                          | Description                                                          | Schema mapping                 |
| --------------------------------- | ----------------------------- | -------------------------------------------------------------------- | ------------------------------ |
| `instance_id`                     | `string`                      | ECS instance ID.                                                     | OTel `cloud.resource_id`       |
| `instance_name`                   | `string`                      | Instance display name.                                               | OTel `host.name`               |
| `instance_type`                   | `string`                      | ECS instance type (e.g. `ecs.g6.large`).                             | OTel `host.type`               |
| `hostname`                        | `string`                      | Hostname.                                                            | OCSF `device.hostname`         |
| `image_id`                        | `string`                      | Source image ID.                                                     | OTel `host.image.id`           |
| `serial_number`                   | `string`                      | Instance serial.                                                     | No direct schema mapping.      |
| `network_type`                    | `string`                      | `vpc` / `classic`.                                                   | No direct schema mapping.      |
| `region`                          | `string`                      | Region (e.g. `cn-hangzhou`).                                         | OTel `cloud.region`            |
| `zone`                            | `string`                      | Availability zone (e.g. `cn-hangzhou-b`).                            | OTel `cloud.availability_zone` |
| `mac`                             | `string`                      | Primary interface MAC.                                               | OCSF `network_interface.mac`   |
| `private_ipv4`                    | `string`                      | Primary private IPv4.                                                | No direct schema mapping.      |
| `public_ipv4`                     | `string`                      | EIP (elastic IP) when attached.                                      | No direct schema mapping.      |
| `vpc_id`                          | `string`                      | VPC ID.                                                              | No direct schema mapping.      |
| `vpc_cidr_block`                  | `string`                      | VPC CIDR.                                                            | No direct schema mapping.      |
| `vswitch_id`                      | `string`                      | vSwitch ID (VPC subnet).                                             | No direct schema mapping.      |
| `vswitch_cidr_block`              | `string`                      | vSwitch CIDR.                                                        | No direct schema mapping.      |
| `dns_nameservers`                 | `[]string`                    | Configured DNS resolvers.                                            | No direct schema mapping.      |
| `ntp_servers`                     | `[]string`                    | Configured NTP servers.                                              | No direct schema mapping.      |
| `max_bandwidth_ingress`           | `int64`                       | Ingress bandwidth cap (bytes/sec).                                   | No direct schema mapping.      |
| `max_bandwidth_egress`            | `int64`                       | Egress bandwidth cap (bytes/sec).                                    | No direct schema mapping.      |
| `ram_role_name`                   | `string`                      | Attached RAM (role) name.                                            | No direct schema mapping.      |
| `owner_account_id`                | `string`                      | Alibaba Cloud account that owns the instance.                        | No direct schema mapping.      |
| `source_address`                  | `string`                      | Package-manager mirror URL Alibaba surfaces for the region.          | No direct schema mapping.      |
| `virtualization_solution`         | `string`                      | Hypervisor family (e.g. `ECS Virt`).                                 | No direct schema mapping.      |
| `virtualization_solution_version` | `string`                      | Build version of the hypervisor.                                     | No direct schema mapping.      |
| `spot_termination_time`           | `string`                      | ISO-8601 warning time for spot termination (spot instances only).    | No direct schema mapping.      |
| `network_interfaces`              | `map[string]NetworkInterface` | All ENIs keyed by MAC. See below.                                    | OCSF `network_interface`       |
| `disks`                           | `map[string]Disk`             | Attached disks keyed by serial number. See below.                    | No direct schema mapping.      |
| `marketplace`                     | `*Marketplace`                | Marketplace image billing info (only on Marketplace-sourced images). | No direct schema mapping.      |
| `tags`                            | `map[string]string`           | User-defined instance tags.                                          | No direct schema mapping.      |

### NetworkInterface

| Field                     | Type       | Description                        |
| ------------------------- | ---------- | ---------------------------------- |
| `network_interface_id`    | `string`   | ENI ID (`eni-...`).                |
| `primary_ip_address`      | `string`   | Primary private IPv4.              |
| `private_ipv4s`           | `[]string` | All private IPv4s on this ENI.     |
| `ipv4_prefixes`           | `[]string` | Delegated IPv4 prefixes.           |
| `netmask`                 | `string`   | Subnet netmask for the primary IP. |
| `gateway`                 | `string`   | Default gateway for this ENI.      |
| `vpc_id`                  | `string`   | VPC ID.                            |
| `vpc_cidr_block`          | `string`   | VPC IPv4 CIDR.                     |
| `vpc_ipv6_cidr_blocks`    | `[]string` | VPC IPv6 CIDRs.                    |
| `vswitch_id`              | `string`   | vSwitch (subnet) ID.               |
| `vswitch_cidr_block`      | `string`   | vSwitch IPv4 CIDR.                 |
| `vswitch_ipv6_cidr_block` | `string`   | vSwitch IPv6 CIDR.                 |
| `ipv6s`                   | `[]string` | All IPv6 addresses on this ENI.    |
| `ipv6_prefixes`           | `[]string` | Delegated IPv6 prefixes.           |
| `ipv6_gateway`            | `string`   | IPv6 default gateway.              |

### Disk

| Field  | Type     | Description                  |
| ------ | -------- | ---------------------------- |
| `id`   | `string` | Disk resource ID (`d-...`).  |
| `name` | `string` | Operator-assigned disk name. |

### Marketplace

| Field          | Type     | Description                      |
| -------------- | -------- | -------------------------------- |
| `product_code` | `string` | Marketplace product code.        |
| `charge_type`  | `string` | Billing method (`PrePaid` etc.). |

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
   `mixin/alibaba_metadata.rb`.
4. **`/user-data` excluded** (root-level only) — matches Ohai's explicit skip to
   avoid surfacing cloud-init scripts that may contain credentials.
5. **Leaf parsing:** each leaf is tried as JSON first; on parse failure the raw
   text is kept. Matches Ohai's `parse_json` fallback pattern.
6. **Key sanitization:** dashes and slashes in path segments become underscores
   in map keys; trailing underscores are stripped. Matches Ohai's
   `sanitize_key`.
7. **Transformation:** the walked tree is projected onto the typed `Info` —
   identity / location / network fields flattened from `meta-data/`, plus the
   nested sub-objects `instance/` (including `spot/termination-time`),
   `image/market-place/`, `tags/instance/`, `disks/<serial>/`, and
   `network/interfaces/macs/<mac>/`. All surfaced fields are typed; no `Raw`
   escape hatch.
8. **Deliberately excluded:** `ram/security-credentials/<role>/` (short-lived
   AccessKey/SecurityToken) and `public-keys/<keypair>/openssh-key` — both
   security-sensitive and of no inventory value.
9. **User-Agent:** `gohai` (the cloudmetadata default) — Alibaba's metadata
   service filters some requests by UA.
10. **Timeout:** 6 seconds per request, matching Ohai's `read_timeout` +
    `keep_alive_timeout`.
11. **Failure handling:** first call (the root listing) failing → `(nil, nil)`.
    Per-path failures during the walk are tolerated and leave those fields empty
    in the typed output.

Mirrors Ohai's `Ohai::Mixin::AlibabaMetadata` methodology: same endpoint, same
recursive walk, same key sanitization, same `/user-data` scrub, same User-Agent
pattern, same 6s timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
