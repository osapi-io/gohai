# Features

gohai provides a rich set of features for collecting and consuming system facts.

| Feature                                               | Description                                    |
| ----------------------------------------------------- | ---------------------------------------------- |
| [🔌 Pluggable Collectors](collectors.md)              | Enable/disable individual fact collectors      |
| [🏗️ Typed Structs](typed-structs.md)                  | Strongly-typed Go structs for all facts        |
| [📄 JSON Output](json-output.md)                      | Nested JSON output for CLI and programmatic use |
| [🗺️ Flat Map Access](flat-map.md)                     | Dot-separated key-value access                 |
| [🐧 Cross-Platform](cross-platform.md)                | Linux primary, macOS best-effort               |
| [🔗 Collector Dependencies](dependencies.md)          | Automatic dependency resolution between facts  |
| [⚡ Concurrent Collection](concurrency.md)            | Parallel collection with dependency ordering   |
| [🎛️ Profiles](profiles.md)                            | Predefined collector sets (minimal, standard, full) |
| [📊 Ohai Compatibility](ohai-compat.md)               | Output format compatible with Chef Ohai        |
| [🔌 SDK Integration](sdk.md)                          | Import as a Go package for OSAPI and others    |
