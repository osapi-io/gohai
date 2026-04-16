# OpenStack

> **Status:** Implemented ✅

## Description

Collects OpenStack (Nova) instance metadata from the link-local server at
`http://169.254.169.254/`. OpenStack intentionally emits an EC2-compatible
metadata service under `/latest/meta-data/...` and a richer Nova-specific
document at `/openstack/latest/meta_data.json` — gohai fetches both.
`facts.OpenStack != nil` is the detection signal.

Detection is gated on DMI `product.name` containing `"OpenStack"` — Ohai's
plugin gates on the virtualization plugin's openstack signal, which itself reads
DMI; we go directly to DMI for the same effect.

## Collected Fields

| Field               | Type                | Description                                                                                                                             | Schema mapping                 |
| ------------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `instance_id`       | `string`            | EC2-mirror instance ID.                                                                                                                 | OTel `cloud.resource_id`       |
| `instance_type`     | `string`            | Instance flavor (e.g. `m1.small`).                                                                                                      | OTel `host.type`               |
| `hostname`          | `string`            | Hostname.                                                                                                                               | OCSF `device.hostname`         |
| `local_hostname`    | `string`            | Internal DNS name.                                                                                                                      | No direct schema mapping.      |
| `public_hostname`   | `string`            | Public DNS name.                                                                                                                        | No direct schema mapping.      |
| `availability_zone` | `string`            | Nova availability zone.                                                                                                                 | OTel `cloud.availability_zone` |
| `local_ipv4`        | `string`            | Primary private IPv4.                                                                                                                   | No direct schema mapping.      |
| `public_ipv4`       | `string`            | Primary public IPv4.                                                                                                                    | No direct schema mapping.      |
| `security_groups`   | `[]string`          | Attached security groups.                                                                                                               | No direct schema mapping.      |
| `ami_id`            | `string`            | EC2-mirror AMI ID.                                                                                                                      | OTel `host.image.id`           |
| `kernel_id`         | `string`            | Boot kernel image.                                                                                                                      | No direct schema mapping.      |
| `ramdisk_id`        | `string`            | Boot ramdisk image.                                                                                                                     | No direct schema mapping.      |
| `reservation_id`    | `string`            | EC2-mirror reservation ID.                                                                                                              | No direct schema mapping.      |
| `name`              | `string`            | Nova `name` field.                                                                                                                      | OTel `host.name`               |
| `project_id`        | `string`            | Nova project ID.                                                                                                                        | OTel `cloud.account.id`        |
| `uuid`              | `string`            | Nova UUID.                                                                                                                              | No direct schema mapping.      |
| `meta_data`         | `map[string]string` | Nova `meta` key/values.                                                                                                                 | No direct schema mapping.      |
| `public_keys`       | `map[string]string` | Name→key map of SSH keys.                                                                                                               | No direct schema mapping.      |
| `launch_index`      | `int`               | Nova launch_index — position within a multi-instance launch request.                                                                    | No direct schema mapping.      |
| `devices`           | `[]Device`          | Attached block-device list from meta_data.json (type / bus / serial / path / address / tags).                                           | No direct schema mapping.      |
| `provider`          | `string`            | `"openstack"` or `"dreamhost"` (legacy Dreamhost OpenStack images shipped a `dhc-user` account). Matches Ohai's `openstack[:provider]`. | No direct schema mapping.      |

Still skipped from meta_data.json: `network_info` (deployment-specific Neutron
data) and `random_seed` (sensitive — no inventory value).

## Platform Support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (only meaningful on an OpenStack VM) |
| Other    | ✅ (only meaningful on an OpenStack VM) |

## Example Output

```json
{
  "openstack": {
    "instance_id": "i-abc",
    "instance_type": "m1.small",
    "availability_zone": "nova",
    "local_ipv4": "10.0.0.5",
    "uuid": "uuid-xxx",
    "project_id": "proj-1"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("openstack"))
facts, _ := g.Collect(context.Background())

if facts.OpenStack != nil {
    fmt.Println(facts.OpenStack.ProjectID, facts.OpenStack.AvailabilityZone)
}
```

## Enable/Disable

```bash
gohai --collector.openstack      # enable (opt-in)
gohai --no-collector.openstack   # disable (default)
gohai --category=cloud           # pulls this + all cloud collectors
```

## Dependencies

`dmi`. OpenStack writes `"OpenStack Nova"` or similar as `product_name`. Fails
open when dmi wasn't run.

## Data Sources

1. **DMI gate:** `dmi.Product.Name` contains `"OpenStack"`. Mirrors Ohai's
   virtualization-plugin gate (which itself reads DMI).
2. **EC2-mirror walk:** starts at `/latest/meta-data/` and recursively walks the
   directory listing — entries ending in `/` are subdirectories (recurse),
   others are leaves. Matches Ohai's `fetch_metadata` style for the
   EC2-compatible tree. Forward-compatible — fields OpenStack adds later surface
   automatically under `Raw`.
3. **Nova endpoint:** `GET /openstack/latest/meta_data.json` — the richer
   Nova-native document with `uuid`, `project_id`, `meta`, etc.
4. **Provider field:** read `/etc/passwd` for a `dhc-user` entry. Present →
   `"dreamhost"`. Absent → `"openstack"`. Matches Ohai's `openstack_provider`.
5. **User-Agent:** `gohai` (the cloudmetadata default).
6. **Timeout:** 2 seconds (Ohai's default; configurable in Ohai but gohai uses
   the cloudmetadata default).
7. **Failure handling:** if both the EC2-mirror walk AND the Nova doc fail, we
   drop with `(nil, nil)`. Either source alone is enough to surface a populated
   Info.

Mirrors Ohai's `Ohai::Mixin::Ec2Metadata` recursive walk (reused by the
openstack plugin) plus the Nova-specific document and the provider/dreamhost
detection.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
