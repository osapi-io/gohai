# GCE

> **Status:** Implemented ✅

## Description

Collects Google Compute Engine instance and project metadata by hitting the
link-local metadata server at
`http://metadata.google.internal./computeMetadata/v1/`. The collector is the
`facts.Gce != nil` signal — if you can read it, you're running on GCE. On
non-GCE hosts the metadata endpoint isn't reachable and the `gce` field drops
from `Facts` silently (no error, no noise).

The shape mirrors every field GCE returns via `?recursive=true`, flattened to
match the rest of the gohai codebase's single-top-level-struct convention.
Resource paths (`machineType`, `zone`, `image`, `network`) are normalized to
their short forms; everything else is surfaced verbatim.

## Collected Fields

### Top-level

| Field                       | Type                 | Description                                                                                             | Schema mapping                        |
| --------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| `instance_id`               | `int64`              | GCE numeric instance ID.                                                                                | OTel `cloud.resource_id` / `host.id`  |
| `name`                      | `string`             | VM instance name.                                                                                       | OTel `host.name`                      |
| `hostname`                  | `string`             | Fully-qualified internal DNS name.                                                                      | OCSF `device.hostname`                |
| `cpu_platform`              | `string`             | Underlying CPU platform (e.g. `Intel Broadwell`).                                                       | OTel `host.cpu.model.name`            |
| `machine_type`              | `string`             | Short machine type (e.g. `n1-standard-1`).                                                              | OTel `host.type`                      |
| `image`                     | `string`             | Short image name (e.g. `debian-12`).                                                                    | OTel `host.image.name`                |
| `description`               | `string`             | Free-form description set on the instance.                                                              | No direct schema mapping.             |
| `tags`                      | `[]string`           | GCE instance tags (firewall-rule targets, not labels).                                                  | No direct schema mapping.             |
| `preemptible`               | `bool`               | `true` when the VM is preemptible/spot.                                                                 | No direct schema mapping.             |
| `automatic_restart`         | `string`             | `"TRUE"` / `"FALSE"` — restart policy on failure.                                                       | No direct schema mapping.             |
| `on_host_maintenance`       | `string`             | `MIGRATE` / `TERMINATE` — host-maintenance behavior.                                                    | No direct schema mapping.             |
| `maintenance_event`         | `string`             | Current maintenance event (`NONE`, `MIGRATE_ON_HOST_MAINTENANCE`, etc.).                                | No direct schema mapping.             |
| `zone`                      | `string`             | GCE zone (e.g. `us-central1-a`).                                                                        | OTel `cloud.availability_zone`        |
| `region`                    | `string`             | GCE region, derived from zone (e.g. `us-central1`).                                                     | OTel `cloud.region`                   |
| `project_id`                | `string`             | GCP project ID.                                                                                         | OTel `cloud.account.id`               |
| `numeric_project_id`        | `int64`              | GCP project numeric ID.                                                                                 | No direct schema mapping.             |
| `project_attributes`        | `map[string]string`  | Project-level metadata. Commonly contains `ssh-keys`, `enable-oslogin`, etc.                            | No direct schema mapping.             |
| `licenses`                  | `[]string`           | GCP license IDs attached to the VM (for BYOL / compliance tracking).                                    | No direct schema mapping.             |
| `attributes`                | `map[string]string`  | Instance-level metadata. Often contains `ssh-keys`, `startup-script`, user tags.                        | No direct schema mapping.             |
| `network_interfaces`        | `[]NetworkInterface` | Attached VNICs — see below.                                                                             | OCSF `network_interface` (per entry). |
| `disks`                     | `[]Disk`             | Attached disks — see below.                                                                             | No direct schema mapping.             |
| `service_accounts`          | `[]ServiceAccount`   | Service accounts attached to the VM — see below.                                                        | No direct schema mapping.             |
| `virtual_clock_drift_token` | `string`             | Opaque drift token from `instance.virtualClock` — used by some monitoring tools to detect VM migration. | No direct schema mapping.             |
| `remaining_cpu_time`        | `int64`              | Remaining CPU seconds before Spot VM preemption. Zero on standard instances.                            | No direct schema mapping.             |
| `partner_attributes`        | `map[string]string`  | Opaque key/value pairs set by partner images (marketplace product metadata).                            | No direct schema mapping.             |

### NetworkInterface

| Field                 | Type             | Description                                      |
| --------------------- | ---------------- | ------------------------------------------------ |
| `ip`                  | `string`         | Private IPv4 address.                            |
| `mac`                 | `string`         | MAC address.                                     |
| `network`             | `string`         | Short network name (e.g. `default`).             |
| `subnetmask`          | `string`         | Subnet mask.                                     |
| `gateway`             | `string`         | Default gateway.                                 |
| `dns_servers`         | `[]string`       | DNS resolvers.                                   |
| `ip_aliases`          | `[]string`       | Alias IP ranges attached to this interface.      |
| `forwarded_ips`       | `[]string`       | Static external IPs forwarded to this interface. |
| `target_instance_ips` | `[]string`       | Target-instance IPs.                             |
| `mtu`                 | `int`            | Interface MTU.                                   |
| `access_configs`      | `[]AccessConfig` | External access configs (NAT / public IPs).      |

### AccessConfig

| Field         | Type     | Description                                                |
| ------------- | -------- | ---------------------------------------------------------- |
| `external_ip` | `string` | Public IPv4 address.                                       |
| `type`        | `string` | Access-config type (`ONE_TO_ONE_NAT`, `DIRECT_IPV6`, ...). |

### Disk

| Field         | Type     | Description                                     |
| ------------- | -------- | ----------------------------------------------- |
| `device_name` | `string` | Device name as seen by the VM (e.g. `boot`).    |
| `type`        | `string` | `PERSISTENT` or `SCRATCH`.                      |
| `mode`        | `string` | `READ_WRITE` or `READ_ONLY`.                    |
| `index`       | `int`    | Attachment index.                               |
| `interface`   | `string` | Bus interface — `SCSI` or `NVME`.               |
| `encrypted`   | `bool`   | `true` when GCP customer-managed-key encrypted. |

### ServiceAccount

| Field     | Type       | Description                                                             |
| --------- | ---------- | ----------------------------------------------------------------------- |
| `key`     | `string`   | Map key from GCE's response (usually `"default"` or the SA full email). |
| `email`   | `string`   | Service-account identity.                                               |
| `aliases` | `[]string` | Alias names (rarely populated).                                         |
| `scopes`  | `[]string` | OAuth scopes granted to the SA (for IAM auditing).                      |

## Platform Support

| Platform | Supported                        |
| -------- | -------------------------------- |
| Linux    | ✅                               |
| macOS    | ✅ (only meaningful on a GCE VM) |
| Other    | ✅ (only meaningful on a GCE VM) |

The collector works on any OS — it's an HTTP call, not a file read. It only
returns data on hosts where `metadata.google.internal` resolves and responds,
which in practice is GCE VMs.

## Example Output

```json
{
  "gce": {
    "instance_id": 1234567890123,
    "name": "my-vm",
    "hostname": "my-vm.c.my-project.internal",
    "cpu_platform": "Intel Broadwell",
    "machine_type": "n1-standard-1",
    "image": "debian-12",
    "description": "primary app server",
    "tags": ["http-server", "https-server"],
    "preemptible": false,
    "automatic_restart": "TRUE",
    "on_host_maintenance": "MIGRATE",
    "maintenance_event": "NONE",
    "zone": "us-central1-a",
    "region": "us-central1",
    "project_id": "my-project",
    "numeric_project_id": 987654321,
    "project_attributes": { "ssh-keys": "admin:ssh-rsa BBBB..." },
    "licenses": ["8045211539491955793"],
    "attributes": {
      "ssh-keys": "user:ssh-rsa AAAA...",
      "startup-script": "#!/bin/bash\necho hi"
    },
    "network_interfaces": [
      {
        "ip": "10.128.0.5",
        "mac": "42:01:0a:80:00:05",
        "network": "default",
        "subnetmask": "255.255.240.0",
        "gateway": "10.128.0.1",
        "dns_servers": ["169.254.169.254"],
        "ip_aliases": ["10.132.0.0/20"],
        "forwarded_ips": ["34.102.0.1"],
        "mtu": 1460,
        "access_configs": [
          { "external_ip": "34.123.45.67", "type": "ONE_TO_ONE_NAT" }
        ]
      }
    ],
    "disks": [
      {
        "device_name": "boot",
        "type": "PERSISTENT",
        "mode": "READ_WRITE",
        "index": 0,
        "interface": "SCSI",
        "encrypted": true
      }
    ],
    "service_accounts": [
      {
        "key": "default",
        "email": "default@my-project.iam.gserviceaccount.com",
        "scopes": [
          "https://www.googleapis.com/auth/cloud-platform",
          "https://www.googleapis.com/auth/logging.write"
        ]
      }
    ]
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("gce"))
facts, _ := g.Collect(context.Background())

if facts.Gce != nil {
    fmt.Println("running on GCE in", facts.Gce.Region)
    for _, sa := range facts.Gce.ServiceAccounts {
        fmt.Println("SA:", sa.Email, sa.Scopes)
    }
    if keys, ok := facts.Gce.Attributes["ssh-keys"]; ok {
        fmt.Println("SSH keys from metadata:", keys)
    }
}
```

## Enable/Disable

```bash
gohai --collector.gce      # enable (opt-in)
gohai --no-collector.gce   # disable (default)
gohai --category=cloud     # pulls gce + dmi + other cloud collectors as they land
```

Opt-in because hitting the metadata endpoint on non-GCE hosts eats the
`cloudmetadata` default timeout (2s).

## Dependencies

`dmi`. The GCE metadata endpoint is gated by a DMI product-name check — the
collector skips the HTTP call when `dmi.Product.Name` doesn't contain
`"Google Compute Engine"`. Mirrors Ohai's `has_gce_dmi?` optimization. Enabling
`gce` automatically pulls `dmi` via the registry's dependency resolution, so you
don't have to enable it yourself. When `dmi` is absent from the prior results
(explicitly disabled by the user) the collector fails open and attempts the HTTP
probe anyway — slower on non-GCE hosts but still correct.

## Data Sources

1. **DMI gate** (see [dmi.md](dmi.md)): reads the prior `dmi` collector result
   and short-circuits with `(nil, nil)` when `product.name` doesn't contain
   `"Google Compute Engine"`. Saves the 2s metadata-endpoint timeout on non-GCE
   hosts. Fails open (tries the HTTP call) when `dmi` was not run.
2. **Endpoint:**
   `http://metadata.google.internal./computeMetadata/v1/?recursive=true`
   (trailing dot on the hostname defeats the host's DNS search path).
3. **Required header:** `Metadata-Flavor: Google` — GCE rejects requests without
   it, which protects against lateral SSRF-style probes that wouldn't know to
   set it.
4. **User-Agent:** `gohai` (the cloudmetadata default).
5. **Timeout:** 6 seconds — matches Ohai's `read_timeout` in
   `mixin/gce_metadata.rb`. GCE's link-local endpoint answers in milliseconds
   when reachable.
6. **Failure handling:** any fetch failure (transport, non-2xx, body read)
   returns `(nil, nil)` so the `gce` field drops from Facts silently. Only
   malformed JSON in the response surfaces as an error.
7. **Transformation:** resource paths (`machineType`, `zone`, `image`,
   `network`) are normalized to their short last-segment forms;
   `scheduling.preemptible` is converted from `"TRUE"`/`"FALSE"` to a real
   `bool`; `region` is derived by stripping the zone's trailing `-<letter>`
   suffix. `instance.virtualClock.driftToken` is lifted to the flat
   `virtual_clock_drift_token`; `instance.remainingCpuTime` and
   `instance.partnerAttributes` are lifted to `remaining_cpu_time` and
   `partner_attributes`. The full untransformed tree is not retained — all
   surfaced fields are typed.

Mirrors Ohai's `Ohai::Mixin::GCEMetadata` collection approach: same URL, same
header, same single recursive call, same full-tree coverage.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector. Handles short-timeout GETs,
  `Metadata-Flavor` header injection, and the `ErrNotAvailable` sentinel.
