package jussive

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteEnvelopeNormalizesEmptyDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	err := WriteEnvelopeFormat(&buf, Envelope{OK: true, Data: map[string]any{"status": "ok"}}, "json")
	if err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	if strings.Contains(text, "null") {
		t.Fatalf("expected empty arrays, got:\n%s", text)
	}
	for _, want := range []string{`"warnings": []`, `"errors": []`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}
