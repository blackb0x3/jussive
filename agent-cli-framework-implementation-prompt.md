# Implementation Prompt: Jussive

You are implementing a new framework based on the accompanying plan:

```text
agent-cli-framework-implementation-plan.md
```

Treat that plan as the source of truth. This prompt gives the backstory, intent, and design rationale so you do not reopen decisions that have already been settled.

## Backstory

The goal is to build a CLI framework for creating human-friendly command-line tools that are also easy for coding agents to discover and use.

The original problem: reusable utilities are useful for agents, but dumping every tool, parameter, and schema into an agent's context does not scale. MCP-style tool exposure can create too much up-front context overhead. The preferred approach here is CLI-first: agents should learn one small discovery surface, search for relevant commands on demand, inspect only the chosen command, then run either the direct command path or a stable command id.

Think of the framework as:

- oclif-style command organization
- Taskfile/just-style discoverability
- first-class agent metadata
- no Node.js, JavaScript, or TypeScript
- no MCP dependency for v1

The chosen implementation stack is Go.

## Product Shape

The framework should provide two things:

1. A Go runtime library used by generated CLI projects.
2. A scaffolding CLI named `jussive`.

Example target workflow:

```sh
jussive new my-tools
cd my-tools

jussive add command test.focused --path "test focused"
jussive validate
jussive doctor

my-tools agent search "run focused tests"
my-tools agent info test.focused
my-tools agent schema test.focused
my-tools agent doctor
my-tools version
my-tools test focused src/foo.ts
```

The agent-facing surface should stay small:

```sh
<cli> agent search "<task intent>" --limit 5
<cli> agent info <command-id>
<cli> agent schema <command-id>
<cli> agent run <command-id> -- [args...]
<cli> agent doctor
```

## Settled Design Decisions

Do not reopen these unless implementation proves they are impossible.

- Use Go for v1.
- Jussive does not own or reimplement generated project versioning.
- The host project's release system owns SemVer.
- For Go projects, release versions should normally come from Git tags and build metadata.
- Do not add per-command or per-subcommand versions.
- Command paths support arbitrary depth greater than zero.
- Command ids and command paths are related but separate.
- YAML is canonical for agent-facing metadata.
- Go code is canonical for runtime command definitions and parameters.
- Every discoverable command must expose `agent schema`.
- Every agent-facing structured response uses the standard envelope.
- YAML is the default structured output format; JSON remains available through `--json` or `--output json`.
- `agent run` exists, but agents may use direct commands when clearer.
- Search index is loaded at runtime in v1 by scanning metadata.
- Deprecated parameter aliases should continue parsing when practical.
- Runnable parent nodes are disabled by default and require explicit opt-in.
- Excessive command depth is a warning by default, with optional hard limits.
- Risk metadata and dry-run conventions are part of v1.
- Generated projects should include a starter `AGENTS.md` snippet.
- Plugins, remote catalogs, embeddings, telemetry, MCP bridge, and web UI are deferred until after v1.

## Important Concepts

### Command IDs vs Command Paths

Command id:

```text
product.pricing.import.validate
```

Human-facing command path:

```sh
my-tools product pricing import validate
```

The id is stable for agent lookup. The path is optimized for human CLI usage.

### Versioning

Jussive should not create a second canonical version source for generated projects.

For generated Go CLIs:

- the host project's release process owns SemVer
- Git tags such as `v0.1.0` and `v1.0.0` are the normal source of release truth
- build metadata should be injected into the binary at build/release time
- `--version`, `version`, and `version --json` should report embedded version, commit, dirty state, and build time
- `jussive version check` and `jussive version plan` should inspect and advise, not mutate canonical version state

Do not add per-command versions and do not require a generated `VERSION` file.

### Arbitrary-Depth Commands

The framework should support any command depth:

```sh
my-tools a
my-tools a b
my-tools a b c d e f
my-tools product pricing import validate
```

Internally, command paths should normalize to token arrays:

```go
[]string{"product", "pricing", "import", "validate"}
```

### Runnable Parent Nodes

A node is a runnable parent when it is both executable and has subcommands:

```sh
my-tools product
my-tools product pricing
```

This is disabled by default and requires explicit opt-in in Go, for example:

```go
AllowRunnableParent: true
```

Agent metadata should expose clearer fields:

```json
{
  "is_runnable": true,
  "has_subcommands": true
}
```

### Structured Output Envelope

All agent-facing structured output should use the same envelope. YAML is the default:

```yaml
ok: true
data: {}
warnings: []
errors: []
```

JSON should remain available through `--json` or `--output json`.

### Exit Codes

Use the stable policy from the plan:

- `0`: success
- `1`: command failed
- `2`: invalid usage or invalid input
- `3`: validation failed
- `4`: command or command id not found
- `5`: unsafe or risky action blocked

## V1 Expectations

The first stable release should be deliberately boring and strict.

Prioritize:

- scaffolding that creates a working project
- stable metadata parsing and validation
- stable YAML outputs
- stable JSON outputs for explicit JSON mode
- formal JSON Schemas for the parsed data model
- runtime-loaded search
- arbitrary-depth command parsing
- generated docs
- generated `AGENTS.md` snippet
- host release metadata integration and version commands
- doctor/validation diagnostics
- golden tests for compatibility

Defer:

- plugin loading
- remote catalogs
- embeddings
- MCP bridge
- telemetry
- web UI
- complex semantic diffing beyond basic version-plan heuristics

## Implementation Guidance

Start with the smallest useful vertical slice:

1. Create the framework repository structure.
2. Implement command registration with arbitrary-depth paths.
3. Implement metadata loading from YAML.
4. Implement `agent search`, `agent info`, and `agent schema`.
5. Implement the structured output envelope and exit codes.
6. Implement `jussive new` with a generated sample project.
7. Implement `jussive validate` and `jussive doctor`.
8. Add golden tests for YAML output, explicit JSON output, and generated files.

Prefer simple, explicit implementation over clever abstractions. The framework is valuable only if generated CLIs are predictable for both humans and agents.

## Before Coding

Read the full implementation plan first:

```text
agent-cli-framework-implementation-plan.md
```

Then inspect the target repository state. If a repository already exists, preserve its conventions unless they conflict with the plan.

If you need to make tradeoffs, favor:

- stable public contracts
- clear generated code
- deterministic output
- easy review
- minimal dependencies
- cross-platform behavior

## Done Criteria

For the first useful implementation, a fresh generated project should be able to:

```sh
jussive new my-tools
cd my-tools
jussive validate
jussive doctor
go test ./...
my-tools --version
my-tools version
my-tools agent search "focused tests"
my-tools agent info test.focused
my-tools agent schema test.focused
```

The YAML outputs should match the standard envelope, explicit JSON mode should return the same data model, and the generated project should include:

- `CHANGELOG.md`
- `jussive.yaml`
- build metadata wiring for `--version`, `version`, and `version --json`
- command metadata files
- schema references
- generated docs support
- starter `AGENTS.md`

