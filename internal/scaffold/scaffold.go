package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type Options struct {
	Name         string
	FrameworkDir string
}

func NewProject(dir string, opts Options) error {
	if opts.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if opts.FrameworkDir == "" {
		opts.FrameworkDir = ".."
	}
	files := map[string]string{
		"go.mod":       render(goModTemplate, opts),
		"go.sum":       goSumTemplate,
		"jussive.yaml": render(projectConfigTemplate, opts),
		"CHANGELOG.md": render(changelogTemplate, opts),
		"AGENTS.md":    render(agentsTemplate, opts),
		filepath.Join("cmd", opts.Name, "main.go"):                                       render(mainTemplate, opts),
		filepath.Join("internal", "build", "build.go"):                                   buildTemplate,
		filepath.Join("commands", "register.go"):                                         render(registerTemplate, opts),
		filepath.Join("commands", "test", "focused.agent.yaml"):                          render(focusedMetadataTemplate, opts),
		filepath.Join("commands", "git", "owners.agent.yaml"):                            render(ownersMetadataTemplate, opts),
		filepath.Join("commands", "product", "pricing", "import", "validate.agent.yaml"): render(pricingMetadataTemplate, opts),
		filepath.Join("internal", "schemas", "jussive.schema.json"):                      jussiveSchema,
		filepath.Join("internal", "schemas", "command.agent.schema.json"):                commandAgentSchema,
		filepath.Join("internal", "schemas", "envelope.schema.json"):                     envelopeSchema,
		filepath.Join("docs", "commands.md"):                                             render(commandsDocTemplate, opts),
		filepath.Join("tests", "golden_test.go"):                                         render(goldenTestTemplate, opts),
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func AddCommand(projectRoot, id, pathText string) error {
	segments := strings.Fields(pathText)
	if id == "" || len(segments) == 0 {
		return fmt.Errorf("command id and --path are required")
	}
	fileBase := safeIdentifier(id)
	goPath := filepath.Join(projectRoot, "commands", fileBase+".go")
	metaPath := filepath.Join(projectRoot, "commands", fileBase+".agent.yaml")
	if err := os.WriteFile(goPath, []byte(render(commandStubTemplate, map[string]any{
		"Func": safeExport(id),
		"ID":   id,
		"Path": pathText,
	})), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(metaPath, []byte(render(commandMetadataStubTemplate, map[string]any{
		"ID":       id,
		"Name":     strings.Title(strings.ReplaceAll(id, ".", " ")),
		"PathYAML": yamlPath(segments),
	})), 0o644); err != nil {
		return err
	}
	registerPath := filepath.Join(projectRoot, "commands", "register.go")
	b, err := os.ReadFile(registerPath)
	if err != nil {
		return err
	}
	insert := fmt.Sprintf("\n\tapp.Command(%s())", safeExport(id))
	marker := []byte("\n}\n\nfunc runFocused")
	idx := bytes.Index(b, marker)
	if idx < 0 {
		return fmt.Errorf("could not update %s", registerPath)
	}
	updated := append([]byte{}, b[:idx]...)
	updated = append(updated, []byte(insert)...)
	updated = append(updated, b[idx:]...)
	return os.WriteFile(registerPath, updated, 0o644)
}

func AgentSnippet(name string) string {
	return render(agentsTemplate, Options{Name: name})
}

func render(tmpl string, data any) string {
	t := template.Must(template.New("template").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func safeIdentifier(id string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	out := strings.Trim(re.ReplaceAllString(id, "_"), "_")
	if out == "" {
		return "command"
	}
	return strings.ToLower(out)
}

func safeExport(id string) string {
	parts := strings.Split(safeIdentifier(id), "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return "New" + strings.Join(parts, "") + "Command"
}

func yamlPath(path []string) string {
	var b strings.Builder
	for _, segment := range path {
		fmt.Fprintf(&b, "    - %s\n", segment)
	}
	return strings.TrimRight(b.String(), "\n")
}

const goModTemplate = `module {{.Name}}

go 1.22

require (
	github.com/blackb0x3/jussive v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/blackb0x3/jussive => {{.FrameworkDir}}
`

const goSumTemplate = `gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
`

const projectConfigTemplate = `name: {{.Name}}
changelog: CHANGELOG.md
runtime:
  kind: go
  entrypoint: ./cmd/{{.Name}}
release:
  version_source: git_tag
  tag_prefix: v
`

const changelogTemplate = `# Changelog

## Unreleased

- Initial generated CLI scaffold.
`

const agentsTemplate = `# Internal CLI Tools

Before hand-rolling automation, search the CLI catalog:

` + "`jussive agent search \"<task intent>\" --limit 5`" + `

Inspect a candidate before use:

` + "`jussive agent info <command-id>`" + `

Prefer direct commands shown by ` + "`agent info`" + `. Use ` + "`jussive agent run`" + ` when a stable command id is safer than a human-facing command path.

Standalone ` + "`{{.Name}}`" + ` binaries are optional export artifacts, not required for local agent use.
`

const mainTemplate = `package main

import (
	"context"
	"os"

	"{{.Name}}/commands"
	"{{.Name}}/internal/build"
	"github.com/blackb0x3/jussive/pkg/jussive"
)

func main() {
	app := jussive.New("{{.Name}}")
	app.Version = build.Info("{{.Name}}")
	commands.Register(app)
	os.Exit(app.Run(context.Background(), os.Args[1:]))
}
`

const buildTemplate = `package build

import (
	"strconv"

	"github.com/blackb0x3/jussive/pkg/jussive"
)

var (
	Version = "dev"
	Commit  = "unknown"
	BuiltAt = "unknown"
	Dirty   = "false"
)

func Info(name string) jussive.BuildInfo {
	dirty, _ := strconv.ParseBool(Dirty)
	return jussive.BuildInfo{
		Name:    name,
		Version: Version,
		Commit:  Commit,
		Dirty:   dirty,
		BuiltAt: BuiltAt,
	}
}
`

const registerTemplate = `package commands

import (
	"context"
	"fmt"

	"github.com/blackb0x3/jussive/pkg/jussive"
)

func Register(app *jussive.App) {
	app.Command(jussive.Command{
		ID:      "test.focused",
		Path:    jussive.Path("test focused"),
		Summary: "Finds and runs the smallest relevant test set for changed files.",
		Parameters: []jussive.Parameter{
			jussive.PathParam("path").Position(0).Required().Description("File or directory to analyze.").Build(),
			jussive.Enum("framework").Flag("--framework").Values("auto", "dotnet", "jest", "pytest").Default("auto").Description("Test framework to use.").Build(),
			jussive.Duration("timeout").Flag("--timeout").Default("5m").Description("Maximum time to allow the focused test run.").Build(),
		},
		Run: runFocused,
	})
	app.Command(jussive.Command{
		ID:      "git.owners",
		Path:    jussive.Path("git owners"),
		Summary: "Finds likely owners and escalation contacts for a file path.",
		Parameters: []jussive.Parameter{
			jussive.PathParam("path").Position(0).Required().Description("File or directory to inspect.").Build(),
		},
		Run: runOwners,
	})
	app.Command(jussive.Command{
		ID:      "product.pricing.import.validate",
		Path:    jussive.Path("product pricing import validate"),
		Summary: "Validates a product pricing import file before release.",
		Parameters: []jussive.Parameter{
			jussive.PathParam("file").Position(0).Required().Description("Pricing import file to validate.").Build(),
			jussive.Bool("dry_run").Flag("--dry-run").Default(false).Description("Preview validation without writing files.").Build(),
		},
		Run: runPricingImportValidate,
	})
}

func runFocused(ctx context.Context, args jussive.Args) error {
	_ = ctx
	fmt.Printf("focused tests for %s\n", args.String("path"))
	return nil
}

func runOwners(ctx context.Context, args jussive.Args) error {
	_ = ctx
	fmt.Printf("owners for %s\n", args.String("path"))
	return nil
}

func runPricingImportValidate(ctx context.Context, args jussive.Args) error {
	_ = ctx
	mode := "apply"
	if args.Bool("dry_run") {
		mode = "dry-run"
	}
	fmt.Printf("pricing import validation for %s (%s)\n", args.String("file"), mode)
	return nil
}
`

const focusedMetadataTemplate = `id: test.focused
name: Focused test selector
summary: Finds and runs the smallest relevant test set for changed files.
tags:
  - test
  - validation
  - changed-files
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
  - {{.Name}} test focused src/foo.ts
when_to_use:
  - Need quick validation after a focused code edit.
  - Need to infer relevant tests from changed files.
when_not_to_use:
  - Need full CI-equivalent release validation.
`

const ownersMetadataTemplate = `id: git.owners
name: Code owner lookup
summary: Finds likely owners and escalation contacts for a file path.
tags:
  - git
  - ownership
  - support
risk: Reads repository metadata only.
risk_level: low
read_only: true
mutates_files: false
mutates_external_systems: false
supports_dry_run: false
requires_confirmation: false
command:
  path:
    - git
    - owners
inputs:
  - name: path
    required: true
    description: File path to inspect.
examples:
  - {{.Name}} git owners src/foo.ts
when_to_use:
  - Need to find who likely owns a file.
when_not_to_use:
  - Need to modify CODEOWNERS.
`

const pricingMetadataTemplate = `id: product.pricing.import.validate
name: Product pricing import validator
summary: Validates a product pricing import file before release.
tags:
  - product
  - pricing
  - import
  - validation
risk: Reads an import file and may write validation artifacts.
risk_level: medium
read_only: false
mutates_files: true
mutates_external_systems: false
supports_dry_run: true
requires_confirmation: false
command:
  path:
    - product
    - pricing
    - import
    - validate
inputs:
  - name: file
    required: true
    description: Pricing import file to validate.
examples:
  - {{.Name}} product pricing import validate prices.csv --dry-run
when_to_use:
  - Need to validate pricing import data before release.
when_not_to_use:
  - Need to publish pricing changes to production.
`

const commandStubTemplate = `package commands

import (
	"context"
	"fmt"

	"github.com/blackb0x3/jussive/pkg/jussive"
)

func {{.Func}}() jussive.Command {
	return jussive.Command{
		ID:      "{{.ID}}",
		Path:    jussive.Path("{{.Path}}"),
		Summary: "TODO: add a concise summary.",
		Run: func(ctx context.Context, args jussive.Args) error {
			_ = ctx
			_ = args
			fmt.Println("{{.ID}}")
			return nil
		},
	}
}
`

const commandMetadataStubTemplate = `id: {{.ID}}
name: {{.Name}}
summary: Add a concise summary.
tags: []
risk: Describe operational risk.
risk_level: low
read_only: true
mutates_files: false
mutates_external_systems: false
supports_dry_run: false
requires_confirmation: false
command:
  path:
{{.PathYAML}}
inputs: []
examples: []
when_to_use: []
when_not_to_use: []
`

const commandsDocTemplate = `# Commands

This documentation is generated from the starter Jussive template.

## test focused

Finds and runs the smallest relevant test set for changed files.

## git owners

Finds likely owners and escalation contacts for a file path.

## product pricing import validate

Validates a product pricing import file before release.
`

const goldenTestTemplate = `package tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestAgentSearchYAML(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/{{.Name}}", "agent", "search", "focused tests")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent search failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{"ok: true", "query: focused tests", "id: test.focused"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in output:\n%s", want, text)
		}
	}
}

func TestVersionJSONEnvelope(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/{{.Name}}", "version", "--json")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version --json failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{` + "`\"ok\": true`" + `, ` + "`\"name\": \"{{.Name}}\"`" + `} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in output:\n%s", want, text)
		}
	}
}
`

const jussiveSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://jussive.dev/schemas/jussive.schema.json",
  "type": "object",
  "required": ["name", "release"],
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "changelog": {"type": "string"},
    "runtime": {
      "type": "object",
      "properties": {
        "kind": {"const": "go"},
        "entrypoint": {"type": "string", "minLength": 1}
      }
    },
    "release": {
      "type": "object",
      "required": ["version_source", "tag_prefix"],
      "properties": {
        "version_source": {"const": "git_tag"},
        "tag_prefix": {"type": "string"}
      }
    }
  }
}
`

const commandAgentSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://jussive.dev/schemas/command.agent.schema.json",
  "type": "object",
  "required": ["id", "name", "summary", "command"],
  "properties": {
    "id": {"type": "string", "minLength": 1},
    "name": {"type": "string", "minLength": 1},
    "summary": {"type": "string", "minLength": 1},
    "tags": {"type": "array", "items": {"type": "string"}},
    "risk_level": {"enum": ["low", "medium", "high"]},
    "command": {
      "type": "object",
      "required": ["path"],
      "properties": {
        "path": {"type": "array", "minItems": 1, "items": {"type": "string", "minLength": 1}}
      }
    }
  }
}
`

const envelopeSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://jussive.dev/schemas/envelope.schema.json",
  "type": "object",
  "required": ["ok", "data", "warnings", "errors"],
  "properties": {
    "ok": {"type": "boolean"},
    "data": {},
    "warnings": {"type": "array"},
    "errors": {"type": "array"}
  }
}
`
