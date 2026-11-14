# Azure

> **Status:** Implemented ✅

## Description

The collector reads the Azure VM metadata from the link-local Instance Metadata
Service at `http://169.254.169.254/metadata/instance`. Each request must have
the `Metadata: true` header. Azure rejects requests without it. At collect time,
the collector **negotiates an api-version** with the service. This is the same
as Ohai's `best_api_version` handshake. If `facts.Azure != nil`, the host is on
Azure.

Detection uses two Linux signals. They are the same as Ohai's chain for hosts
that are not Windows:

- `/usr/sbin/waagent` exists. This shows that the Azure Linux Agent is
  installed.
- `/var/lib/dhcp/dhclient.eth0.leases` contains the DHCP option-245 signature
  (`unknown-245`).

If neither signal is there, the collector stops and makes no HTTP call.

## Collected fields

| Field                            | Type                      | Description                                                                                                       | Schema mapping                 |
| -------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `id`                             | `string`                  | The VM UUID.                                                                                                      | OTel `cloud.resource_id`       |
| `name`                           | `string`                  | The VM name.                                                                                                      | OTel `host.name`               |
| `type`                           | `string`                  | The VM SKU, for example `Standard_D2s_v3`.                                                                        | OTel `host.type`               |
| `resource_id`                    | `string`                  | The full ARM resource ID.                                                                                         | No direct schema mapping.      |
| `resource_group_name`            | `string`                  | The resource group name.                                                                                          | No direct schema mapping.      |
| `vm_scale_set_name`              | `string`                  | The VMSS name, if the VM is part of a VMSS.                                                                       | No direct schema mapping.      |
| `priority`                       | `string`                  | `Regular`, `Low` or `Spot`.                                                                                       | No direct schema mapping.      |
| `eviction_policy`                | `string`                  | The spot eviction policy.                                                                                         | No direct schema mapping.      |
| `region`                         | `string`                  | The Azure region, for example `eastus`.                                                                           | OTel `cloud.region`            |
| `zone`                           | `string`                  | The availability zone, for example `1`.                                                                           | OTel `cloud.availability_zone` |
| `placement_group_id`             | `string`                  | The placement group.                                                                                              | No direct schema mapping.      |
| `platform_fault_domain`          | `string`                  | The fault domain index.                                                                                           | No direct schema mapping.      |
| `platform_update_domain`         | `string`                  | The update domain index.                                                                                          | No direct schema mapping.      |
| `account_uid`                    | `string`                  | The Azure subscription UUID.                                                                                      | OTel `cloud.account.id`        |
| `cloud_partition`                | `string`                  | For example `AzurePublicCloud` or `AzureUSGovernment`.                                                            | No direct schema mapping.      |
| `offer`                          | `string`                  | The Marketplace offer.                                                                                            | No direct schema mapping.      |
| `publisher`                      | `string`                  | The Marketplace publisher.                                                                                        | No direct schema mapping.      |
| `sku`                            | `string`                  | The Marketplace SKU.                                                                                              | No direct schema mapping.      |
| `version`                        | `string`                  | The Marketplace image version.                                                                                    | OTel `host.image.version`      |
| `license_type`                   | `string`                  | The BYO license type.                                                                                             | No direct schema mapping.      |
| `os_type`                        | `string`                  | `Linux` or `Windows`.                                                                                             | OTel `os.type`                 |
| `provider`                       | `string`                  | The Azure RP, usually `Microsoft.Compute`.                                                                        | No direct schema mapping.      |
| `plan`                           | `*Plan`                   | The Marketplace plan.                                                                                             | No direct schema mapping.      |
| `storage_profile`                | `*StorageProfile`         | The OS disk and the data disks. Refer to pkg.go.dev.                                                              | No direct schema mapping.      |
| `tags`                           | `string`                  | A list of `key:value` tags, with a semicolon between tags.                                                        | No direct schema mapping.      |
| `tags_list`                      | `[]Tag`                   | The parsed tags.                                                                                                  | No direct schema mapping.      |
| `user_data`                      | `string`                  | The custom data, in Base64.                                                                                       | No direct schema mapping.      |
| `custom_data`                    | `string`                  | The legacy custom data field.                                                                                     | No direct schema mapping.      |
| `is_host_compatibility_layer_vm` | `bool`                    | Shows if the VM is a compatibility-layer VM.                                                                      | No direct schema mapping.      |
| `security_profile`               | `*SecurityProfile`        | The secure boot, vTPM and encryption flags.                                                                       | No direct schema mapping.      |
| `public_keys`                    | `[]PublicKey`             | The SSH keys that the profile attaches.                                                                           | No direct schema mapping.      |
| `host`                           | `*Host`                   | The Azure Dedicated Host that the VM is pinned to (`{id}`). Empty on VMs on pooled hosts.                         | No direct schema mapping.      |
| `host_group`                     | `*HostGroup`              | The Azure Dedicated Host Group that contains the host.                                                            | No direct schema mapping.      |
| `os_profile`                     | `*OSProfile`              | The OS provisioning settings (`admin_username`, `computer_name`, `disable_password_authentication`).              | No direct schema mapping.      |
| `additional_capabilities`        | `*AdditionalCapabilities` | The feature switches (`hibernation_enabled`). The values are Azure string booleans (`"true"` or `"false"`).       | No direct schema mapping.      |
| `extended_location`              | `*ExtendedLocation`       | A special placement (`{name, type}`) for edge zones, Azure Arc or Azure Local.                                    | No direct schema mapping.      |
| `interfaces`                     | `map[string]Interface`    | The VNICs, with the MAC address as the key. The shape is the same as Ohai's `metadata.network.interfaces[<mac>]`. | OCSF `network_interface`       |
| `public_ipv4` / `local_ipv4`     | `[]string`                | Flat lists of the IP addresses of all interfaces.                                                                 | No direct schema mapping.      |
| `public_ipv6` / `local_ipv6`     | `[]string`                | Flat lists of the IPv6 addresses of all interfaces.                                                               | No direct schema mapping.      |

The details of the sub-structs (`Plan`, `StorageProfile`, `Disk`, `ManagedDisk`,
`Tag`, `SecurityProfile`, `PublicKey`, `Host`, `HostGroup`, `OSProfile`,
`AdditionalCapabilities`, `ExtendedLocation`, `Interface`, `IPAddrs`,
`IPAddress`, `Subnet`) are on pkg.go.dev.

## Platform support

| Platform | Supported                           |
| -------- | ----------------------------------- |
| Linux    | ✅                                  |
| macOS    | ✅ (useful only on an Azure VM)     |
| Other    | ✅ (only meaningful on an Azure VM) |

## Example output

```json
{
  "azure": {
    "id": "abcd-1234",
    "name": "web-1",
    "type": "Standard_D2s_v3",
    "region": "eastus",
    "zone": "1",
    "account_uid": "sub-uuid",
    "interfaces": {
      "000D3A...": {
        "mac_address": "000D3A...",
        "ipv4": { "ip_addresses": [{ "private_ip": "10.0.0.4" }] }
      }
    }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("azure"))
facts, _ := g.Collect(context.Background())

if facts.Azure != nil {
    fmt.Println(facts.Azure.AccountUID, facts.Azure.Region)
}
```

## Enable/disable

```bash
gohai collect --collector.azure      # enable (opt-in)
gohai collect --no-collector.azure   # disable (default)
gohai collect --category=cloud       # pulls this + all cloud collectors
```

## Dependencies

None. Detection uses file signals that are specific to Azure (the waagent binary
and DHCP option 245), not DMI.

## Data sources

1. **Detection gate:** the collector returns `(nil, nil)` unless one of these is
   true:
   - `/usr/sbin/waagent` exists. This shows that the Azure Linux Agent is
     installed. This check is the same as Ohai's `has_waagent?`.
   - `/var/lib/dhcp/dhclient.eth0.leases` contains `"unknown-245"`. This check
     is the same as Ohai's `has_dhcp_option_245?`.
2. **API version negotiation:** the collector sends `GET /metadata/instance`
   without `api-version`. Azure returns HTTP 400 with
   `{"newest-versions": [...]}`. The collector compares this list with its list
   of supported versions and selects the latest version in both. If the
   negotiation fails, the collector uses the latest hardcoded version. This is
   the same as Ohai's `best_api_version` in `mixin/azure_metadata.rb`.
3. **Metadata fetch:** `GET /metadata/instance?api-version=<negotiated>`.
4. **Required header:** `Metadata: true`. Azure rejects requests without it. The
   header gives protection against lateral SSRF.
5. **User-Agent:** `gohai` (the cloudmetadata default).
6. **Timeout:** 6 seconds. This is the same as Ohai's `read_timeout`.
7. **Failure handling:** if the transport fails, or Azure returns 404 or an
   other error, the collector returns `(nil, nil)`.
8. **Transformation:** the collector flattens the `compute` sub-object into the
   top-level `Info` (identity, placement, marketplace, tags, user-data). It
   keeps the optional nested records as typed sub-structs when Azure gives them:
   `host`, `host_group`, `os_profile`, `additional_capabilities`,
   `extended_location`, `plan`, `storage_profile`, `security_profile` and
   `public_keys`. It also collects the IP addresses of each interface from
   `network.interface[].ipv4/ipv6` into the flat top-level `public_ipv4`,
   `local_ipv4`, `public_ipv6` and `local_ipv6` lists. These lists are in
   addition to the map of interfaces, which has the MAC address as the key.

The method is the same as Ohai's `Ohai::Mixin::AzureMetadata`. It uses the same
detection chain (the signals for hosts that are not Windows), the same
version-negotiation handshake, the same 6s timeout and the same structure of
network interfaces, with the MAC as the key. The collector does not implement
the detection signals that are specific to Windows (`C:\WindowsAzure`, the
`DhcpDomain` registry key).

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/). The shared HTTP
  client of all cloud-provider collectors.
