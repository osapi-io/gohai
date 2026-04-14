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

| Field                   | Type       | Description                               | Schema mapping                 |
| ----------------------- | ---------- | ----------------------------------------- | ------------------------------ |
| `instance_id`           | `string`   | ECS instance ID.                          | OTel `cloud.resource_id`       |
| `instance_name`         | `string`   | Instance display name.                    | OTel `host.name`               |
| `instance_type`         | `string`   | ECS instance type (e.g. `ecs.g6.large`).  | OTel `host.type`               |
| `hostname`              | `string`   | Hostname.                                 | OCSF `device.hostname`         |
| `image_id`              | `string`   | Source image ID.                          | OTel `host.image.id`           |
| `serial_number`         | `string`   | Instance serial.                          | No direct schema mapping.      |
| `network_type`          | `string`   | `vpc` / `classic`.                        | No direct schema mapping.      |
| `region`                | `string`   | Region (e.g. `cn-hangzhou`).              | OTel `cloud.region`            |
| `zone`                  | `string`   | Availability zone (e.g. `cn-hangzhou-b`). | OTel `cloud.availability_zone` |
| `mac`                   | `string`   | Primary interface MAC.                    | OCSF `network_interface.mac`   |
| `private_ipv4`          | `string`   | Primary private IPv4.                     | No direct schema mapping.      |
| `public_ipv4`           | `string`   | EIP (elastic IP) when attached.           | No direct schema mapping.      |
| `vpc_id`                | `string`   | VPC ID.                                   | No direct schema mapping.      |
| `vpc_cidr_block`        | `string`   | VPC CIDR.                                 | No direct schema mapping.      |
| `vswitch_id`            | `string`   | vSwitch ID (VPC subnet).                  | No direct schema mapping.      |
| `vswitch_cidr_block`    | `string`   | vSwitch CIDR.                             | No direct schema mapping.      |
| `dns_nameservers`       | `[]string` | Configured DNS resolvers.                 | No direct schema mapping.      |
| `ntp_servers`           | `[]string` | Configured NTP servers.                   | No direct schema mapping.      |
| `max_bandwidth_ingress` | `string`   | Ingress bandwidth cap (bytes/sec).        | No direct schema mapping.      |
| `max_bandwidth_egress`  | `string`   | Egress bandwidth cap (bytes/sec).         | No direct schema mapping.      |
| `ram_role_name`         | `string`   | Attached RAM (role) name.                 | No direct schema mapping.      |

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

1. **DMI gate:** `dmi.Product.Vendor` contains `"Alibaba"`. Matches Ohai's
   `has_ali_dmi?`.
2. **Endpoint:** `http://100.100.100.200/2016-01-01/meta-data/<path>`. Each
   field is fetched as its own GET against the curated path list. Ohai walks the
   entire tree recursively; we fetch the specific known fields to produce a
   typed flat struct — same field coverage, fewer requests.
3. **Timeout:** 2 seconds per request.
4. **Failure handling:** first-probe (`/meta-data/hostname`) failure returns
   `(nil, nil)`. Subsequent path failures are tolerated and leave their field
   zero-valued.
5. **Transformation:** space-separated DNS and NTP lists are split into Go
   slices.

Mirrors Ohai's `Ohai::Mixin::AlibabaMetadata` collection approach and Alibaba
field coverage.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
