package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackb0x3/jussive/internal/project"
	"github.com/blackb0x3/jussive/internal/scaffold"
	"github.com/blackb0x3/jussive/pkg/jussive"
)

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
	dirty   = "false"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	if len(argv) == 0 || argv[0] == "--help" || argv[0] == "-h" {
		printHelp()
		return jussive.ExitSuccess
	}
	if argv[0] == "--version" {
		fmt.Println(version)
		return jussive.ExitSuccess
	}
	switch argv[0] {
	case "agent":
		return runWorkspaceAgent(argv[1:])
	case "run":
		return runWorkspaceCommand(argv[1:])
	case "new":
		return runNew(argv[1:])
	case "add":
		return runAdd(argv[1:])
	case "validate":
		return runValidate(argv[1:])
	case "doctor":
		return runDoctor(argv[1:])
	case "docs":
		return runDocs(argv[1:])
	case "agents":
		return runAgents(argv[1:])
	case "version":
		return runVersion(argv[1:])
	case "list":
		return runList(argv[1:])
	case "build":
		return runBuild(argv[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", argv[0])
		return jussive.ExitInvalidUsage
	}
}

func runNew(argv []string) int {
	if len(argv) != 1 {
		fmt.Fprintln(os.Stderr, "usage: jussive new <path>")
		return jussive.ExitInvalidUsage
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	root, err := frameworkRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	target := argv[0]
	if !filepath.IsAbs(target) {
		target = filepath.Join(cwd, target)
	}
	name := filepath.Base(filepath.Clean(target))
	if err := scaffold.NewProject(target, scaffold.Options{Name: name, FrameworkDir: root}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	fmt.Fprintf(os.Stdout, "created %s\n", target)
	return jussive.ExitSuccess
}

func runAdd(argv []string) int {
	if len(argv) < 1 {
		fmt.Fprintln(os.Stderr, "usage: jussive add command <id> --path \"segments\"")
		return jussive.ExitInvalidUsage
	}
	if argv[0] != "command" {
		fmt.Fprintf(os.Stderr, "unsupported add target %q\n", argv[0])
		return jussive.ExitInvalidUsage
	}
	if len(argv) < 4 {
		fmt.Fprintln(os.Stderr, "usage: jussive add command <id> --path \"segments\"")
		return jussive.ExitInvalidUsage
	}
	id := argv[1]
	path := ""
	for i := 2; i < len(argv); i++ {
		if argv[i] == "--path" && i+1 < len(argv) {
			path = argv[i+1]
			i++
		}
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "--path is required")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if err := scaffold.AddCommand(workspace.Root, id, path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	fmt.Fprintf(os.Stdout, "added %s\n", id)
	return jussive.ExitSuccess
}

func runValidate(argv []string) int {
	rest, format, err := outputFlags(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if len(rest) != 0 {
		fmt.Fprintln(os.Stderr, "jussive validate does not accept positional arguments")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		return write(jussive.Envelope{OK: false, Data: map[string]any{}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{{Code: "project.load_failed", Message: err.Error()}}}, format, jussive.ExitValidationFailed)
	}
	result, err := project.Validate(workspace.Root)
	if err != nil {
		return write(jussive.Envelope{OK: false, Data: map[string]any{}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{{Code: "project.load_failed", Message: err.Error()}}}, format, jussive.ExitValidationFailed)
	}
	data := map[string]any{"commands": result.Commands, "status": "ok"}
	code := jussive.ExitSuccess
	if len(result.Errors) > 0 {
		data["status"] = "failed"
		code = jussive.ExitValidationFailed
	}
	return write(jussive.Envelope{OK: code == 0, Data: data, Warnings: result.Warnings, Errors: result.Errors}, format, code)
}

func runDoctor(argv []string) int {
	rest, format, err := outputFlags(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if len(rest) != 0 {
		fmt.Fprintln(os.Stderr, "jussive doctor does not accept positional arguments")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		return write(jussive.Envelope{OK: false, Data: map[string]any{}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{{Code: "project.load_failed", Message: err.Error()}}}, format, jussive.ExitValidationFailed)
	}
	result, err := project.Validate(workspace.Root)
	if err != nil {
		return write(jussive.Envelope{OK: false, Data: map[string]any{}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{{Code: "project.load_failed", Message: err.Error()}}}, format, jussive.ExitValidationFailed)
	}
	warnings, errors := project.VersionCheck(workspace.Root)
	result.Warnings = append(result.Warnings, warnings...)
	result.Errors = append(result.Errors, errors...)
	data := map[string]any{"commands": result.Commands, "status": "ok"}
	code := jussive.ExitSuccess
	if len(result.Errors) > 0 {
		data["status"] = "failed"
		code = jussive.ExitValidationFailed
	}
	return write(jussive.Envelope{OK: code == 0, Data: data, Warnings: result.Warnings, Errors: result.Errors}, format, code)
}

func runDocs(argv []string) int {
	if len(argv) != 0 {
		fmt.Fprintln(os.Stderr, "jussive docs does not accept positional arguments")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if err := project.GenerateDocs(workspace.Root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	fmt.Fprintln(os.Stdout, "wrote docs/commands.md")
	return jussive.ExitSuccess
}

func runAgents(argv []string) int {
	if len(argv) != 1 || argv[0] != "snippet" {
		fmt.Fprintln(os.Stderr, "usage: jussive agents snippet")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	fmt.Print(scaffold.AgentSnippet(workspace.Config.Name))
	return jussive.ExitSuccess
}

func runVersion(argv []string) int {
	if len(argv) == 0 {
		fmt.Printf("jussive %s\ncommit: %s\nbuilt_at: %s\ndirty: %s\n", version, commit, builtAt, dirty)
		return jussive.ExitSuccess
	}
	switch argv[0] {
	case "check":
		workspace, err := project.OpenWorkspace(".")
		if err != nil {
			return writeAgentError("yaml", jussive.ExitCommandFailed, "project.load_failed", err.Error())
		}
		warnings, errors := project.VersionCheck(workspace.Root)
		return write(jussive.Envelope{OK: len(errors) == 0, Data: map[string]any{"status": status(errors)}, Warnings: warnings, Errors: errors}, "yaml", exitForErrors(errors))
	case "plan":
		workspace, err := project.OpenWorkspace(".")
		if err != nil {
			return writeAgentError("yaml", jussive.ExitCommandFailed, "project.load_failed", err.Error())
		}
		data, warnings, errors := project.VersionPlan(workspace.Root)
		return write(jussive.Envelope{OK: len(errors) == 0, Data: data, Warnings: warnings, Errors: errors}, "yaml", exitForErrors(errors))
	default:
		fmt.Fprintf(os.Stderr, "unknown version subcommand %q\n", argv[0])
		return jussive.ExitInvalidUsage
	}
}

func runList(argv []string) int {
	rest, format, err := outputFlags(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if len(rest) != 0 {
		fmt.Fprintln(os.Stderr, "jussive list does not accept positional arguments")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		return write(jussive.Envelope{OK: false, Data: map[string]any{}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{{Code: "project.load_failed", Message: err.Error()}}}, format, jussive.ExitCommandFailed)
	}
	metadata := workspace.Metadata
	ids := make([]string, 0, len(metadata))
	for _, m := range metadata {
		ids = append(ids, m.ID)
	}
	return write(jussive.Envelope{OK: true, Data: map[string]any{"commands": ids}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{}}, format, jussive.ExitSuccess)
}

func runBuild(argv []string) int {
	output := ""
	for i := 0; i < len(argv); i++ {
		switch argv[i] {
		case "-o", "--output":
			if i+1 >= len(argv) {
				fmt.Fprintln(os.Stderr, argv[i]+" requires a value")
				return jussive.ExitInvalidUsage
			}
			output = argv[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "unexpected build argument %q\n", argv[i])
			return jussive.ExitInvalidUsage
		}
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	return workspace.Build(output, os.Stdout, os.Stderr)
}

func runWorkspaceAgent(argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "agent requires a subcommand")
		return jussive.ExitInvalidUsage
	}
	switch argv[0] {
	case "search":
		return runWorkspaceAgentSearch(argv[1:])
	case "info":
		return runWorkspaceAgentInfo(argv[1:])
	case "schema":
		return runWorkspaceAgentSchema(argv[1:])
	case "run":
		return runWorkspaceAgentRun(argv[1:])
	case "doctor":
		return runWorkspaceAgentDoctor(argv[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown agent subcommand %q\n", argv[0])
		return jussive.ExitInvalidUsage
	}
}

func runWorkspaceAgentSearch(argv []string) int {
	rest, format, err := outputFlags(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	limit := 5
	queryParts := []string{}
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--limit":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "--limit requires a value")
				return jussive.ExitInvalidUsage
			}
			var n int
			if _, err := fmt.Sscanf(rest[i+1], "%d", &n); err != nil || n <= 0 {
				fmt.Fprintln(os.Stderr, "--limit must be a positive integer")
				return jussive.ExitInvalidUsage
			}
			limit = n
			i++
		default:
			queryParts = append(queryParts, rest[i])
		}
	}
	if len(queryParts) == 0 {
		fmt.Fprintln(os.Stderr, "agent search requires a query")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		return writeAgentError(format, jussive.ExitCommandFailed, "project.load_failed", err.Error())
	}
	query := strings.Join(queryParts, " ")
	data := map[string]any{
		"query":   query,
		"results": jussive.SearchMetadata(workspace.Metadata, query, limit),
	}
	return write(jussive.Envelope{OK: true, Data: data, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{}}, format, jussive.ExitSuccess)
}

func runWorkspaceAgentInfo(argv []string) int {
	rest, format, err := outputFlags(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "agent info requires exactly one command id")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		return writeAgentError(format, jussive.ExitCommandFailed, "project.load_failed", err.Error())
	}
	metadata, ok := workspace.FindCommandByID(rest[0])
	if !ok {
		return writeAgentError(format, jussive.ExitNotFound, "command.not_found", fmt.Sprintf("command id %q was not found", rest[0]))
	}
	data := map[string]any{
		"metadata":        metadata,
		"is_runnable":     true,
		"has_subcommands": workspace.HasSubcommands(metadata.Command.Path),
	}
	return write(jussive.Envelope{OK: true, Data: data, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{}}, format, jussive.ExitSuccess)
}

func runWorkspaceAgentSchema(argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "agent schema requires a command id")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		format := formatFromArgs(argv)
		return writeAgentError(format, jussive.ExitCommandFailed, "project.load_failed", err.Error())
	}
	if _, ok := workspace.FindCommandByID(firstNonOutputArg(argv)); !ok {
		format := formatFromArgs(argv)
		return writeAgentError(format, jussive.ExitNotFound, "command.not_found", fmt.Sprintf("command id %q was not found", firstNonOutputArg(argv)))
	}
	return workspace.GoRun(os.Stdout, os.Stderr, append([]string{"agent", "schema"}, argv...)...)
}

func runWorkspaceAgentRun(argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "agent run requires a command id")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	if _, ok := workspace.FindCommandByID(argv[0]); !ok {
		fmt.Fprintf(os.Stderr, "command id %q was not found\n", argv[0])
		return jussive.ExitNotFound
	}
	return workspace.GoRun(os.Stdout, os.Stderr, append([]string{"agent", "run"}, argv...)...)
}

func runWorkspaceAgentDoctor(argv []string) int {
	rest, format, err := outputFlags(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitInvalidUsage
	}
	if len(rest) != 0 {
		fmt.Fprintln(os.Stderr, "agent doctor does not accept positional arguments")
		return jussive.ExitInvalidUsage
	}
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		return writeAgentError(format, jussive.ExitCommandFailed, "project.load_failed", err.Error())
	}
	result, err := project.Validate(workspace.Root)
	if err != nil {
		return writeAgentError(format, jussive.ExitCommandFailed, "project.validate_failed", err.Error())
	}
	data := map[string]any{"commands": result.Commands, "status": "ok"}
	code := jussive.ExitSuccess
	if len(result.Errors) > 0 {
		data["status"] = "failed"
		code = jussive.ExitValidationFailed
	}
	return write(jussive.Envelope{OK: code == 0, Data: data, Warnings: result.Warnings, Errors: result.Errors}, format, code)
}

func runWorkspaceCommand(argv []string) int {
	separator := -1
	for i, arg := range argv {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 1 {
		fmt.Fprintln(os.Stderr, "usage: jussive run <command path> -- [args...]")
		return jussive.ExitInvalidUsage
	}
	path := argv[:separator]
	args := argv[separator+1:]
	workspace, err := project.OpenWorkspace(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	if _, ok := workspace.FindCommandByPath(path); !ok {
		fmt.Fprintf(os.Stderr, "command path %q was not found\n", strings.Join(path, " "))
		return jussive.ExitNotFound
	}
	childArgs := append(append([]string{}, path...), args...)
	return workspace.GoRun(os.Stdout, os.Stderr, childArgs...)
}

func outputFlags(args []string) ([]string, string, error) {
	format := "yaml"
	kept := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			format = "json"
		case "--output":
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("--output requires a value")
			}
			i++
			if args[i] != "json" && args[i] != "yaml" {
				return nil, "", fmt.Errorf("unsupported output format %q", args[i])
			}
			format = args[i]
		default:
			kept = append(kept, args[i])
		}
	}
	return kept, format, nil
}

func formatFromArgs(args []string) string {
	_, format, err := outputFlags(args)
	if err != nil {
		return "yaml"
	}
	return format
}

func firstNonOutputArg(args []string) string {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			continue
		case "--output":
			i++
			continue
		default:
			return args[i]
		}
	}
	return ""
}

func writeAgentError(format string, code int, diagCode, message string) int {
	return write(jussive.Envelope{OK: false, Data: map[string]any{}, Warnings: []jussive.Diagnostic{}, Errors: []jussive.Diagnostic{{Code: diagCode, Message: message}}}, format, code)
}

func write(env jussive.Envelope, format string, code int) int {
	if err := jussive.WriteEnvelopeFormat(os.Stdout, env, format); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return jussive.ExitCommandFailed
	}
	return code
}

func status(errors []jussive.Diagnostic) string {
	if len(errors) == 0 {
		return "ok"
	}
	return "failed"
}

func exitForErrors(errors []jussive.Diagnostic) int {
	if len(errors) > 0 {
		return jussive.ExitValidationFailed
	}
	return jussive.ExitSuccess
}

func frameworkRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err == nil && strings.Contains(string(b), "module github.com/blackb0x3/jussive") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd, nil
		}
	}
}

func printHelp() {
	fmt.Println(`jussive commands:
  agent search|info|schema|run|doctor
  run <command path> -- [args...]
  new <path>
  add command <id> --path "segments"
  validate [--json|--output yaml|json]
  doctor [--json|--output yaml|json]
  docs
  agents snippet
  version check|plan
  build [-o path]
  list`)
}
