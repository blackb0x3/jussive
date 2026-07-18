package jussive

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestArbitraryDepthDirectDispatch(t *testing.T) {
	var called bool
	app := New("tools")
	app.Stdout = &bytes.Buffer{}
	app.Stderr = &bytes.Buffer{}
	app.Command(Command{
		ID:   "product.pricing.import.validate",
		Path: Path("product pricing import validate"),
		Parameters: []Parameter{
			PathParam("file").Position(0).Required().Build(),
		},
		Run: func(ctx context.Context, args Args) error {
			called = true
			if got := args.String("file"); got != "prices.csv" {
				t.Fatalf("file = %q", got)
			}
			return nil
		},
	})
	if code := app.Run(context.Background(), []string{"product", "pricing", "import", "validate", "prices.csv"}); code != ExitSuccess {
		t.Fatalf("exit code = %d", code)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestAgentSchemaJSONEnvelope(t *testing.T) {
	stdout := &bytes.Buffer{}
	app := New("tools")
	app.Stdout = stdout
	app.Stderr = &bytes.Buffer{}
	app.Command(Command{
		ID:   "test.focused",
		Path: Path("test focused"),
		Parameters: []Parameter{
			PathParam("path").Position(0).Required().Build(),
		},
		Run: func(context.Context, Args) error { return nil },
	})
	if code := app.Run(context.Background(), []string{"agent", "schema", "test.focused", "--json"}); code != ExitSuccess {
		t.Fatalf("exit code = %d", code)
	}
	text := stdout.String()
	for _, want := range []string{`"ok": true`, `"id": "test.focused"`, `"warnings": []`, `"errors": []`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}
