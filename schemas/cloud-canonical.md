# Cloud Canonical Overlay

Cross-provider field standardization for gohai cloud collectors. This document
maps 10 canonical cloud-instance fields to each provider's raw metadata source,
showing exactly which gohai `Info` struct field carries the data and where it
originates in the provider's IMDS response.

## Canonical Fields

| Canonical       | Type              | Description                                |
| --------------- | ----------------- | ------------------------------------------ |
| `instance_id`   | string            | Provider-assigned unique instance identifier |
| `region`        | string            | Cloud region                               |
| `zone`          | string            | Availability zone within region            |
| `instance_type` | string            | Instance size/shape                        |
| `hostname`      | string            | Instance hostname from provider metadata   |
| `public_ips`    | []string          | Public IP addresses                        |
| `private_ips`   | []string          | Private IP addresses                       |
| `account_id`    | string            | Cloud account/project identifier           |
| `image_id`      | string            | Machine image identifier                   |
| `tags`          | map[string]string | Provider tags/labels                       |

## Per-Provider Mapping

### `instance_id`

| Provider   | gohai field       | IMDS source                                            |
| ---------- | ----------------- | ------------------------------------------------------ |
| EC2        | `instance_id`     | `/meta-data/instance-id` (fallback: identity document `instanceId`) |
| GCE        | `instance_id`     | `instance.id` (int64 from recursive JSON)              |
| Azure      | `vm_id`           | `compute.vmId`                                         |
| DO         | `droplet_id`      | `droplet_id` (int64 from `/metadata/v1.json`)          |
| OpenStack  | `instance_id`     | `/latest/meta-data/instance-id` (EC2-compatible tree)  |
| Alibaba    | `instance_id`     | `/meta-data/instance-id`                               |
| Linode     | N/A               | No metadata service; detection only (apt sources / domain heuristic) |
| OCI        | `id`              | `/opc/v2/instance` field `id` (OCID)                   |
| Scaleway   | `id`              | `/conf?format=json` field `id`                         |

### `region`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `region`                 | `/meta-data/placement/region` (fallback: identity document `region`) |
| GCE        | `region`                 | Derived: strip trailing `-<letter>` from `zone` (e.g. `us-central1-a` -> `us-central1`) |
| Azure      | `location`               | `compute.location` (Azure calls regions "locations")   |
| DO         | `region`                 | `region` (slug like `nyc1`, `sfo3`)                    |
| OpenStack  | N/A                      | Not exposed via EC2-compatible tree or Nova meta_data.json |
| Alibaba    | `region`                 | `/meta-data/region-id`                                 |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `region`                 | `/opc/v2/instance` field `region`                      |
| Scaleway   | N/A                      | Not exposed directly; only `zone` is available         |

### `zone`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `availability_zone`      | `/meta-data/placement/availability-zone`               |
| GCE        | `zone`                   | `instance.zone` (last segment of resource path)        |
| Azure      | `zone`                   | `compute.zone`                                         |
| DO         | N/A                      | DO regions are single-zone; no separate AZ concept     |
| OpenStack  | `availability_zone`      | `/latest/meta-data/placement/availability-zone` or Nova `availability_zone` |
| Alibaba    | `zone`                   | `/meta-data/zone-id`                                   |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `availability_domain`    | `/opc/v2/instance` field `availabilityDomain`          |
| Scaleway   | `zone`                   | `location.zone_id`                                     |

### `instance_type`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `instance_type`          | `/meta-data/instance-type`                             |
| GCE        | `machine_type`           | `instance.machineType` (last segment of resource path) |
| Azure      | `vm_size`                | `compute.vmSize`                                       |
| DO         | N/A                      | DO metadata does not expose droplet size/type           |
| OpenStack  | `instance_type`          | `/latest/meta-data/instance-type`                      |
| Alibaba    | `instance_type`          | `/meta-data/instance/instance-type`                    |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `shape`                  | `/opc/v2/instance` field `shape`                       |
| Scaleway   | `commercial_type`        | `commercial_type`                                      |

### `hostname`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `hostname`               | `/meta-data/hostname`                                  |
| GCE        | `hostname`               | `instance.hostname`                                    |
| Azure      | `name`                   | `compute.name` (VM name; no dedicated hostname field)  |
| DO         | `hostname`               | `hostname`                                             |
| OpenStack  | `hostname`               | `/latest/meta-data/hostname` (fallback: Nova `hostname`) |
| Alibaba    | `hostname`               | `/meta-data/hostname`                                  |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `hostname`               | `/opc/v2/instance` field `hostname`                    |
| Scaleway   | `hostname`               | `hostname`                                             |

### `public_ips`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `public_ipv4`            | `/meta-data/public-ipv4` (single string)               |
| GCE        | ---                      | Assembled from `network_interfaces[*].access_configs[*].external_ip` |
| Azure      | `public_ipv4`            | Aggregated from `network.interface[*].ipv4.ipAddress[*].publicIpAddress` |
| DO         | ---                      | `interfaces` where `scope == "public"`: `ipv4` field   |
| OpenStack  | `public_ipv4`            | `/latest/meta-data/public-ipv4` (single string)        |
| Alibaba    | `public_ipv4`            | `/meta-data/eipv4` (single string)                     |
| Linode     | `public_ip`              | First non-link-local IPv4 on `eth0` (host NIC, not IMDS) |
| OCI        | N/A                      | VNICs carry `private_ip` only; public IP not in IMDS   |
| Scaleway   | `public_ip`              | `public_ip.address` (single string)                    |

### `private_ips`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `local_ipv4` / `local_ipv4s` | `/meta-data/local-ipv4` (single) and `/meta-data/local-ipv4s` (array) |
| GCE        | ---                      | `network_interfaces[*].ip`                             |
| Azure      | `local_ipv4`             | Aggregated from `network.interface[*].ipv4.ipAddress[*].privateIpAddress` |
| DO         | ---                      | `interfaces` where `scope == "private"`: `ipv4` field  |
| OpenStack  | `local_ipv4`             | `/latest/meta-data/local-ipv4` (single string)         |
| Alibaba    | `private_ipv4`           | `/meta-data/private-ipv4` (single string)              |
| Linode     | `private_ip`             | First non-link-local IPv4 on `eth0:1` (host NIC, not IMDS) |
| OCI        | ---                      | `vnics[*].private_ip`                                  |
| Scaleway   | `private_ip`             | `private_ip` (single string)                           |

### `account_id`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `account_id`             | Identity document `accountId`                          |
| GCE        | `project_id`             | `project.projectId` (string) and `numeric_project_id` (int64) |
| Azure      | `subscription_id`        | `compute.subscriptionId`                               |
| DO         | N/A                      | DO metadata does not expose team/account ID             |
| OpenStack  | `project_id`             | Nova `/openstack/latest/meta_data.json` field `project_id` |
| Alibaba    | `owner_account_id`       | `/meta-data/owner-account-id`                          |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `tenant_id`              | `/opc/v2/instance` field `tenantId` (also `compartment_id` for finer scoping) |
| Scaleway   | `organization`           | `organization` (also `project` for project-level scoping) |

### `image_id`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | `ami_id`                 | `/meta-data/ami-id`                                    |
| GCE        | `image`                  | `instance.image` (last segment of resource path)       |
| Azure      | ---                      | Composite: `compute.publisher` / `compute.offer` / `compute.sku` / `compute.version` |
| DO         | N/A                      | DO metadata does not expose image ID                    |
| OpenStack  | `ami_id`                 | `/latest/meta-data/ami-id` (EC2-compatible)            |
| Alibaba    | `image_id`               | `/meta-data/image-id`                                  |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `image`                  | `/opc/v2/instance` field `image` (also `source_details.image_id` from `sourceDetails`) |
| Scaleway   | N/A                      | Scaleway metadata does not expose image ID              |

### `tags`

| Provider   | gohai field              | IMDS source                                            |
| ---------- | ------------------------ | ------------------------------------------------------ |
| EC2        | N/A                      | EC2 IMDS does not expose instance tags (requires API call with `ec2:DescribeTags`) |
| GCE        | `tags`                   | `instance.tags` ([]string network tags, not key-value labels) |
| Azure      | `tags` / `tags_list`     | `compute.tags` (semicolon-separated string) and `compute.tagsList` ([]Tag with name/value) |
| DO         | `tags`                   | `tags` ([]string, not key-value pairs)                 |
| OpenStack  | N/A                      | Not exposed via EC2-compatible tree; Nova `meta` field serves as user-defined metadata |
| Alibaba    | `tags`                   | `/meta-data/tags/instance` (map[string]string)         |
| Linode     | N/A                      | No metadata service                                    |
| OCI        | `freeform_tags`          | `/opc/v2/instance` field `freeformTags` (map[string]string); also `defined_tags` (namespaced map[string]any) |
| Scaleway   | `tags`                   | `tags` ([]string, not key-value pairs)                 |

## Notes

### GCE region is derived, not native

GCE metadata does not expose a standalone `region` field. The collector derives
it by stripping the trailing availability-zone letter from `zone`:
`us-central1-a` becomes `us-central1`. This is implemented in `zoneToRegion()`
which finds the last `-` and truncates. The same approach Ohai uses.

### GCE resource paths are normalized

GCE returns `machineType`, `zone`, `image`, and `network` as full resource
paths (`projects/my-project/zones/us-central1-a`). The collector's
`lastSegment()` helper extracts the trailing name, so consumers see `n2-standard-4`
rather than `projects/12345/machineTypes/n2-standard-4`.

### Azure uses "location" instead of "region"

Azure's IMDS names it `compute.location` rather than `region`. The gohai field
is `location` to match the raw source. A canonical overlay would map this to
`region`.

### Azure image identity is composite

Azure has no single `image_id` field. A marketplace image is identified by four
coordinates: `publisher`, `offer`, `sku`, `version`. Custom images use
`storageProfile.osDisk.managedDisk.id`. A canonical `image_id` would need to be
assembled from these components.

### Azure tags have two representations

Azure surfaces tags both as a semicolon-separated flat string (`compute.tags`)
and as a structured array of `{name, value}` objects (`compute.tagsList`). The
gohai collector exposes both, but the `tags_list` field is the one with proper
key-value structure.

### DigitalOcean and Scaleway tags are string arrays

DO and Scaleway `tags` are flat string arrays (`[]string`), not key-value maps.
They function as labels rather than the key-value tagging model used by AWS,
Azure, Alibaba, and OCI. GCE's `tags` are similarly flat strings (network tags),
though GCE also has key-value `labels` which are not exposed via the metadata
service.

### Linode has no metadata service

Linode detection is purely heuristic (apt sources / hostname domain). The
collector reads IP addresses from host network interfaces (`eth0`, `eth0:1`),
not from a cloud metadata endpoint. It exposes only `public_ip` and `private_ip`
-- no instance ID, region, type, account, image, or tags.

### OCI uses "shape" for instance type

OCI calls its VM size a `shape` (e.g. `VM.Standard.E4.Flex`). The associated
`shape_config` sub-object carries OCPU count, memory, and networking bandwidth.

### OCI availability zones are called "availability domains"

OCI's equivalent of an availability zone is `availabilityDomain`. The gohai
field is `availability_domain`. Additionally, OCI has `fault_domain` for
finer-grained placement within an AD.

### OCI public IPs are not in instance IMDS

OCI's IMDS VNIC endpoint (`/opc/v2/vnics`) only carries `private_ip`. Public
IPs require the OCI API (`GetVnic` call), not the metadata service.

### OpenStack region is absent

OpenStack's EC2-compatible metadata tree does not expose a region field. The
Nova-specific `meta_data.json` similarly omits region. Discovering the region
requires either the Keystone catalog or operator-specific configuration.

### Alibaba uses a non-standard metadata address

Alibaba's IMDS lives at `http://100.100.100.200` rather than the
`169.254.169.254` link-local address used by AWS, Azure, GCE, DO, OpenStack,
OCI, and Scaleway (which uses `169.254.42.42`).

### EC2 tags require API access

EC2's IMDS does not expose instance tags. The `InstanceMetadataTags` feature
(added in 2022) requires explicit opt-in at the instance level and uses a
separate IMDS path (`/latest/meta-data/tags/instance/`). gohai does not
currently fetch this path.

## Coverage Summary

| Canonical       | Full | Partial | N/A   |
| --------------- | ---- | ------- | ----- |
| `instance_id`   | EC2, GCE, Azure, DO, OpenStack, Alibaba, OCI, Scaleway | -- | Linode |
| `region`        | EC2, Alibaba, OCI | GCE (derived) | Azure (as `location`), DO (slug), OpenStack, Linode, Scaleway |
| `zone`          | EC2, GCE, Azure, OpenStack, Alibaba, OCI, Scaleway | -- | DO, Linode |
| `instance_type` | EC2, GCE, Azure, OpenStack, Alibaba, OCI, Scaleway | -- | DO, Linode |
| `hostname`      | EC2, GCE, DO, OpenStack, Alibaba, OCI, Scaleway | Azure (as `name`) | Linode |
| `public_ips`    | EC2, Azure, OpenStack, Alibaba, Scaleway | GCE (per-NIC), DO (per-interface), Linode (host NIC) | OCI |
| `private_ips`   | EC2, Azure, OpenStack, Alibaba, Scaleway | GCE (per-NIC), DO (per-interface), OCI (per-VNIC), Linode (host NIC) | -- |
| `account_id`    | EC2, GCE, Azure, OpenStack, Alibaba, OCI, Scaleway | -- | DO, Linode |
| `image_id`      | EC2, Alibaba, OCI | Azure (composite), GCE (resource path), OpenStack (EC2 mirror) | DO, Linode, Scaleway |
| `tags`          | Alibaba, OCI, Azure | DO ([]string), GCE ([]string), Scaleway ([]string) | EC2 (API only), OpenStack, Linode |
