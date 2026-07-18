# Jussive Implementation Plan

## 1. Objective

Build a CLI framework that lets developers create normal human-friendly CLI projects while also exposing a compact, searchable discovery surface for coding agents.

The framework should avoid the MCP-style problem of loading every tool definition into context. Agents should only need to know one small set of discovery commands, then search and inspect specific commands on demand.

## 2. Core Concept

The framework provides:

- A standard command structure for CLI projects.
- Metadata beside each command.
- Arbitrary-depth command paths made from one or more path segments.
- Integration with host-project SemVer and build metadata.
- Searchable command discovery by intent, description, tags, examples, and usage constraints.
- Machine-readable command details as YAML by default, with JSON available explicitly.
- Optional execution through a framework-managed `run` command.
- Human-friendly help output and shell completion.

The agent-facing surface should stay small:

```sh
<cli> agent search "<task intent>" --limit 5
<cli> agent info <command-id>
<cli> agent schema <command-id>
<cli> agent run <command-id> -- [args...]
```

Normal human usage should remain natural:

```sh
<cli> test focused src/foo.ts
<cli> git owners src/foo.ts
<cli> release notes --from main
<cli> product pricing import validate prices.csv
```

## 3. Recommended Stack

Use **Go** for the first implementation.

Reasons:

- Single static binary distribution is simple.
- Cross-platform support is strong.
- Startup time is low.
- JSON/YAML support is mature.
- CLI ergonomics are good.
- Implementation complexity is lower than Rust.
- The result is easier to adopt internally than a Python runtime-dependent tool.

Rust remains a good future option if correctness and strict compile-time guarantees become more important than implementation speed.

## 4. Target Users

Primary users:

- Developers building internal CLI utilities.
- Platform teams maintaining reusable automation.
- Coding agents that need discoverability without full tool-schema context bloat.

Secondary users:

- CI systems.
- Release engineers.
- Support engineers.
- QA and test automation owners.

## 5. Non-Goals

- Do not build an MCP server in v1.
- Do not build a web developer portal in v1.
- Do not require a database.
- Do not require Node.js, JavaScript, or TypeScript.
- Do not require agents to preload all command schemas.
- Do not make every CLI command agent-facing by default.

## 6. Framework Shape

The framework should support two layers:

1. **Runtime library**
   - Used by CLI authors to define commands and metadata.
   - Provides command registration, help, validation, and agent discovery output.

2. **Scaffolding CLI**
   - Used to create and maintain CLI projects.
   - Generates new projects, commands, metadata files, tests, and release config.

Example:

```sh
jussive new my-tools
cd my-tools

jussive add command test.focused --path "test focused"
jussive add command git.owners --path "git owners"
jussive validate
jussive build
```

The generated project then exposes its own binary:

```sh
my-tools agent search "run tests for changed files"
my-tools agent info test.focused
my-tools test focused src/foo.ts
```

Command paths can have any depth greater than zero:

```sh
my-tools product pricing import validate prices.csv
my-tools ops azure pipelines approve 12345
my-tools a b c d e f
```

The framework treats these as tokenized paths rather than a fixed parent/child command type.

## 7. Project Structure

Framework repository:

```text
jussive-framework/
  cmd/
    jussive/
      main.go
  internal/
    scaffold/
    registry/
    search/
    metadata/
    validation/
  pkg/
    jussive/
      app.go
      command.go
      metadata.go
      agent_commands.go
  templates/
    basic/
  docs/
  examples/
    my-tools/
```

Generated CLI project:

```text
my-tools/
  jussive.yaml
  CHANGELOG.md
  go.mod
  cmd/
    my-tools/
      main.go
  commands/
    test/
      focused.go
      focused.agent.yaml
    git/
      owners.go
      owners.agent.yaml
  internal/
    schemas/
  tests/
  AGENTS.md
```

## 8. Command Metadata

Each discoverable command gets an adjacent metadata file.

Example:

```yaml
id: test.focused
name: Focused test selector
summary: Finds and runs the smallest relevant test set for changed files.
tags:
  - test
  - validation
  - changed-files
  - dotnet
  - jest
risk: Runs local tests only.
risk_level: low
read_only: false
mutates_files: false
mutates_external_systems: false
supports_dry_run: false
requires_confirmation: false
command:
  path:
    - test
    - focused
inputs:
  - name: path
    required: true
    description: File or directory to analyze.
examples:
  - my-tools test focused src/foo.ts
when_to_use:
  - Need quick validation after a focused code edit.
  - Need to infer relevant tests from changed files.
when_not_to_use:
  - Need full CI-equivalent release validation.
  - Need performance or load testing.
```

Required fields:

- `id`
- `name`
- `summary`
- `command.path`

Do not include per-command versions in command metadata. Versioning is handled by the host project's release system and exposed through generated build metadata.

Recommended fields:

- `tags`
- `inputs`
- `examples`
- `when_to_use`
- `when_not_to_use`
- `risk`
- `risk_level`
- `read_only`
- `mutates_files`
- `mutates_external_systems`
- `supports_dry_run`
- `requires_confirmation`

Risk fields should use stable meanings:

| Field | Meaning |
|---|---|
| `risk_level` | One of `low`, `medium`, or `high`. |
| `read_only` | The command does not mutate local files or external systems. |
| `mutates_files` | The command can write, move, or delete local files. |
| `mutates_external_systems` | The command can mutate remote services, databases, tickets, cloud resources, or production systems. |
| `supports_dry_run` | The command supports a non-mutating preview mode. |
| `requires_confirmation` | The command should require explicit user confirmation before execution. |

Mutating commands should support `--dry-run` where practical. Validation should warn when a command declares mutation risk but does not support dry-run.

## 9. Metadata Schemas

The metadata and structured output formats are part of the public v1 API. The framework should ship formal JSON Schemas for the parsed data model, even when the default serialized output is YAML:

- `jussive.yaml`
- `*.agent.yaml`
- `agent schema <command-id>`
- `agent search`
- `agent info`
- `version`

Structured output commands should default to YAML and also support:

```sh
--json
--output json
--output yaml
```

Generated projects should include copies or references under:

```text
internal/schemas/
```

The schemas should be versioned with the project-level SemVer release of the framework. Generated projects should validate metadata against these schemas during:

```sh
jussive validate
jussive doctor
```

## 10. Parameter Contracts

Commands and subcommands should expose typed parameter contracts. Agents should be able to inspect parameters only when needed, rather than loading every command's full argument shape up front.

The command implementation should be the source of truth for runtime command definitions and parameters. YAML should be canonical for agent-facing metadata such as descriptions, examples, tags, risk metadata, and usage guidance. Argument parsing should not depend on hand-maintained YAML unless the command is fully declarative.

This avoids drift between:

- what the command actually accepts
- what help text says
- what agent metadata says
- what generated docs say

### Parameter Schema

Each discoverable command should be able to emit a schema like this:

```yaml
id: test.focused
command:
  path:
    - test
    - focused
parameters:
  - name: path
    type: path
    required: true
    position: 0
    description: File or directory to analyze.

  - name: framework
    type: enum
    values:
      - auto
      - dotnet
      - jest
      - pytest
    default: auto
    flag: --framework
    description: Test framework to use.

  - name: include_slow
    type: boolean
    default: false
    flag: --include-slow
    description: Include slow tests in the selected test set.

  - name: timeout
    type: duration
    default: 5m
    flag: --timeout
    description: Maximum time to allow the focused test run.
```

### Supported Parameter Types

Keep v1 types intentionally boring:

| Type | Example | Notes |
|---|---|---|
| `string` | `customer-123` | Default text type. |
| `integer` | `10` | Supports optional min/max validation. |
| `number` | `0.75` | Supports optional min/max validation. |
| `boolean` | `--verbose` | Flag-style parameter. |
| `enum` | `auto`, `jest` | Bounded set of values. |
| `path` | `src/foo.ts` | String with path semantics. |
| `duration` | `30s`, `5m` | Useful for timeouts and polling. |
| `string[]` | `--tag a --tag b` | Repeated string flag. |
| `path[]` | `--path src --path tests` | Repeated path flag. |

Avoid nested object parameters in v1. Commands that need structured input should accept one of:

```sh
<cli> command --json-input input.json
<cli> command --json-stdin
```

### Defining Parameters

The preferred Go API should define parameters with the command:

```go
app.Command(jussive.Command{
    ID:   "test.focused",
    Path: jussive.Path("test focused"),
    Parameters: []jussive.Parameter{
        jussive.Path("path").
            Position(0).
            Required().
            Description("File or directory to analyze."),
        jussive.Enum("framework").
            Flag("--framework").
            Values("auto", "dotnet", "jest", "pytest").
            Default("auto").
            Description("Test framework to use."),
        jussive.Bool("include_slow").
            Flag("--include-slow").
            Default(false).
            Description("Include slow tests in the selected test set."),
        jussive.Duration("timeout").
            Flag("--timeout").
            Default("5m").
            Description("Maximum time to allow the focused test run."),
    },
    Run: func(ctx context.Context, args jussive.Args) error {
        return runFocusedTests(ctx, args)
    },
})
```

The framework should use this single definition to generate:

- CLI parsing
- `--help` output
- `agent schema`
- generated markdown docs
- validation fixtures

### Adding Parameters

Adding an optional parameter is non-breaking.

Adding a required parameter is breaking unless:

- it has a safe default
- it is only required under a new mode
- the host project makes a major release bump and the breaking change is documented

The scaffolding CLI should support:

```sh
jussive add parameter test.focused framework --type enum --flag --framework
jussive add parameter test.focused timeout --type duration --flag --timeout --default 5m
```

### Removing Or Renaming Parameters

Removing a required parameter is breaking.

Renaming should be handled as:

1. add the new parameter
2. deprecate the old parameter
3. keep parsing the old parameter temporarily when practical
4. remove the old parameter in a later major host-project release

Example:

```yaml
parameters:
  - name: test_project
    flag: --test-project
    type: path
    deprecated: true
    replaced_by: --project
    removal_after: "2026-10-01"
```

The agent-facing schema should expose deprecation information so agents can avoid obsolete parameters.

Deprecated parameter aliases should continue parsing during the deprecation window when practical. The command should emit a warning that names the replacement parameter.

### Versioning Parameter Contracts

Commands and subcommands do not have independent versions. The generated CLI project has one release version from the host project's release system, and any public command or subcommand contract change should be reflected by the next host-project release.

Version rules:

- patch releases cover bug fixes, documentation clarifications, metadata fixes, and non-behavioral schema corrections
- minor releases cover backward-compatible command additions, subcommand additions, and optional parameter additions
- major releases cover breaking command contract changes, such as removed commands, removed parameters, renamed parameters, changed required parameters, or materially changed defaults
- implementation-only changes do not require a version change unless they are released

Generated projects should integrate with host-project SemVer from the start. Avoid per-command versions because they add bookkeeping without much benefit for normal CLI users.

## 11. Agent Discovery Commands

### `agent search`

Search command metadata by natural-language intent.

```sh
my-tools agent search "find the owner of this file" --limit 5
```

Returns:

```yaml
ok: true
data:
  query: find the owner of this file
  results:
    - id: git.owners
      name: Code owner lookup
      summary: Finds likely owners and escalation contacts for a file path.
      tags:
        - git
        - ownership
        - support
      score: 12.4
warnings: []
errors: []
```

### `agent info`

Return full metadata for one command.

```sh
my-tools agent info git.owners
```

### `agent schema`

Return machine-readable argument and output schema.

```sh
my-tools agent schema git.owners
```

This is mandatory for every discoverable command in v1. Simple commands can expose a minimal schema:

```yaml
ok: true
data:
  parameters: []
  accepts_passthrough_args: true
warnings: []
errors: []
```

### `agent run`

Execute a discoverable command by id.

```sh
my-tools agent run git.owners -- src/foo.ts
```

This is useful for agents because it decouples stable command ids from human-facing command paths. It should pass through exit codes and stderr.

`agent run` is part of v1, but agents are not required to use it. Direct command execution is often clearer and should remain the preferred path when `agent info` returns an obvious invocation. `agent run` is the fallback when stable command ids are safer than human-facing command paths.

## 12. Structured Output Contract

Every agent-facing structured response should use the same envelope. YAML is the default serialization:

```yaml
ok: true
data: {}
warnings: []
errors: []
```

JSON remains available through `--json` or `--output json` for tools that need it.

Rules:

- `ok` is `true` only when the command completed successfully.
- `data` contains the command-specific payload.
- `warnings` contains non-fatal issues with stable warning codes where practical.
- `errors` contains fatal issues with stable error codes and human-readable messages.
- stderr is still used for human-readable diagnostics.
- machine-readable diagnostics belong in the structured output envelope.

## 13. Exit Code Policy

Use stable exit codes:

| Code | Meaning |
|---:|---|
| `0` | Success. |
| `1` | Command failed. |
| `2` | Invalid usage or invalid input. |
| `3` | Validation failed. |
| `4` | Command or command id not found. |
| `5` | Unsafe or risky action blocked. |

Additional exit codes should be avoided in v1 unless a strong need appears.

## 14. Command Tree Model

The framework should support any number of arbitrary-depth command paths, where each path has one or more segments.

Examples:

```sh
my-tools a
my-tools a b
my-tools a b c d e f
my-tools product pricing import validate
my-tools ops azure pipelines approve
```

Internally, command paths should be represented as token arrays:

```go
type Command struct {
    ID   string
    Path []string
    Run  Handler
}
```

String paths can be accepted as a convenience, but the normalized internal model should always be `[]string`.

```go
app.Command(jussive.Command{
    ID:   "product.pricing.import.validate",
    Path: []string{"product", "pricing", "import", "validate"},
    Run:  runValidatePricingImport,
})
```

Convenience helper:

```go
app.Command(jussive.Command{
    ID:   "product.pricing.import.validate",
    Path: jussive.Path("product pricing import validate"),
    Run:  runValidatePricingImport,
})
```

Parent nodes can be implicit namespaces. This should be valid even if only the final command is runnable:

```text
product
product pricing
product pricing import
product pricing import validate
```

A node may be both runnable and a parent only when explicitly allowed:

```go
app.Command(jussive.Command{
    ID:                  "product.pricing",
    Path:                jussive.Path("product pricing"),
    AllowRunnableParent: true,
    Run:                 runPricingOverview,
})
```

This avoids ambiguous help output and argument parsing.

Runnable parent nodes are disabled by default. When enabled, generated docs and agent metadata should expose the behavior with clear fields rather than the internal Go flag name:

```json
{
  "id": "product",
  "path": ["product"],
  "is_runnable": true,
  "has_subcommands": true,
  "subcommands": [
    {
      "id": "product.pricing",
      "path": ["product", "pricing"],
      "summary": "Pricing tools."
    }
  ]
}
```

Command depth is unlimited by default, but validation should warn on excessive nesting because deeply nested command paths reduce human ergonomics. Projects may configure a hard maximum depth if they want stricter conventions.

Agent discovery should stay flat by command id and search ranking. Agents should not need to walk the command tree manually.

```sh
my-tools agent search "validate product pricing import"
my-tools agent info product.pricing.import.validate
```

The tree exists primarily for human ergonomics, help output, shell completion, and direct invocation.

## 15. SemVer And Release Metadata

Jussive should not own or reimplement project versioning. The generated CLI project's release system is canonical.

For Go projects, the canonical release version should normally come from Git tags:

```text
v0.1.0
v1.0.0
v1.2.3
```

Generated projects should include build-time version metadata hooks, but not a required `VERSION` file. A typical build can inject metadata with `-ldflags` or a release tool such as GoReleaser.

Example build metadata fields:

```text
version
commit
built_at
dirty
```

Example `jussive.yaml`:

```yaml
name: my-tools
changelog: CHANGELOG.md
release:
  version_source: git_tag
  tag_prefix: v
```

### Project Version

The project version describes the released CLI binary as a whole. It is read from the host project's release metadata, not from Jussive-owned state.

Project version rules:

- `MAJOR` changes when the CLI release contains breaking changes to public command behavior
- `MINOR` changes when new backward-compatible commands or parameters are added
- `PATCH` changes when bugs, docs, metadata, or internal behavior are fixed without changing public contracts

Generated binaries should expose:

```sh
my-tools --version
my-tools version
```

Example JSON:

```json
{
  "name": "my-tools",
  "version": "v0.1.0",
  "commit": "abc1234",
  "dirty": false,
  "built_at": "2026-07-17T10:00:00Z"
}
```

### Command Compatibility

Commands and subcommands inherit the project version. There are no independent command or subcommand versions.

Command and parameter changes should still be visible in:

- `CHANGELOG.md`
- generated command docs
- `agent info <command-id>`
- `agent schema <command-id>`
- `jussive version plan`

This keeps the compatibility story simple: if any public command or subcommand contract changes, the next host-project release should reflect that change.

### Version Commands

The scaffolding CLI should provide:

```sh
jussive version check
jussive version plan
```

Generated projects should also expose:

```sh
my-tools version
my-tools version
```

### Release Notes

`jussive version plan` should inspect command metadata changes and suggest the release bump the host project should make:

- new command -> project minor
- removed command -> project major
- new optional parameter -> project minor
- new required parameter -> project major
- removed or renamed parameter -> project major
- documentation or metadata-only clarification -> project patch

The user and host project's release tooling remain in control of the actual version bump. Jussive should make unsafe versioning obvious, but it should not mutate canonical version state.

## 16. Search Strategy

Start simple:

- Tokenize query and metadata.
- Use BM25 or a lightweight fuzzy scoring algorithm.
- Search over:
  - id
  - name
  - summary
  - tags
  - examples
  - inputs
  - when_to_use
  - when_not_to_use

Later enhancements:

- Optional local embeddings index.
- Synonym support.
- Per-repo aliases.
- Usage telemetry to improve ranking.
- Negative matching against `when_not_to_use`.

Do not require embeddings in v1. Keyword search is transparent, deterministic, and easy to debug.

The search index should be loaded at runtime in v1 by scanning metadata files. This avoids stale generated indexes and keeps the implementation simple. A build-time cache such as `jussive index build` can be added later if startup time becomes a real problem.

## 17. Runtime API Sketch

Example Go API:

```go
package main

import (
    "context"
    "github.com/example/jussive/pkg/jussive"
)

func main() {
    app := jussive.New("my-tools")

    app.Command(jussive.Command{
        ID:   "test.focused",
        Path: jussive.Path("test focused"),
        Parameters: []jussive.Parameter{
            jussive.Path("path").
                Position(0).
                Required().
                Description("File or directory to analyze."),
            jussive.Enum("framework").
                Flag("--framework").
                Values("auto", "dotnet", "jest", "pytest").
                Default("auto").
                Description("Test framework to use."),
        },
        Run: func(ctx context.Context, args jussive.Args) error {
            return runFocusedTests(ctx, args)
        },
    })

    app.Run()
}
```

Metadata can be loaded from YAML files, from Go structs, or both.

Prefer this split for v1:

- command parameters live in Go code
- agent descriptions, tags, examples, risk metadata, and usage guidance live in YAML
- validation ensures both sources agree on command id and path
- generated docs merge both sources

## 18. Scaffolding Commands

The `jussive` framework CLI should support:

```sh
jussive new <name>
jussive add command <id> --path "<segments...>"
jussive add parameter <command-id> <name>
jussive deprecate parameter <command-id> <name>
jussive validate
jussive doctor
jussive list
jussive build
jussive docs
jussive agents snippet
jussive template diff
jussive template upgrade
jussive version check
jussive version plan
```

### `new`

Creates a new CLI project from a template.

Generated projects should include:

- `CHANGELOG.md`
- `jussive.yaml`
- build metadata wiring for `--version`, `version`, and `version --json`
- initial metadata/schema files
- a starter `AGENTS.md` snippet
- schema references under `internal/schemas/`

### `add command`

Creates:

- command implementation stub
- metadata YAML
- unit test stub
- docs stub

Example:

```sh
jussive add command product.pricing.import.validate --path "product pricing import validate"
```

The command id and command path are related but separate. The id is the stable agent-facing identifier; the path is the human-facing invocation route.

### `add parameter`

Adds a typed parameter to a command definition and updates generated metadata/docs.

Example:

```sh
jussive add parameter test.focused framework --type enum --flag --framework --values auto,dotnet,jest,pytest --default auto
```

### `deprecate parameter`

Marks a parameter as deprecated and optionally records its replacement.

Example:

```sh
jussive deprecate parameter test.focused test_project --replaced-by --project --removal-after 2026-10-01
```

### `validate`

Checks:

- unique command ids
- required metadata fields
- valid command paths
- examples parse cleanly
- no discoverable command lacks a summary
- no command has stale metadata
- parameter metadata matches the command implementation
- deprecated parameters include replacement or removal guidance
- metadata conforms to the published JSON Schemas
- every discoverable command exposes `agent schema`
- mutating commands declare risk metadata
- mutating commands either support `--dry-run` or document why not

### `docs`

Generates markdown documentation from metadata.

This keeps human docs and agent metadata aligned.

### `doctor`

Runs a diagnostic pass over the current project.

Checks should include:

- metadata validity
- schema validity
- PATH and executable resolution
- release metadata availability
- malformed command trees
- reserved namespace conflicts
- broken examples
- missing or stale generated docs
- missing JSON schema output
- mutating commands without dry-run support

Generated CLIs should also expose:

```sh
my-tools agent doctor
```

### `agents snippet`

Generates or refreshes the recommended `AGENTS.md` snippet for the project.

### `template diff`

Shows differences between the generated project and the current framework template.

### `template upgrade`

Applies safe template updates where possible. Risky changes should be shown as a diff and require explicit approval outside the tool.

### `version check`

Checks release metadata availability, SemVer tag validity when tags are present, and consistency with detected command, parameter, schema, and metadata changes.

### `version plan`

Suggests a host-project release bump based on command, parameter, schema, and metadata changes.

## 19. Validation Rules

Validation should fail on:

- duplicate command ids
- duplicate full command paths
- empty command path segments
- command paths with zero segments
- reserved top-level namespaces such as `agent`, `help`, `version`, `completion`, and `doctor`
- missing required metadata
- invalid YAML
- metadata that does not satisfy the published JSON Schemas
- invalid SemVer Git tag when release tags are present
- missing build metadata wiring for generated version commands
- command paths that do not resolve
- examples that reference unknown command paths
- declared parameters with invalid types
- enum parameters without values
- required parameters with impossible defaults
- discoverable commands that do not expose `agent schema`
- malformed structured output envelopes
- duplicate flags or duplicate positional indexes
- metadata parameters that do not match the implementation
- breaking parameter changes without a recommended major release bump
- breaking public command changes without a recommended major release bump

Validation should warn on:

- missing tags
- missing examples
- very long summaries
- vague descriptions such as "does stuff" or "helper command"
- too many tags
- no `when_not_to_use` for risky commands
- missing risk metadata
- mutating commands without `--dry-run`
- high-risk commands without `requires_confirmation: true`
- removed optional parameters referenced by examples or docs
- renamed parameters without a deprecation window
- changed defaults on commands used by generated examples
- excessive command nesting that may reduce human ergonomics
- runnable parent nodes that do not explicitly set `AllowRunnableParent`
- new backward-compatible parameters without a recommended minor release bump
- metadata-only command changes without at least a recommended patch release bump when published docs change

## 20. Parameter Change Workflow

Developers should use the framework tooling for parameter changes when possible.

### Add Optional Parameter

```sh
jussive add parameter test.focused timeout --type duration --flag --timeout --default 5m
jussive validate
```

Expected result:

- command parser accepts the new flag
- `agent schema` includes the new parameter
- docs update
- examples remain valid
- Jussive should recommend a minor release bump before release

### Add Required Parameter

```sh
jussive add parameter test.focused project --type path --flag --project --required
jussive validate
```

Expected result:

- validation warns or fails unless release planning accounts for the breaking change
- Jussive should recommend a major release bump before release
- examples must be updated
- release notes should include the breaking change

### Rename Parameter

```sh
jussive add parameter test.focused project --type path --flag --project
jussive deprecate parameter test.focused test_project --replaced-by --project --removal-after 2026-10-01
jussive validate
```

Expected result:

- both flags can parse during the deprecation window when practical
- agent schema marks the old flag as deprecated
- search/info output recommends the new flag
- Jussive should recommend a minor release bump while the old flag remains supported
- final removal should recommend a major release bump

### Remove Parameter

```sh
jussive remove parameter test.focused test_project
jussive validate
```

Expected result:

- validation checks examples, docs, and generated schemas
- Jussive should recommend a major release bump before release when removal is public
- release notes should include migration guidance

## 21. Agent-Facing Design Principles

Keep the always-visible instructions tiny.

Recommended `AGENTS.md` snippet for a CLI project:

```md
Internal CLI Tools

Before hand-rolling automation, search the CLI catalog:

`my-tools agent search "<task intent>" --limit 5`

Inspect a candidate before use:

`my-tools agent info <command-id>`

Prefer direct commands shown by `agent info`. Use `my-tools agent run` when a stable command id is safer than a human-facing command path.
```

Agent output rules:

- Structured output must be compact, stable, YAML by default, and wrapped in the standard envelope.
- Errors go to stderr.
- Exit codes must follow the stable exit-code policy.
- Commands should support `--help`.
- Every discoverable command must support `agent schema`.
- Mutating commands should support `--dry-run`.
- Commands should avoid interactive prompts unless explicitly requested.

## 22. Milestones

### Milestone 1: Prototype

Deliverables:

- Basic Go CLI framework.
- Static command registration.
- Arbitrary-depth command path registration.
- Generated build metadata wiring.
- Generated metadata JSON Schemas.
- Standard structured output envelope.
- Stable exit-code policy.
- YAML metadata loading.
- `agent search`.
- `agent info`.
- `agent schema`.
- `agent doctor`.
- Example generated CLI.

Success criteria:

- A generated CLI can expose at least three discoverable commands.
- At least one generated command uses a path deeper than two segments.
- A generated CLI exposes `--version`, `version`, and `version --json`.
- Every agent-facing structured response uses the standard envelope.
- Every discoverable command exposes at least a minimal schema.
- An agent can search by intent and inspect a command without seeing all command details up front.

### Milestone 2: Scaffolding

Deliverables:

- `jussive new`.
- `jussive add command`.
- `jussive validate`.
- `jussive doctor`.
- `jussive version check`.
- `jussive agents snippet`.
- `jussive template diff`.
- `jussive template upgrade`.
- Project template.
- Basic documentation generation.

Success criteria:

- A developer can create a working CLI project in under five minutes.
- A developer can add a command with any path depth greater than zero.
- Metadata validation catches common mistakes.
- Version validation catches invalid release tags and unsafe release bumps.
- Generated projects include a usable `AGENTS.md` snippet.
- Template diffing identifies drift from the current template.

### Milestone 3: Execution and Schemas

Deliverables:

- `agent run`.
- Typed command input schema.
- Parameter add/deprecate scaffolding.
- SemVer-aware parameter change validation.
- Risk metadata validation.
- Dry-run convention validation.
- Better error handling.
- Shell completion.
- Cross-platform test coverage.

Success criteria:

- Commands can be invoked by stable id.
- Agents can inspect expected inputs before execution.
- Agents can see deprecated parameters and preferred replacements.
- Agents can inspect release/build metadata.
- Agents can inspect risk metadata before execution.

### Milestone 4: Search Quality

Deliverables:

- BM25 or equivalent ranking.
- Tag weighting.
- Negative matching from `when_not_to_use`.
- Search result explanations.

Success criteria:

- Search returns the expected command in the top three for a curated query set.

### Milestone 5: Packaging

Deliverables:

- GitHub Actions release pipeline.
- Windows, Linux, and macOS binaries.
- Install script.
- Versioned templates.
- SemVer release workflow.
- v1.0.0 compatibility checklist.

Success criteria:

- Users can install the framework with a single command or downloaded binary.
- A generated project can pass `jussive validate`, `jussive doctor`, and the golden structured-output compatibility tests.

## 23. Testing Strategy

Unit tests:

- metadata parsing
- JSON Schema validation
- validation rules
- search ranking
- command id resolution
- command path tree construction
- arbitrary-depth command parsing
- runnable parent validation
- structured output shape
- structured output envelope behavior
- exit-code mapping
- parameter parsing
- schema generation
- parameter deprecation behavior
- SemVer parsing and bump planning
- build metadata consistency
- risk metadata validation
- dry-run warning behavior

Integration tests:

- generated project builds
- generated command runs
- generated deep command paths run
- `agent search` returns expected results
- `agent info` returns full metadata
- `agent schema` returns a schema for every discoverable command
- `agent doctor` reports project health
- `agent run` passes arguments and exit codes correctly
- parameter add/deprecate workflow updates generated outputs correctly
- generated projects expose `--version`, `version`, and `version --json`
- generated `AGENTS.md` snippet references the right CLI name
- template diff detects drift

Golden tests:

- stable YAML output
- stable JSON output for explicit JSON mode
- standard structured output envelope
- generated docs
- scaffolded file structure
- generated command schemas
- generated version metadata
- generated metadata schemas
- generated `AGENTS.md` snippet

Cross-platform tests:

- Windows PowerShell
- Linux bash
- macOS zsh

## 24. V1.0.0 Release Criteria

The first stable release should be boring and strict.

Required for v1.0.0:

- project scaffolding with host release metadata integration
- arbitrary-depth command path registration
- mandatory `agent schema` for every discoverable command
- formal JSON Schemas for project config, command metadata, and agent-facing structured outputs
- standard structured output envelope for every agent-facing structured response
- stable exit-code policy
- `agent search`, `agent info`, `agent schema`, `agent run`, and `agent doctor`
- `jussive validate`, `jussive doctor`, `jussive docs`, `jussive version check`, and `jussive version plan`
- generated `AGENTS.md` snippet
- risk metadata and dry-run validation
- generated docs from metadata
- golden compatibility tests for JSON and docs
- Windows, Linux, and macOS release artifacts

Explicitly defer until after v1.0.0:

- embeddings
- plugin loading
- remote catalogs
- MCP bridge
- telemetry
- web UI
- automatic semantic diffing beyond basic version-plan heuristics

## 25. Recommended First Build

Build the smallest useful version:

```sh
jussive new my-tools
jussive add command test.focused --path "test focused"
my-tools agent search "run focused tests"
my-tools agent info test.focused
my-tools agent schema test.focused
my-tools agent doctor
my-tools version
my-tools test focused src/foo.ts
```

Avoid plugin architecture, embeddings, remote catalogs, MCP integration, telemetry, and web UI until the core developer experience feels solid.

The framework wins if a developer can add a command once and get all of this for free:

- human help
- agent search
- YAML metadata
- typed parameter schema
- generated docs
- validation
- stable command id
