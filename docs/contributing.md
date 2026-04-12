# Contributing

Contributions to gohai are very welcome, but we ask that you read this document
before submitting a PR.

## Before you start

- Read the [Development](development.md) guide for prerequisites, setup, code
  style, testing, and commit message conventions.
- **Check existing work** — Is there an existing PR? Are there issues discussing
  the feature/change you want to make? Please make sure you consider/address
  these discussions in your work.
- **Backwards compatibility** — Will your change break existing consumers of
  gohai? It is much more likely that your change will be merged if it is
  backwards compatible. Is there an approach you can take that maintains this
  compatibility? If not, consider opening an issue first so that API changes can
  be discussed before you invest your time into a PR.

## Making changes

- **Code style** — Follow the conventions described in the
  [Development](development.md#code-style) guide.
- **Documentation** — Ensure that you add/update any relevant documentation. For
  new collectors this means: a collector doc in `docs/collectors/`, an update to
  the collector index, and a README update. See
  [Adding a New Collector](../CLAUDE.md#adding-a-new-collector) for the full
  checklist.
- **Tests** — Ensure that you add/update any relevant tests and that all tests
  are passing before submitting the PR. All packages must reach 100% coverage.
  See [Development](development.md#testing) for how to run tests.

## Submitting a PR

- **Describe your changes** — Ensure that you provide a comprehensive
  description of your changes.
- **Issue/PR links** — Link any previous work such as related issues or PRs.
  Please describe how your changes differ to/extend this work.
- **Examples** — Add any examples or screenshots that you think are useful to
  demonstrate the effect of your changes.
- **Draft PRs** — If your changes are incomplete, but you would like to discuss
  them, open the PR as a draft and add a comment to start a discussion. Using
  comments rather than the PR description allows the description to be updated
  later while preserving any discussions.

## FAQ

> I want to contribute, where do I start?

All kinds of contributions are welcome, whether it's a typo fix or a shiny new
collector. You can also contribute by upvoting/commenting on issues or helping
to answer questions.

> I'm stuck, where can I get help?

If you have questions, feel free to open a [Discussion][] on GitHub.

[Discussion]: https://github.com/osapi-io/gohai/discussions
