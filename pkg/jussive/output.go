package jussive

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type outputFormat string

const (
	outputYAML outputFormat = "yaml"
	outputJSON outputFormat = "json"
)

func parseOutputFlags(args []string) ([]string, outputFormat, error) {
	format := outputYAML
	kept := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			format = outputJSON
		case "--output":
			if i+1 >= len(args) {
				return nil, "", UsageError("--output requires a value")
			}
			i++
			switch args[i] {
			case "json":
				format = outputJSON
			case "yaml":
				format = outputYAML
			default:
				return nil, "", UsageError("unsupported output format %q", args[i])
			}
		default:
			kept = append(kept, args[i])
		}
	}
	return kept, format, nil
}

func WriteEnvelope(w io.Writer, env Envelope, format outputFormat) error {
	if env.Warnings == nil {
		env.Warnings = []Diagnostic{}
	}
	if env.Errors == nil {
		env.Errors = []Diagnostic{}
	}
	switch format {
	case outputJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(env)
	case outputYAML:
		b, err := yaml.Marshal(env)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(b))
		return err
	default:
		return UsageError("unsupported output format %q", format)
	}
}

func WriteEnvelopeFormat(w io.Writer, env Envelope, format string) error {
	switch format {
	case "json":
		return WriteEnvelope(w, env, outputJSON)
	case "yaml", "":
		return WriteEnvelope(w, env, outputYAML)
	default:
		return UsageError("unsupported output format %q", format)
	}
}
