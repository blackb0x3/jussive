package scaffold

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectGeneratesWorkingCLI(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "my-tools")
	if err := NewProject(target, Options{Name: "my-tools", FrameworkDir: root}); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = target
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}
	run("go", "test", "./...")
	search := run("go", "run", filepath.Join(root, "cmd/jussive"), "agent", "search", "focused tests")
	if !strings.Contains(search, "id: test.focused") {
		t.Fatalf("search did not return test.focused:\n%s", search)
	}
	info := run("go", "run", filepath.Join(root, "cmd/jussive"), "agent", "info", "test.focused")
	if !strings.Contains(info, "is_runnable: true") {
		t.Fatalf("info did not report runnable command:\n%s", info)
	}
	schema := run("go", "run", filepath.Join(root, "cmd/jussive"), "agent", "schema", "test.focused", "--json")
	for _, want := range []string{`"ok": true`, `"parameters": [`} {
		if !strings.Contains(schema, want) {
			t.Fatalf("schema missing %q:\n%s", want, schema)
		}
	}
	agentRun := run("go", "run", filepath.Join(root, "cmd/jussive"), "agent", "run", "test.focused", "--", "src/foo.ts")
	if !strings.Contains(agentRun, "focused tests for src/foo.ts") {
		t.Fatalf("agent run failed:\n%s", agentRun)
	}
	pathRun := run("go", "run", filepath.Join(root, "cmd/jussive"), "run", "test", "focused", "--", "src/foo.ts")
	if !strings.Contains(pathRun, "focused tests for src/foo.ts") {
		t.Fatalf("jussive run failed:\n%s", pathRun)
	}
	run("go", "run", filepath.Join(root, "cmd/jussive"), "build")
	exported := run(filepath.Join(target, "bin", "my-tools"), "agent", "search", "focused tests")
	if !strings.Contains(exported, "id: test.focused") {
		t.Fatalf("exported cli search failed:\n%s", exported)
	}
	if err := AddCommand(target, "ops.azure.pipelines.approve", "ops azure pipelines approve"); err != nil {
		t.Fatal(err)
	}
	run("go", "test", "./...")
	addedSchema := run("go", "run", filepath.Join(root, "cmd/jussive"), "agent", "schema", "ops.azure.pipelines.approve")
	if !strings.Contains(addedSchema, "id: ops.azure.pipelines.approve") {
		t.Fatalf("added command schema missing id:\n%s", addedSchema)
	}
}
