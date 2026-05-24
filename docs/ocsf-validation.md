# OCSF Validation

How to validate gohai's OCSF output and vendor extension against the upstream
OCSF schema.

## Prerequisites

- [uv](https://docs.astral.sh/uv/) — Python package runner
- A local clone of the [OCSF schema](https://github.com/ocsf/ocsf-schema)

## Step 1: Clone the OCSF schema

```bash
git clone --depth 1 https://github.com/ocsf/ocsf-schema.git /tmp/ocsf-schema
```

## Step 2: Compile the schema with our extension

gohai registers a vendor extension (uid 1337) at `schemas/ocsf-extension/`. The
OCSF schema compiler merges it with the base schema:

```bash
uvx ocsf-schema-compiler /tmp/ocsf-schema -e schemas/ocsf-extension > /tmp/ocsf-compiled.json
```

This should print:

```
INFO: Read extension "gohai" from directory: schemas/ocsf-extension
INFO: Extension "gohai" is patching "os" from base schema
INFO: Extension "gohai" is patching "device" from base schema
INFO: Extension "gohai" is patching "device_hw_info" from base schema
```

If compilation fails with a shadowing error, an attribute in our extension
conflicts with a base schema attribute. Remove the conflicting attribute from
`schemas/ocsf-extension/dictionary.json` — it's already defined upstream.

## Step 3: Generate OCSF output

```bash
gohai collect --format ocsf --pretty > ocsf-output.json
```

This produces an OCSF `inventory_info` event (class_uid 5001) with:

- Standard OCSF attributes on `device`, `device.os`, `device.hw_info`,
  `device.network_interfaces`, and `cloud` objects
- gohai extension (uid 1337) attributes for fields OCSF doesn't yet cover (e.g.
  `fqdn`, `cpu_flags`, `cpu_vulnerabilities`, `init_system`)

## Step 4: Validate the schema + extension

Copy our extension into the OCSF schema tree and run the validator:

```bash
cp -r schemas/ocsf-extension /tmp/ocsf-schema/extensions/gohai
uvx --from ocsf-validator python -m ocsf_validator /tmp/ocsf-schema
```

All 12 tests should PASS:

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

This confirms our gohai extension (uid 1337) is a valid OCSF extension —
dictionary attributes are correctly typed, object extensions reference valid
attributes, and the metaschema definitions are well-formed.

## Extension structure

```
schemas/ocsf-extension/
  extension.json              — uid 1337, name "gohai"
  dictionary.json             — attribute type definitions
  objects/
    device.json               — extends device with fqdn, machine_id, etc.
    device_hw_info.json        — extends hw_info with cpu_flags, etc.
    os.json                   — extends os with distribution_id, etc.
```

## Output formats

| Flag            | Format                    | Schema                      |
| --------------- | ------------------------- | --------------------------- |
| `--format ohai` | Collector-centric JSON    | `schemas/gohai.schema.json` |
| `--format ocsf` | OCSF inventory_info event | OCSF schema + extension     |

## Future work

- Automate event data validation once OCSF publishes a validator
- Submit gohai extension to the
  [OCSF extensions registry](https://github.com/ocsf/ocsf-schema/blob/main/extensions.md)
  for an official UID assignment
- Contribute gap candidates from `schemas/ocsf-gaps.md` as upstream OCSF PRs
