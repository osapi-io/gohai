# Features

gohai provides a rich set of features for collecting and consuming system facts.

| Feature                                       | Description                                                                               |
| --------------------------------------------- | ----------------------------------------------------------------------------------------- |
| [🔌 Pluggable Collectors](collectors.md)      | Enable/disable individual fact collectors                                                 |
| [🏗️ Typed Structs](typed-structs.md)          | Strongly-typed Go structs for all facts                                                   |
| [📄 JSON Output](json-output.md)              | Nested JSON output for CLI and programmatic use                                           |
| [🗺️ Flat Map Access](flat-map.md)             | Dot-separated key-value access                                                            |
| [🐧 Cross-Platform](cross-platform.md)        | Linux primary, macOS best-effort                                                          |
| [🔗 Collector Dependencies](dependencies.md)  | Automatic dependency resolution between facts                                             |
| [⚡ Concurrent Collection](concurrency.md)    | Collectors run concurrently; dependency graph resolves order when declared                |
| [🎛️ Profiles](profiles.md)                    | Predefined collector sets (minimal, standard, full)                                       |
| [📊 OCSF + OpenTelemetry + Ohai](ocsf-ohai.md) | Field names follow [OCSF](https://schema.ocsf.io/) then [OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/resource/); data sources mirror Chef Ohai plugins |
| [🔌 SDK Integration](sdk.md)                  | Import as a Go package for OSAPI and others                                               |
