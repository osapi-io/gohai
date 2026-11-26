# OCI

> **Status:** Implemented ✅

## Description

The collector reads the Oracle Cloud Infrastructure instance metadata from the
link-local server at `http://169.254.169.254/opc/v2/`. It reads three different
JSON endpoints: `/instance`, `/vnics` and `/allVolumeAttachments`. If
`facts.OCI != nil`, the host is an OCI instance.

The collector does the detection only if the DMI `chassis.asset_tag` contains
`"OracleCloud.com"`. This is the same as Ohai's `oci_chassis_asset_tag?` regex.
Each request has the literal `Authorization: Bearer Oracle` header that OCI's
IMDSv2 requires. This value is not a real JWT. It is a fixed secret that OCI
sets.

## Collected fields

| Field                   | Type                          | Description                                                                                                                                      | Schema mapping                 |
| ----------------------- | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------ |
| `id`                    | `string`                      | The instance OCID.                                                                                                                               | OTel `cloud.resource_id`       |
| `name`                  | `string`                      | The instance display name.                                                                                                                       | OTel `host.name`               |
| `hostname`              | `string`                      | The instance hostname.                                                                                                                           | OCSF `device.hostname`         |
| `type`                  | `string`                      | The compute shape, for example `VM.Standard.E4.Flex`.                                                                                            | OTel `host.type`               |
| `shape_config`          | `*ShapeConfig`                | The resource profile of the shape.                                                                                                               | No direct schema mapping.      |
| `image_id`              | `string`                      | The image OCID.                                                                                                                                  | OTel `host.image.id`           |
| `region`                | `string`                      | The short code of the region, for example `phx`.                                                                                                 | OTel `cloud.region`            |
| `canonical_region_name` | `string`                      | The full region name, for example `us-phoenix-1`.                                                                                                | No direct schema mapping.      |
| `zone`                  | `string`                      | The OCI availability domain.                                                                                                                     | OTel `cloud.availability_zone` |
| `fault_domain`          | `string`                      | The OCI fault domain.                                                                                                                            | No direct schema mapping.      |
| `compartment_id`        | `string`                      | The compartment OCID.                                                                                                                            | OTel `cloud.account.id`        |
| `account_uid`           | `string`                      | The tenancy OCID.                                                                                                                                | No direct schema mapping.      |
| `state`                 | `string`                      | The lifecycle state, for example `RUNNING`.                                                                                                      | No direct schema mapping.      |
| `time_created`          | `int64`                       | The creation time, in Unix milliseconds.                                                                                                         | No direct schema mapping.      |
| `metadata`              | `map[string]string`           | The key-value pairs of the instance metadata.                                                                                                    | No direct schema mapping.      |
| `defined_tags`          | `map[string]any`              | The OCI defined tags, in namespaces.                                                                                                             | No direct schema mapping.      |
| `freeform_tags`         | `map[string]string`           | The OCI free-form tags.                                                                                                                          | No direct schema mapping.      |
| `region_info`           | `*RegionInfo`                 | A subrecord that identifies the geographic location.                                                                                             | No direct schema mapping.      |
| `agent_config`          | `*AgentConfig`                | The Oracle Cloud Agent state: the management, monitoring and plugin switches.                                                                    | No direct schema mapping.      |
| `availability_config`   | `*AvailabilityConfig`         | The recovery preferences for maintenance: live migration and the recovery action.                                                                | No direct schema mapping.      |
| `instance_pool_id`      | `string`                      | The OCID of the parent instance pool. It is empty on stand-alone VMs.                                                                            | No direct schema mapping.      |
| `dedicated_vm_host_id`  | `string`                      | The OCID of the dedicated VM host. It is empty on shared infrastructure.                                                                         | No direct schema mapping.      |
| `launch_options`        | `*LaunchOptions`              | The VM configuration at launch: boot volume type, firmware, network type and encryption in transit.                                              | No direct schema mapping.      |
| `source_details`        | `*SourceDetails`              | The image or boot volume source data: source_type, image_id, boot_volume_size_in_gbs, kms_key_id.                                                | No direct schema mapping.      |
| `platform_config`       | `map[string]any`              | The platform configuration of the shape family. The AMD, Intel and GPU variants have different subshapes.                                        | No direct schema mapping.      |
| `vnics`                 | `[]VNIC`                      | The virtual NICs. Refer to the section below.                                                                                                    | OCSF `network_interface`       |
| `volume_attachments`    | `map[string]VolumeAttachment` | The attached volumes, with the attachment OCID as the key. The shape is the same as Ohai's `metadata.volumes[<id>]`. Refer to the section below. | No direct schema mapping.      |

### ShapeConfig

| Field                          | Type      | Description                                 |
| ------------------------------ | --------- | ------------------------------------------- |
| `ocpus`                        | `float64` | The number of Oracle CPUs.                  |
| `memory_in_gbs`                | `float64` | The memory in GB.                           |
| `networking_bandwidth_in_gbps` | `float64` | The network bandwidth.                      |
| `max_vnic_attachments`         | `int`     | The maximum number of VNICs for this shape. |
| `gpus`                         | `int`     | The number of GPUs.                         |

### RegionInfo

| Field                    | Type     | Description                                       |
| ------------------------ | -------- | ------------------------------------------------- |
| `realm_key`              | `string` | The realm identifier.                             |
| `realm_domain_component` | `string` | The DNS domain component of the realm.            |
| `region_key`             | `string` | The short key of the region, for example `PHX`.   |
| `region_identifier`      | `string` | The full region name, for example `us-phoenix-1`. |

### VNIC

| Field               | Type     | Description                                         |
| ------------------- | -------- | --------------------------------------------------- |
| `vnic_id`           | `string` | The VNIC OCID.                                      |
| `private_ip`        | `string` | The private IPv4 address.                           |
| `vlan_tag`          | `int`    | The 802.1Q VLAN tag.                                |
| `mac_addr`          | `string` | The MAC address.                                    |
| `virtual_router_ip` | `string` | The IP address of the virtual router (the gateway). |
| `subnet_cidr_block` | `string` | The subnet CIDR.                                    |
| `nic_index`         | `int`    | The NIC index.                                      |

### VolumeAttachment

| Field                   | Type     | Description                            |
| ----------------------- | -------- | -------------------------------------- |
| `id`                    | `string` | The attachment OCID.                   |
| `attachment_type`       | `string` | `paravirtualized` or `iscsi`.          |
| `display_name`          | `string` | The display name of the attachment.    |
| `volume_id`             | `string` | The volume OCID.                       |
| `is_read_only`          | `bool`   | True if the attachment is read-only.   |
| `lifecycle_state`       | `string` | The lifecycle state of the attachment. |
| `device`                | `string` | The device path on the instance.       |
| `iqn`                   | `string` | The iSCSI IQN. Only for iSCSI.         |
| `ipv4`                  | `string` | The IPv4 address of the iSCSI target.  |
| `port`                  | `int`    | The port of the iSCSI target.          |
| `encryption_in_transit` | `bool`   | True if encryption in transit is on.   |

## Platform support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (data only on an OCI instance)       |
| Other    | ✅ (only meaningful on an OCI instance) |

## Example output

```json
{
  "oci": {
    "id": "ocid1.instance.oc1.phx.xxx",
    "type": "VM.Standard.E4.Flex",
    "region": "phx",
    "zone": "pPrU:PHX-AD-1",
    "compartment_id": "ocid1.compartment.oc1..zzz",
    "vnics": [{ "private_ip": "10.0.1.5", "mac_addr": "02:00:17:01:02:03" }]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("oci"))
facts, _ := g.Collect(context.Background())

if facts.OCI != nil {
    fmt.Println("running on OCI,", facts.OCI.CanonicalRegionName)
}
```

## Enable/disable

```bash
gohai collect --collector.oci      # enable (opt-in)
gohai collect --no-collector.oci   # disable (default)
gohai collect --category=cloud     # pulls this + all cloud collectors
```

## Dependencies

`dmi`. OCI writes `"OracleCloud.com"` into the chassis asset tag. The collector
continues only if the asset tag contains this substring. If `dmi` did not run,
the collector fails open and continues.

## Data sources

1. **DMI gate.** The collector continues only if `dmi.Chassis.AssetTag` contains
   `"OracleCloud.com"`. This is the same as Ohai's `oci_chassis_asset_tag?`.
2. **Endpoints.** The collector reads three JSON documents:
   `GET /opc/v2/instance`, `GET /opc/v2/vnics` and
   `GET /opc/v2/allVolumeAttachments`.
3. **Required header.** Each request has the literal header
   `Authorization: Bearer Oracle`.
4. **User-Agent.** The User-Agent is `gohai`, the cloudmetadata default.
5. **Timeout.** The timeout is 6 seconds. This is the same as Ohai's
   `read_timeout` in `mixin/oci_metadata.rb`.
6. **Failures.** If the instance document fails, the collector returns
   `(nil, nil)`. If `/vnics` or `/allVolumeAttachments` fails, the collector
   continues, because small shapes correctly return 404 for those paths. Those
   sections stay empty in the Info struct.
7. **Transformation.** The collector unmarshals the raw camelCase fields and
   maps them to snake_case in the typed Info. The instance document can have
   optional subobjects: `agent_config`, `availability_config`, `launch_options`
   and `source_details`. If they are there, they become typed nested records.
   `platform_config` stays a `map[string]any`, because its shape changes with
   the shape family. AMD, Intel and GPU shapes each have different subfields.
   The collector copies `instance_pool_id` and `dedicated_vm_host_id` directly
   from the instance document. They are empty on stand-alone VMs and on shared
   infrastructure. The OCID of each volume attachment is its key. This is the
   same as Ohai's `metadata.volumes[<id>]` output shape.

The method is the same as Ohai's `Ohai::Mixin::OciMetadata`. It uses the same
endpoint, the same auth header, the same three reads, the same volume keys and
the same 6-second timeout.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors. Its transport is the `HTTP` of the
  sources, so a fake host can answer the metadata reads.
