# OCSF validation

How to validate the gohai OCSF output and the vendor extension against the
upstream OCSF schema.

## Prerequisites

- [uv](https://docs.astral.sh/uv/). The Python package runner.
- A local clone of the [OCSF schema](https://github.com/ocsf/ocsf-schema).

## Step 1: clone the OCSF schema

```bash
git clone --depth 1 https://github.com/ocsf/ocsf-schema.git /tmp/ocsf-schema
```

## Step 2: compile the schema with our extension

gohai registers a vendor extension (uid 1337) at `schemas/ocsf-extension/`. The
OCSF schema compiler merges it with the base schema:

```bash
uvx ocsf-schema-compiler /tmp/ocsf-schema -e schemas/ocsf-extension > /tmp/ocsf-compiled.json
```

The compiler prints:

```
INFO: Read extension "gohai" from directory: schemas/ocsf-extension
INFO: Extension "gohai" is patching "os" from base schema
INFO: Extension "gohai" is patching "device" from base schema
INFO: Extension "gohai" is patching "device_hw_info" from base schema
```

If the compilation fails with a shadowing error, an attribute in our extension
has the same name as an attribute in the base schema. Remove that attribute from
`schemas/ocsf-extension/dictionary.json`. The upstream schema already defines
it.

## Step 3: generate OCSF output

```bash
gohai collect --format ocsf --pretty > ocsf-output.json
```

This command makes an OCSF `inventory_info` event (class_uid 5001) with:

- Standard OCSF attributes on the `device`, `device.os`, `device.hw_info`,
  `device.network_interfaces` and `cloud` objects.
- gohai extension (uid 1337) attributes for fields that OCSF does not have yet,
  for example `fqdn`, `cpu_flags`, `cpu_vulnerabilities` and `init_system`.

## Step 4: validate the schema + extension

Copy our extension into the OCSF schema tree. Then run the validator:

```bash
cp -r schemas/ocsf-extension /tmp/ocsf-schema/extensions/gohai
uvx --from ocsf-validator python -m ocsf_validator /tmp/ocsf-schema
```

All 12 tests must PASS:

```
PASSED: Schema definitions can be loaded
PASSED: Schema types can be inferred
PASSED: Check observable type_id definitions
PASSED: Dependency targets are resolvable and exist
PASSED: Required keys are present
PASSED: There are no unrecognized keys
PASSED: All attributes in the dictionary are used
PASSED: All attributes are defined in dictionary.json
PASSED: Names are not used multiple times within a record type
PASSED: Attribute type references are defined
PASSED: Event class categories are defined
PASSED: JSON files match their metaschema definitions
```

This result shows these facts:

- The gohai extension (uid 1337) is a valid OCSF extension.
- The dictionary attributes have the correct types.
- The object extensions refer to valid attributes.
- The metaschema definitions are correctly formed.

## Extension structure

```
schemas/ocsf-extension/
  extension.json              # uid 1337, name "gohai"
  dictionary.json             # the type definitions of the attributes
  objects/
    device.json               # adds fqdn, machine_id and more to device
    device_hw_info.json        # adds cpu_flags and more to hw_info
    os.json                   # adds distribution_id and more to os
```

## Output formats

| Flag            | Format                    | Schema                      |
| --------------- | ------------------------- | --------------------------- |
| `--format ohai` | JSON grouped by collector | `schemas/gohai.schema.json` |
| `--format ocsf` | OCSF inventory_info event | OCSF schema + extension     |

## Future work

- Validate the event data automatically when OCSF publishes a validator.
- Send the gohai extension to the
  [OCSF extensions registry](https://github.com/ocsf/ocsf-schema/blob/main/extensions.md)
  to get an official UID.
- Send the gap candidates from `schemas/ocsf-gaps.md` as upstream OCSF PRs.
