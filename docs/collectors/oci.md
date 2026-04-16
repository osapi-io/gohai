# OCI

> **Status:** Implemented ✅

## Description

Collects Oracle Cloud Infrastructure instance metadata from the link-local
server at `http://169.254.169.254/opc/v2/`. Three separate JSON endpoints are
fetched: `/instance`, `/vnics`, and `/allVolumeAttachments`. `facts.OCI != nil`
is the detection signal.

Detection is gated on DMI `chassis.asset_tag` containing `"OracleCloud.com"` —
matches Ohai's `oci_chassis_asset_tag?` regex. Every request carries the literal
`Authorization: Bearer Oracle` header OCI's IMDSv2 requires (not a real JWT —
that's the hardcoded secret).

## Collected Fields

| Field                   | Type                          | Description                                                                                           | Schema mapping                 |
| ----------------------- | ----------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------ |
| `id`                    | `string`                      | Instance OCID.                                                                                        | OTel `cloud.resource_id`       |
| `display_name`          | `string`                      | Instance display name.                                                                                | OTel `host.name`               |
| `hostname`              | `string`                      | Instance hostname.                                                                                    | OCSF `device.hostname`         |
| `shape`                 | `string`                      | Compute shape (e.g. `VM.Standard.E4.Flex`).                                                           | OTel `host.type`               |
| `shape_config`          | `*ShapeConfig`                | Shape resource profile.                                                                               | No direct schema mapping.      |
| `image`                 | `string`                      | Image OCID.                                                                                           | OTel `host.image.id`           |
| `region`                | `string`                      | Region short code (e.g. `phx`).                                                                       | OTel `cloud.region`            |
| `canonical_region_name` | `string`                      | Full region (e.g. `us-phoenix-1`).                                                                    | No direct schema mapping.      |
| `availability_domain`   | `string`                      | OCI availability domain.                                                                              | OTel `cloud.availability_zone` |
| `fault_domain`          | `string`                      | OCI fault domain.                                                                                     | No direct schema mapping.      |
| `compartment_id`        | `string`                      | Compartment OCID.                                                                                     | OTel `cloud.account.id`        |
| `tenant_id`             | `string`                      | Tenancy OCID.                                                                                         | No direct schema mapping.      |
| `state`                 | `string`                      | Lifecycle state (e.g. `RUNNING`).                                                                     | No direct schema mapping.      |
| `time_created`          | `int64`                       | Creation time (Unix ms).                                                                              | No direct schema mapping.      |
| `metadata`              | `map[string]string`           | Instance metadata key/values.                                                                         | No direct schema mapping.      |
| `defined_tags`          | `map[string]any`              | OCI defined tags (namespaced).                                                                        | No direct schema mapping.      |
| `freeform_tags`         | `map[string]string`           | OCI free-form tags.                                                                                   | No direct schema mapping.      |
| `region_info`           | `*RegionInfo`                 | Geographic identification sub-record.                                                                 | No direct schema mapping.      |
| `agent_config`          | `*AgentConfig`                | Oracle Cloud Agent state (management/monitoring/plugin toggles).                                      | No direct schema mapping.      |
| `availability_config`   | `*AvailabilityConfig`         | Maintenance recovery preferences (live migration + recovery action).                                  | No direct schema mapping.      |
| `instance_pool_id`      | `string`                      | Parent instance pool OCID (empty on stand-alone VMs).                                                 | No direct schema mapping.      |
| `dedicated_vm_host_id`  | `string`                      | Dedicated VM host OCID (empty on shared infrastructure).                                              | No direct schema mapping.      |
| `launch_options`        | `*LaunchOptions`              | VM launch-time config (boot volume type, firmware, network type, encryption-in-transit).              | No direct schema mapping.      |
| `source_details`        | `*SourceDetails`              | Image/boot-volume source info (source_type, image_id, boot_volume_size_in_gbs, kms_key_id).           | No direct schema mapping.      |
| `platform_config`       | `map[string]any`              | Shape-family-specific platform config (AMD / Intel / GPU variants have different sub-shapes).         | No direct schema mapping.      |
| `vnics`                 | `[]VNIC`                      | Virtual NICs — see below.                                                                             | OCSF `network_interface`       |
| `volume_attachments`    | `map[string]VolumeAttachment` | Attached volumes keyed by attachment OCID — matches Ohai's `metadata.volumes[<id>]` shape. See below. | No direct schema mapping.      |

### ShapeConfig

| Field                          | Type      | Description                           |
| ------------------------------ | --------- | ------------------------------------- |
| `ocpus`                        | `float64` | Oracle CPU count.                     |
| `memory_in_gbs`                | `float64` | Memory in GB.                         |
| `networking_bandwidth_in_gbps` | `float64` | Networking bandwidth.                 |
| `max_vnic_attachments`         | `int`     | Maximum VNICs allowed for this shape. |
| `gpus`                         | `int`     | GPU count.                            |

### RegionInfo

| Field                    | Type     | Description                        |
| ------------------------ | -------- | ---------------------------------- |
| `realm_key`              | `string` | Realm identifier.                  |
| `realm_domain_component` | `string` | Realm DNS domain component.        |
| `region_key`             | `string` | Region short key (e.g. `PHX`).     |
| `region_identifier`      | `string` | Full region (e.g. `us-phoenix-1`). |

### VNIC

| Field               | Type     | Description                  |
| ------------------- | -------- | ---------------------------- |
| `vnic_id`           | `string` | VNIC OCID.                   |
| `private_ip`        | `string` | Private IPv4.                |
| `vlan_tag`          | `int`    | 802.1Q VLAN tag.             |
| `mac_addr`          | `string` | MAC address.                 |
| `virtual_router_ip` | `string` | Virtual router IP (gateway). |
| `subnet_cidr_block` | `string` | Subnet CIDR.                 |
| `nic_index`         | `int`    | NIC index.                   |

### VolumeAttachment

| Field                   | Type     | Description                    |
| ----------------------- | -------- | ------------------------------ |
| `id`                    | `string` | Attachment OCID.               |
| `attachment_type`       | `string` | `paravirtualized` / `iscsi`.   |
| `display_name`          | `string` | Attachment display name.       |
| `volume_id`             | `string` | Volume OCID.                   |
| `is_read_only`          | `bool`   | Whether the attachment is RO.  |
| `lifecycle_state`       | `string` | Attachment lifecycle state.    |
| `device`                | `string` | Device path on the instance.   |
| `iqn`                   | `string` | iSCSI IQN (iSCSI only).        |
| `ipv4`                  | `string` | iSCSI target IPv4.             |
| `port`                  | `int`    | iSCSI target port.             |
| `encryption_in_transit` | `bool`   | Whether encryption in transit. |

## Platform Support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (only meaningful on an OCI instance) |
| Other    | ✅ (only meaningful on an OCI instance) |

## Example Output

```json
{
  "oci": {
    "id": "ocid1.instance.oc1.phx.xxx",
    "shape": "VM.Standard.E4.Flex",
    "region": "phx",
    "availability_domain": "pPrU:PHX-AD-1",
    "compartment_id": "ocid1.compartment.oc1..zzz",
    "vnics": [{ "private_ip": "10.0.1.5", "mac_addr": "02:00:17:01:02:03" }]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("oci"))
facts, _ := g.Collect(context.Background())

if facts.OCI != nil {
    fmt.Println("running on OCI,", facts.OCI.CanonicalRegionName)
}
```

## Enable/Disable

```bash
gohai --collector.oci      # enable (opt-in)
gohai --no-collector.oci   # disable (default)
gohai --category=cloud     # pulls this + all cloud collectors
```

## Dependencies

`dmi`. OCI writes `"OracleCloud.com"` into chassis asset tag. The collector
gates on a substring match. Fails open when dmi wasn't run.

## Data Sources

1. **DMI gate:** `dmi.Chassis.AssetTag` contains `"OracleCloud.com"`. Matches
   Ohai's `oci_chassis_asset_tag?`.
2. **Endpoints:** `GET /opc/v2/instance`, `GET /opc/v2/vnics`,
   `GET /opc/v2/allVolumeAttachments` — three JSON documents.
3. **Required header:** `Authorization: Bearer Oracle` (literal).
4. **User-Agent:** `gohai` (the cloudmetadata default).
5. **Timeout:** 6 seconds — matches Ohai's `read_timeout` in
   `mixin/oci_metadata.rb`.
6. **Failure handling:** instance-doc failure returns `(nil, nil)`. Per-section
   failures on `/vnics` or `/allVolumeAttachments` are tolerated (lightweight
   shapes legitimately 404 those paths) — the sections are left empty in the
   Info struct.
7. **Transformation:** raw camelCase fields are unmarshalled and mapped to
   snake_case in the typed Info. Optional sub-objects from the instance document
   — `agent_config`, `availability_config`, `launch_options`, `source_details` —
   become typed nested records when present. `platform_config` is kept as
   `map[string]any` because its shape varies by shape family (AMD / Intel / GPU
   each expose different sub-fields). `instance_pool_id` and
   `dedicated_vm_host_id` are lifted straight from the instance document (empty
   on stand-alone / shared-infrastructure VMs). Volume attachments are keyed by
   their OCID to match Ohai's `metadata.volumes[<id>]` output shape.

Mirrors Ohai's `Ohai::Mixin::OciMetadata` methodology — same endpoint, same auth
header, same three sub-fetches, same volume keying, same 6s timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
