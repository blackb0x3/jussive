package jussive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type App struct {
	Name          string
	Version       BuildInfo
	MetadataRoot  string
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	commands      []Command
	commandsByID  map[string]Command
	commandsByKey map[string]Command
}

func New(name string) *App {
	return &App{
		Name:          name,
		Version:       BuildInfo{Name: name, Version: "dev", Commit: "unknown", BuiltAt: "unknown"},
		MetadataRoot:  ".",
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		commandsByID:  map[string]Command{},
		commandsByKey: map[string]Command{},
	}
}

func (a *App) Command(c Command) {
	a.commands = append(a.commands, c)
	a.commandsByID[c.ID] = c
	a.commandsByKey[pathKey(c.Path)] = c
}

func (a *App) Commands() []Command {
	return append([]Command(nil), a.commands...)
}

func (a *App) Run(ctx context.Context, argv []string) int {
	if len(argv) == 0 {
		a.printHelp()
		return ExitSuccess
	}
	if argv[0] == "--version" || argv[0] == "-v" {
		fmt.Fprintln(a.Stdout, a.Version.Version)
		return ExitSuccess
	}
	switch argv[0] {
	case "version":
		return a.runVersion(argv[1:])
	case "agent":
		return a.runAgent(ctx, argv[1:])
	case "help", "--help", "-h":
		a.printHelp()
		return ExitSuccess
	default:
		return a.runDirect(ctx, argv)
	}
}

func (a *App) runVersion(argv []string) int {
	rest, format, err := parseOutputFlags(argv)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitInvalidUsage
	}
	if len(rest) > 0 {
		fmt.Fprintf(a.Stderr, "unexpected version arguments: %s\n", strings.Join(rest, " "))
		return ExitInvalidUsage
	}
	if format == outputJSON {
		if err := WriteEnvelope(a.Stdout, Envelope{OK: true, Data: a.Version, Warnings: []Diagnostic{}, Errors: []Diagnostic{}}, outputJSON); err != nil {
			fmt.Fprintln(a.Stderr, err)
			return ExitCommandFailed
		}
		return ExitSuccess
	}
	fmt.Fprintf(a.Stdout, "name: %s\nversion: %s\ncommit: %s\ndirty: %t\nbuilt_at: %s\n", a.Version.Name, a.Version.Version, a.Version.Commit, a.Version.Dirty, a.Version.BuiltAt)
	return ExitSuccess
}

func (a *App) runAgent(ctx context.Context, argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintln(a.Stderr, "agent requires a subcommand")
		return ExitInvalidUsage
	}
	switch argv[0] {
	case "search":
		return a.agentSearch(argv[1:])
	case "info":
		return a.agentInfo(argv[1:])
	case "schema":
		return a.agentSchema(argv[1:])
	case "doctor":
		return a.agentDoctor(argv[1:])
	case "run":
		return a.agentRun(ctx, argv[1:])
	default:
		fmt.Fprintf(a.Stderr, "unknown agent subcommand %q\n", argv[0])
		return ExitInvalidUsage
	}
}

func (a *App) agentSearch(argv []string) int {
	rest, format, err := parseOutputFlags(argv)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitInvalidUsage
	}
	limit := 5
	queryParts := []string{}
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--limit":
			if i+1 >= len(rest) {
				fmt.Fprintln(a.Stderr, "--limit requires a value")
				return ExitInvalidUsage
			}
			n, err := strconv.Atoi(rest[i+1])
			if err != nil || n <= 0 {
				fmt.Fprintln(a.Stderr, "--limit must be a positive integer")
				return ExitInvalidUsage
			}
			limit = n
			i++
		default:
			queryParts = append(queryParts, rest[i])
		}
	}
	if len(queryParts) == 0 {
		fmt.Fprintln(a.Stderr, "agent search requires a query")
		return ExitInvalidUsage
	}
	metadata, err := LoadMetadata(a.MetadataRoot)
	if err != nil {
		return a.writeAgentError(format, ExitCommandFailed, "metadata.load_failed", err.Error())
	}
	data := map[string]any{
		"query":   strings.Join(queryParts, " "),
		"results": SearchMetadata(metadata, strings.Join(queryParts, " "), limit),
	}
	return a.writeAgent(format, data, nil)
}

func (a *App) agentInfo(argv []string) int {
	rest, format, err := parseOutputFlags(argv)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitInvalidUsage
	}
	if len(rest) != 1 {
		fmt.Fprintln(a.Stderr, "agent info requires exactly one command id")
		return ExitInvalidUsage
	}
	metadata, err := LoadMetadata(a.MetadataRoot)
	if err != nil {
		return a.writeAgentError(format, ExitCommandFailed, "metadata.load_failed", err.Error())
	}
	for _, m := range metadata {
		if m.ID == rest[0] {
			c, ok := a.commandsByID[m.ID]
			data := map[string]any{
				"metadata":        m,
				"is_runnable":     ok && c.Run != nil,
				"has_subcommands": a.hasSubcommands(m.Command.Path),
			}
			return a.writeAgent(format, data, nil)
		}
	}
	return a.writeAgentError(format, ExitNotFound, "command.not_found", fmt.Sprintf("command id %q was not found", rest[0]))
}

func (a *App) agentSchema(argv []string) int {
	rest, format, err := parseOutputFlags(argv)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitInvalidUsage
	}
	if len(rest) != 1 {
		fmt.Fprintln(a.Stderr, "agent schema requires exactly one command id")
		return ExitInvalidUsage
	}
	c, ok := a.commandsByID[rest[0]]
	if !ok {
		return a.writeAgentError(format, ExitNotFound, "command.not_found", fmt.Sprintf("command id %q was not found", rest[0]))
	}
	data := map[string]any{
		"id":                       c.ID,
		"command":                  map[string]any{"path": c.Path},
		"parameters":               c.Parameters,
		"accepts_passthrough_args": len(c.Parameters) == 0,
	}
	return a.writeAgent(format, data, nil)
}

func (a *App) agentDoctor(argv []string) int {
	rest, format, err := parseOutputFlags(argv)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitInvalidUsage
	}
	if len(rest) != 0 {
		fmt.Fprintln(a.Stderr, "agent doctor does not accept positional arguments")
		return ExitInvalidUsage
	}
	metadata, err := LoadMetadata(a.MetadataRoot)
	if err != nil {
		return a.writeAgentError(format, ExitCommandFailed, "metadata.load_failed", err.Error())
	}
	result := ValidateMetadata(metadata, a.commands)
	data := map[string]any{
		"commands": result.Commands,
		"status":   "ok",
	}
	code := ExitSuccess
	if len(result.Errors) > 0 {
		data["status"] = "failed"
		code = ExitValidationFailed
	}
	if err := WriteEnvelope(a.Stdout, Envelope{OK: code == ExitSuccess, Data: data, Warnings: result.Warnings, Errors: result.Errors}, format); err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitCommandFailed
	}
	return code
}

func (a *App) agentRun(ctx context.Context, argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintln(a.Stderr, "agent run requires a command id")
		return ExitInvalidUsage
	}
	id := argv[0]
	rest := argv[1:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	c, ok := a.commandsByID[id]
	if !ok {
		fmt.Fprintf(a.Stderr, "command id %q was not found\n", id)
		return ExitNotFound
	}
	return a.invoke(ctx, c, rest)
}

func (a *App) runDirect(ctx context.Context, argv []string) int {
	for i := len(argv); i >= 1; i-- {
		if c, ok := a.commandsByKey[pathKey(argv[:i])]; ok {
			if c.Run == nil {
				break
			}
			return a.invoke(ctx, c, argv[i:])
		}
	}
	fmt.Fprintf(a.Stderr, "unknown command path %q\n", strings.Join(argv, " "))
	return ExitNotFound
}

func (a *App) invoke(ctx context.Context, c Command, raw []string) int {
	args, err := parseArgs(c, raw)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitInvalidUsage
	}
	if err := c.Run(ctx, args); err != nil {
		if errors.Is(err, ErrInvalidUsage) {
			fmt.Fprintln(a.Stderr, err)
			return ExitInvalidUsage
		}
		fmt.Fprintln(a.Stderr, err)
		return ExitCommandFailed
	}
	return ExitSuccess
}

func parseArgs(c Command, raw []string) (Args, error) {
	args := Args{Flags: map[string]string{}, Bools: map[string]bool{}, Raw: append([]string(nil), raw...)}
	byFlag := map[string]Parameter{}
	for _, p := range c.Parameters {
		if p.Flag != "" {
			byFlag[p.Flag] = p
		}
	}
	for i := 0; i < len(raw); i++ {
		part := raw[i]
		if strings.HasPrefix(part, "--") {
			p, ok := byFlag[part]
			if !ok {
				return args, UsageError("unknown flag %s", part)
			}
			if p.Type == "boolean" {
				args.Bools[p.Name] = true
				continue
			}
			if i+1 >= len(raw) {
				return args, UsageError("%s requires a value", part)
			}
			args.Flags[p.Name] = raw[i+1]
			i++
			continue
		}
		args.Positionals = append(args.Positionals, part)
	}
	for _, p := range c.Parameters {
		if p.Position != nil && *p.Position < len(args.Positionals) {
			args.Flags[p.Name] = args.Positionals[*p.Position]
		}
		if p.Required {
			_, hasFlag := args.Flags[p.Name]
			_, hasBool := args.Bools[p.Name]
			if !hasFlag && !hasBool {
				return args, UsageError("missing required parameter %s", p.Name)
			}
		}
	}
	return args, nil
}

func (a *App) writeAgent(format outputFormat, data any, warnings []Diagnostic) int {
	if warnings == nil {
		warnings = []Diagnostic{}
	}
	if err := WriteEnvelope(a.Stdout, Envelope{OK: true, Data: data, Warnings: warnings, Errors: []Diagnostic{}}, format); err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitCommandFailed
	}
	return ExitSuccess
}

func (a *App) writeAgentError(format outputFormat, code int, diagCode, message string) int {
	env := Envelope{OK: false, Data: map[string]any{}, Warnings: []Diagnostic{}, Errors: []Diagnostic{{Code: diagCode, Message: message}}}
	if err := WriteEnvelope(a.Stdout, env, format); err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitCommandFailed
	}
	return code
}

func (a *App) hasSubcommands(path []string) bool {
	prefix := pathKey(path)
	for _, c := range a.commands {
		if len(c.Path) > len(path) && strings.HasPrefix(pathKey(c.Path), prefix+"\x00") {
			return true
		}
	}
	return false
}

func (a *App) printHelp() {
	fmt.Fprintf(a.Stdout, "%s commands:\n", a.Name)
	var paths []string
	for _, c := range a.commands {
		paths = append(paths, "  "+strings.Join(c.Path, " "))
	}
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Fprintln(a.Stdout, path)
	}
	fmt.Fprintln(a.Stdout, "  agent search|info|schema|run|doctor")
	fmt.Fprintln(a.Stdout, "  version")
}

func pathKey(path []string) string {
	return strings.Join(path, "\x00")
}
