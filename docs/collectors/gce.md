# GCE

> **Status:** Implemented ✅

## Description

The collector reads the Google Compute Engine instance and project metadata from
the link-local metadata server at
`http://metadata.google.internal./computeMetadata/v1/`. The collector gives the
`facts.Gce != nil` signal. If you can read the data, the host is on GCE. On a
host that is not on GCE, the collector cannot reach the metadata endpoint. Then
the `gce` field is not in `Facts`, and there is no error or log message.

The output has each field that GCE returns for `?recursive=true`. The collector
makes the fields flat, the same as the other gohai collectors, which each have
one top-level struct. The collector changes the resource paths (`machineType`,
`zone`, `image`, `network`) to their short forms. It reports all other values
exactly as GCE gives them.

## Collected fields

### Top-level

| Field                       | Type                 | Description                                                                                             | Schema mapping                        |
| --------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| `id`                        | `int64`              | The numeric GCE instance ID.                                                                            | OTel `cloud.resource_id` / `host.id`  |
| `name`                      | `string`             | The VM instance name.                                                                                   | OTel `host.name`                      |
| `hostname`                  | `string`             | The fully qualified internal DNS name.                                                                  | OCSF `device.hostname`                |
| `cpu_platform`              | `string`             | The CPU platform, for example `Intel Broadwell`.                                                        | OTel `host.cpu.model.name`            |
| `type`                      | `string`             | The short machine type, for example `n1-standard-1`.                                                    | OTel `host.type`                      |
| `image_id`                  | `string`             | The short image name, for example `debian-12`.                                                          | OTel `host.image.name`                |
| `description`               | `string`             | The free-form description of the instance.                                                              | No direct schema mapping.             |
| `tags`                      | `[]string`           | The GCE instance tags. These are firewall-rule targets, not labels.                                     | No direct schema mapping.             |
| `preemptible`               | `bool`               | `true` when the VM is preemptible or spot.                                                              | No direct schema mapping.             |
| `automatic_restart`         | `string`             | `"TRUE"` or `"FALSE"`. The restart policy after a failure.                                              | No direct schema mapping.             |
| `on_host_maintenance`       | `string`             | `MIGRATE` or `TERMINATE`. What the VM does during host maintenance.                                     | No direct schema mapping.             |
| `maintenance_event`         | `string`             | The current maintenance event, for example `NONE` or `MIGRATE_ON_HOST_MAINTENANCE`.                     | No direct schema mapping.             |
| `zone`                      | `string`             | The GCE zone, for example `us-central1-a`.                                                              | OTel `cloud.availability_zone`        |
| `region`                    | `string`             | The GCE region, from the zone, for example `us-central1`.                                               | OTel `cloud.region`                   |
| `project_uid`               | `string`             | The GCP project ID.                                                                                     | OTel `cloud.account.id`               |
| `numeric_project_id`        | `int64`              | The numeric GCP project ID.                                                                             | No direct schema mapping.             |
| `project_attributes`        | `map[string]string`  | The project metadata. It usually contains `ssh-keys`, `enable-oslogin` and similar keys.                | No direct schema mapping.             |
| `licenses`                  | `[]string`           | The GCP license IDs of the VM, for BYOL and compliance records.                                         | No direct schema mapping.             |
| `attributes`                | `map[string]string`  | The instance metadata. It frequently contains `ssh-keys`, `startup-script` and user tags.               | No direct schema mapping.             |
| `network_interfaces`        | `[]NetworkInterface` | The attached VNICs. Refer to the table below.                                                           | OCSF `network_interface` (per entry). |
| `disks`                     | `[]Disk`             | The attached disks. Refer to the table below.                                                           | No direct schema mapping.             |
| `service_accounts`          | `[]ServiceAccount`   | The service accounts of the VM. Refer to the table below.                                               | No direct schema mapping.             |
| `virtual_clock_drift_token` | `string`             | The opaque drift token from `instance.virtualClock`. Some monitoring tools use it to find VM migration. | No direct schema mapping.             |
| `remaining_cpu_time`        | `int64`              | The CPU seconds that remain before Spot VM preemption. Zero on standard instances.                      | No direct schema mapping.             |
| `partner_attributes`        | `map[string]string`  | Opaque key/value pairs that partner images set (marketplace product metadata).                          | No direct schema mapping.             |

### NetworkInterface

| Field                 | Type             | Description                                                  |
| --------------------- | ---------------- | ------------------------------------------------------------ |
| `ip`                  | `string`         | The private IPv4 address.                                    |
| `mac`                 | `string`         | The MAC address.                                             |
| `network`             | `string`         | The short network name, for example `default`.               |
| `subnetmask`          | `string`         | The subnet mask.                                             |
| `gateway`             | `string`         | The default gateway.                                         |
| `dns_servers`         | `[]string`       | The DNS resolvers.                                           |
| `ip_aliases`          | `[]string`       | The alias IP ranges of this interface.                       |
| `forwarded_ips`       | `[]string`       | The static external IPs that GCE forwards to this interface. |
| `target_instance_ips` | `[]string`       | The target-instance IPs.                                     |
| `mtu`                 | `int`            | The interface MTU.                                           |
| `access_configs`      | `[]AccessConfig` | The external access configs (NAT or public IPs).             |

### AccessConfig

| Field         | Type     | Description                                                            |
| ------------- | -------- | ---------------------------------------------------------------------- |
| `external_ip` | `string` | The public IPv4 address.                                               |
| `type`        | `string` | The access-config type, for example `ONE_TO_ONE_NAT` or `DIRECT_IPV6`. |

### Disk

| Field         | Type     | Description                                               |
| ------------- | -------- | --------------------------------------------------------- |
| `device_name` | `string` | The device name that the VM sees, for example `boot`.     |
| `type`        | `string` | `PERSISTENT` or `SCRATCH`.                                |
| `mode`        | `string` | `READ_WRITE` or `READ_ONLY`.                              |
| `index`       | `int`    | The attachment index.                                     |
| `interface`   | `string` | The bus interface, `SCSI` or `NVME`.                      |
| `encrypted`   | `bool`   | `true` when a GCP customer-managed key encrypts the disk. |

### ServiceAccount

| Field     | Type       | Description                                                                         |
| --------- | ---------- | ----------------------------------------------------------------------------------- |
| `key`     | `string`   | The map key from the GCE response. Usually `"default"` or the full email of the SA. |
| `email`   | `string`   | The service-account identity.                                                       |
| `aliases` | `[]string` | The alias names. This field is usually empty.                                       |
| `scopes`  | `[]string` | The OAuth scopes of the SA, for IAM audits.                                         |

## Platform support

| Platform | Supported                    |
| -------- | ---------------------------- |
| Linux    | ✅                           |
| macOS    | ✅ (useful only on a GCE VM) |
| Other    | ✅ (useful only on a GCE VM) |

The collector operates on all operating systems, because it sends an HTTP
request. It does not read a file. It returns data only on hosts where
`metadata.google.internal` resolves and responds. In practice, these are GCE
VMs.

## Example output

```json
{
  "gce": {
    "id": 1234567890123,
    "name": "my-vm",
    "hostname": "my-vm.c.my-project.internal",
    "cpu_platform": "Intel Broadwell",
    "type": "n1-standard-1",
    "image_id": "debian-12",
    "description": "primary app server",
    "tags": ["http-server", "https-server"],
    "preemptible": false,
    "automatic_restart": "TRUE",
    "on_host_maintenance": "MIGRATE",
    "maintenance_event": "NONE",
    "zone": "us-central1-a",
    "region": "us-central1",
    "project_uid": "my-project",
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

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.gce      # enable (opt-in)
gohai collect --no-collector.gce   # disable (default)
gohai collect --category=cloud     # pulls gce + dmi + other cloud collectors as they land
```

The collector is opt-in. On a host that is not on GCE, a request to the metadata
endpoint waits for the `cloudmetadata` default timeout (2 s).

## Dependencies

`dmi`. A DMI product-name check controls the request to the GCE metadata
endpoint. The collector skips the HTTP request when `dmi.Product.Name` does not
contain `"Google Compute Engine"`. This is the same optimization as Ohai's
`has_gce_dmi?`. When you enable `gce`, the dependency resolution of the registry
enables `dmi` automatically. You do not have to enable it. If `dmi` is not in
the prior results (because the user disabled it), the collector fails open and
sends the HTTP request. This is slower on hosts that are not on GCE, but the
result is still correct.

## Data sources

1. **DMI check** (refer to [dmi.md](dmi.md)). The collector reads the prior
   result of the `dmi` collector. If `product.name` does not contain
   `"Google Compute Engine"`, it stops and returns `(nil, nil)`. This prevents
   the 2 s metadata-endpoint timeout on hosts that are not on GCE. If `dmi` did
   not run, the check fails open and the collector sends the HTTP request.
2. **Endpoint.**
   `http://metadata.google.internal./computeMetadata/v1/?recursive=true`. The
   dot at the end of the hostname stops the DNS search path of the host.
3. **Required header.** `Metadata-Flavor: Google`. GCE refuses requests without
   it. This gives protection against lateral SSRF probes that do not know to set
   the header.
4. **User-Agent.** `gohai` (the cloudmetadata default).
5. **Timeout.** 6 seconds. This is the same as Ohai's `read_timeout` in
   `mixin/gce_metadata.rb`. When the GCE link-local endpoint is reachable, it
   answers in milliseconds.
6. **Failures.** If a read fails (transport, a status that is not 2xx, the body
   read), the collector returns `(nil, nil)`. Then the `gce` field is not in
   Facts, and there is no error. Only JSON that is not valid in the response
   gives an error.
7. **Changes to the data.** The collector changes the resource paths
   (`machineType`, `zone`, `image`, `network`) to their last segment. It changes
   `scheduling.preemptible` from `"TRUE"` or `"FALSE"` to a real `bool`. It gets
   `region` from the zone: it removes the `-<letter>` suffix at the end of the
   zone. It moves `instance.virtualClock.driftToken` to the flat
   `virtual_clock_drift_token`. It moves `instance.remainingCpuTime` and
   `instance.partnerAttributes` to `remaining_cpu_time` and
   `partner_attributes`. The collector does not keep the full tree as GCE gives
   it. All fields in the output are typed.

The method is the same as Ohai's `Ohai::Mixin::GCEMetadata`: the same URL, the
same header, the same single recursive request, and the same full tree.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors. It sends GET requests with a short
  timeout, adds the `Metadata-Flavor` header, and gives the `ErrNotAvailable`
  sentinel. Its transport is the `HTTP` of the sources, so a fake host can
  answer the metadata reads.
