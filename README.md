# Jussive

Jussive is a Go framework for building human-friendly CLI tools with a small, discoverable agent surface.

The core idea is CLI-first discovery: agents learn one compact command surface, search a workspace catalog on demand, inspect only the command they need, then run it by stable command id or by human-facing path.

## Current Shape

Jussive provides:

- a Go runtime library in `pkg/jussive`
- a scaffolding and workspace CLI in `cmd/jussive`
- YAML command metadata loaded at runtime
- standard YAML/JSON structured output envelopes
- source-mode agent execution through `jussive agent ...`
- optional standalone CLI export through `jussive build`

Generated standalone CLIs are export artifacts. They are useful for distribution, but agents do not need them for local workspace use.

## Quick Start

Create a project:

```sh
go run ./cmd/jussive new my-tools
cd my-tools
```

Use the workspace agent surface without building a binary:

```sh
go run ../cmd/jussive agent search "focused tests"
go run ../cmd/jussive agent info test.focused
go run ../cmd/jussive agent schema test.focused --json
go run ../cmd/jussive agent run test.focused -- src/foo.ts
```

Run a human-facing command path from source mode:

```sh
go run ../cmd/jussive run test focused -- src/foo.ts
```

Validate the workspace:

```sh
go run ../cmd/jussive validate
go test ./...
```

Optionally export a standalone CLI:

```sh
go run ../cmd/jussive build
./bin/my-tools agent search "focused tests"
```

## Generated Project Layout

```text
my-tools/
  jussive.yaml
  CHANGELOG.md
  AGENTS.md
  cmd/my-tools/main.go
  commands/
  internal/build/
  internal/schemas/
  docs/
  tests/
```

`jussive.yaml` declares the workspace runtime:

```yaml
name: my-tools
runtime:
  kind: go
  entrypoint: ./cmd/my-tools
release:
  version_source: git_tag
  tag_prefix: v
```

## Agent Surface

The workspace-level agent commands are:

```sh
jussive agent search "<task intent>" --limit 5
jussive agent info <command-id>
jussive agent schema <command-id>
jussive agent run <command-id> -- [args...]
jussive agent doctor
```

Structured agent responses use the standard envelope:

```yaml
ok: true
data: {}
warnings: []
errors: []
```

YAML is the default. JSON is available with `--json` or `--output json`.

## Development

Run the framework tests:

```sh
go test ./...
```

Useful framework commands:

```sh
go run ./cmd/jussive --help
go run ./cmd/jussive new my-tools
go run ./cmd/jussive validate
go run ./cmd/jussive doctor
go run ./cmd/jussive docs
```

## Versioning

Jussive does not own generated project versioning. Host projects own SemVer and should normally inject build metadata at release time with Go linker flags or release tooling.

Generated CLIs expose:

```sh
my-tools --version
my-tools version
my-tools version --json
```
