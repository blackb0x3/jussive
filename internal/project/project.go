package project

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blackb0x3/jussive/pkg/jussive"
)

type Result struct {
	Project  jussive.ProjectConfig     `json:"project" yaml:"project"`
	Metadata []jussive.CommandMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Workspace struct {
	Root     string
	Config   jussive.ProjectConfig
	Metadata []jussive.CommandMetadata
}

func FindRoot(start string) (string, error) {
	if start == "" {
		start = "."
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "jussive.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("jussive.yaml was not found from %s", abs)
		}
	}
}

func OpenWorkspace(start string) (Workspace, error) {
	root, err := FindRoot(start)
	if err != nil {
		return Workspace{}, err
	}
	loaded, err := Load(root)
	if err != nil {
		return Workspace{}, err
	}
	return Workspace{Root: root, Config: loaded.Project, Metadata: loaded.Metadata}, nil
}

func Load(root string) (Result, error) {
	cfg, err := jussive.LoadProjectConfig(filepath.Join(root, "jussive.yaml"))
	if err != nil {
		return Result{}, err
	}
	metadata, err := jussive.LoadMetadata(root)
	if err != nil {
		return Result{}, err
	}
	return Result{Project: cfg, Metadata: metadata}, nil
}

func Validate(root string) (jussive.ValidationResult, error) {
	loaded, err := Load(root)
	if err != nil {
		return jussive.ValidationResult{}, err
	}
	result := jussive.ValidateMetadata(loaded.Metadata, nil)
	if loaded.Project.Name == "" {
		result.Errors = append(result.Errors, jussive.Diagnostic{Code: "project.missing_name", Message: "jussive.yaml is missing name", File: filepath.Join(root, "jussive.yaml")})
	}
	if loaded.Project.Release.VersionSource != "git_tag" {
		result.Errors = append(result.Errors, jussive.Diagnostic{Code: "project.invalid_version_source", Message: "release.version_source must be git_tag", File: filepath.Join(root, "jussive.yaml")})
	}
	if loaded.Project.Runtime.Kind != "" && loaded.Project.Runtime.Kind != "go" {
		result.Errors = append(result.Errors, jussive.Diagnostic{Code: "project.unsupported_runtime", Message: "runtime.kind must be go", File: filepath.Join(root, "jussive.yaml")})
	}
	if _, err := os.Stat(resolvePath(root, RuntimeEntrypoint(loaded.Project))); err != nil {
		result.Errors = append(result.Errors, jussive.Diagnostic{Code: "project.missing_runtime_entrypoint", Message: RuntimeEntrypoint(loaded.Project) + " is missing"})
	}
	for _, rel := range []string{
		"CHANGELOG.md",
		"AGENTS.md",
		filepath.Join("internal", "schemas", "jussive.schema.json"),
		filepath.Join("internal", "schemas", "command.agent.schema.json"),
		filepath.Join("internal", "schemas", "envelope.schema.json"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			result.Errors = append(result.Errors, jussive.Diagnostic{Code: "project.missing_file", Message: rel + " is missing"})
		}
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "build", "build.go")); err != nil {
		result.Errors = append(result.Errors, jussive.Diagnostic{Code: "project.missing_build_metadata", Message: "internal/build/build.go is missing"})
	}
	result.Errors = append(result.Errors, checkRuntimeSchemas(root, loaded.Project, loaded.Metadata)...)
	return result, nil
}

func RuntimeEntrypoint(cfg jussive.ProjectConfig) string {
	if cfg.Runtime.Entrypoint != "" {
		return cfg.Runtime.Entrypoint
	}
	if cfg.Name == "" {
		return ""
	}
	return "./cmd/" + cfg.Name
}

func (w Workspace) RuntimeEntrypoint() string {
	return RuntimeEntrypoint(w.Config)
}

func (w Workspace) FindCommandByID(id string) (jussive.CommandMetadata, bool) {
	for _, m := range w.Metadata {
		if m.ID == id {
			return m, true
		}
	}
	return jussive.CommandMetadata{}, false
}

func (w Workspace) FindCommandByPath(path []string) (jussive.CommandMetadata, bool) {
	key := strings.Join(path, "\x00")
	for _, m := range w.Metadata {
		if strings.Join(m.Command.Path, "\x00") == key {
			return m, true
		}
	}
	return jussive.CommandMetadata{}, false
}

func (w Workspace) HasSubcommands(path []string) bool {
	prefix := strings.Join(path, "\x00")
	for _, m := range w.Metadata {
		candidate := strings.Join(m.Command.Path, "\x00")
		if len(m.Command.Path) > len(path) && strings.HasPrefix(candidate, prefix+"\x00") {
			return true
		}
	}
	return false
}

func (w Workspace) GoRun(stdout, stderr io.Writer, argv ...string) int {
	args := append([]string{"run", w.RuntimeEntrypoint()}, argv...)
	cmd := exec.Command("go", args...)
	cmd.Dir = w.Root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return jussive.ExitCommandFailed
	}
	return jussive.ExitSuccess
}

func (w Workspace) Build(output string, stdout, stderr io.Writer) int {
	if output == "" {
		output = filepath.Join("bin", w.Config.Name)
	}
	if err := os.MkdirAll(resolvePath(w.Root, filepath.Dir(output)), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return jussive.ExitCommandFailed
	}
	cmd := exec.Command("go", "build", "-o", output, w.RuntimeEntrypoint())
	cmd.Dir = w.Root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return jussive.ExitCommandFailed
	}
	fmt.Fprintf(stdout, "built %s\n", filepath.Join(w.Root, output))
	return jussive.ExitSuccess
}

func checkRuntimeSchemas(root string, cfg jussive.ProjectConfig, metadata []jussive.CommandMetadata) []jussive.Diagnostic {
	if cfg.Name == "" {
		return nil
	}
	entrypoint := RuntimeEntrypoint(cfg)
	if entrypoint == "" {
		return nil
	}
	if _, err := os.Stat(resolvePath(root, entrypoint)); err != nil {
		return nil
	}
	var errors []jussive.Diagnostic
	for _, m := range metadata {
		cmd := exec.Command("go", "run", entrypoint, "agent", "schema", m.ID, "--json")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			errors = append(errors, jussive.Diagnostic{Code: "runtime.schema_missing", Message: fmt.Sprintf("%s does not expose agent schema: %s", m.ID, strings.TrimSpace(string(out)))})
			continue
		}
		var env struct {
			OK       bool              `json:"ok"`
			Data     any               `json:"data"`
			Warnings []json.RawMessage `json:"warnings"`
			Errors   []json.RawMessage `json:"errors"`
		}
		if err := json.Unmarshal(out, &env); err != nil {
			errors = append(errors, jussive.Diagnostic{Code: "runtime.schema_malformed_json", Message: fmt.Sprintf("%s agent schema did not return JSON", m.ID)})
			continue
		}
		if !env.OK || env.Warnings == nil || env.Errors == nil {
			errors = append(errors, jussive.Diagnostic{Code: "runtime.schema_bad_envelope", Message: fmt.Sprintf("%s agent schema did not return a valid envelope", m.ID)})
		}
	}
	return errors
}

func resolvePath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func VersionCheck(root string) ([]jussive.Diagnostic, []jussive.Diagnostic) {
	var warnings []jussive.Diagnostic
	var errors []jussive.Diagnostic
	cmd := exec.Command("git", "tag", "--list")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		warnings = append(warnings, jussive.Diagnostic{Code: "version.git_unavailable", Message: "git tags could not be inspected"})
		return warnings, errors
	}
	re := regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	for _, tag := range strings.Fields(string(out)) {
		if strings.HasPrefix(tag, "v") && !re.MatchString(tag) {
			errors = append(errors, jussive.Diagnostic{Code: "version.invalid_tag", Message: fmt.Sprintf("tag %q is not valid SemVer with v prefix", tag)})
		}
	}
	return warnings, errors
}

func VersionPlan(root string) (map[string]any, []jussive.Diagnostic, []jussive.Diagnostic) {
	result, err := Validate(root)
	if err != nil {
		return map[string]any{"recommended_bump": "unknown"}, nil, []jussive.Diagnostic{{Code: "project.load_failed", Message: err.Error()}}
	}
	bump := "patch"
	reasons := []string{"metadata or documentation changes should release at least a patch when published"}
	for _, diag := range result.Errors {
		if strings.Contains(diag.Code, "missing_metadata") || strings.Contains(diag.Code, "duplicate") || strings.Contains(diag.Code, "path_mismatch") {
			bump = "major"
			reasons = append(reasons, diag.Message)
		}
	}
	for _, m := range result.Warnings {
		if m.Code == "metadata.missing_examples" && bump != "major" {
			bump = "minor"
			reasons = append(reasons, "new or incomplete command contracts should be reviewed before release")
			break
		}
	}
	return map[string]any{"recommended_bump": bump, "reasons": reasons}, result.Warnings, result.Errors
}

func GenerateDocs(root string) error {
	loaded, err := Load(root)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Commands\n\n")
	for _, m := range loaded.Metadata {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", strings.Join(m.Command.Path, " "), m.Summary)
		if len(m.Examples) > 0 {
			b.WriteString("Examples:\n\n")
			for _, example := range m.Examples {
				fmt.Fprintf(&b, "- `%s`\n", example)
			}
			b.WriteString("\n")
		}
	}
	path := filepath.Join(root, "docs", "commands.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
