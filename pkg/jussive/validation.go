package jussive

import (
	"fmt"
	"strings"
)

var reservedTopLevel = map[string]bool{
	"agent":      true,
	"help":       true,
	"version":    true,
	"completion": true,
	"doctor":     true,
}

type ValidationResult struct {
	Commands int          `json:"commands" yaml:"commands"`
	Warnings []Diagnostic `json:"warnings" yaml:"warnings"`
	Errors   []Diagnostic `json:"errors" yaml:"errors"`
}

func ValidateMetadata(metadata []CommandMetadata, commands []Command) ValidationResult {
	result := ValidationResult{Commands: len(metadata)}
	ids := map[string]string{}
	paths := map[string]string{}
	registeredByID := map[string]Command{}
	for _, c := range commands {
		registeredByID[c.ID] = c
	}
	for _, m := range metadata {
		file := m.File
		if m.ID == "" {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.missing_id", Message: "metadata is missing id", File: file})
		} else if prev := ids[m.ID]; prev != "" {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.duplicate_id", Message: fmt.Sprintf("duplicate command id %q also appears in %s", m.ID, prev), File: file})
		} else {
			ids[m.ID] = file
		}
		if strings.TrimSpace(m.Name) == "" {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.missing_name", Message: "metadata is missing name", File: file})
		}
		if strings.TrimSpace(m.Summary) == "" {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.missing_summary", Message: "metadata is missing summary", File: file})
		}
		if len(m.Command.Path) == 0 {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.empty_path", Message: "command.path must contain at least one segment", File: file})
		}
		for _, segment := range m.Command.Path {
			if strings.TrimSpace(segment) == "" {
				result.Errors = append(result.Errors, Diagnostic{Code: "metadata.empty_path_segment", Message: "command.path contains an empty segment", File: file})
			}
		}
		if len(m.Command.Path) > 0 && reservedTopLevel[m.Command.Path[0]] {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.reserved_namespace", Message: fmt.Sprintf("%q is a reserved top-level namespace", m.Command.Path[0]), File: file})
		}
		pathKey := strings.Join(m.Command.Path, "\x00")
		if pathKey != "" {
			if prev := paths[pathKey]; prev != "" {
				result.Errors = append(result.Errors, Diagnostic{Code: "metadata.duplicate_path", Message: fmt.Sprintf("duplicate command path %q also appears in %s", strings.Join(m.Command.Path, " "), prev), File: file})
			} else {
				paths[pathKey] = file
			}
		}
		if len(m.Tags) == 0 {
			result.Warnings = append(result.Warnings, Diagnostic{Code: "metadata.missing_tags", Message: fmt.Sprintf("%s has no tags", m.ID), File: file})
		}
		if len(m.Examples) == 0 {
			result.Warnings = append(result.Warnings, Diagnostic{Code: "metadata.missing_examples", Message: fmt.Sprintf("%s has no examples", m.ID), File: file})
		}
		if len(m.Summary) > 140 {
			result.Warnings = append(result.Warnings, Diagnostic{Code: "metadata.long_summary", Message: fmt.Sprintf("%s summary is longer than 140 characters", m.ID), File: file})
		}
		if m.RiskLevel != "" && m.RiskLevel != "low" && m.RiskLevel != "medium" && m.RiskLevel != "high" {
			result.Errors = append(result.Errors, Diagnostic{Code: "metadata.invalid_risk_level", Message: fmt.Sprintf("%s risk_level must be low, medium, or high", m.ID), File: file})
		}
		if (m.MutatesFiles || m.MutatesExternalSystems) && !m.SupportsDryRun {
			result.Warnings = append(result.Warnings, Diagnostic{Code: "metadata.mutating_without_dry_run", Message: fmt.Sprintf("%s mutates state but does not declare dry-run support", m.ID), File: file})
		}
		if m.RiskLevel == "high" && !m.RequiresConfirmation {
			result.Warnings = append(result.Warnings, Diagnostic{Code: "metadata.high_risk_without_confirmation", Message: fmt.Sprintf("%s is high risk but does not require confirmation", m.ID), File: file})
		}
		if c, ok := registeredByID[m.ID]; ok {
			if strings.Join(c.Path, "\x00") != strings.Join(m.Command.Path, "\x00") {
				result.Errors = append(result.Errors, Diagnostic{Code: "metadata.path_mismatch", Message: fmt.Sprintf("%s metadata path does not match registered command path", m.ID), File: file})
			}
		}
	}
	for _, c := range commands {
		if c.ID == "" {
			result.Errors = append(result.Errors, Diagnostic{Code: "command.missing_id", Message: fmt.Sprintf("registered path %q has no id", strings.Join(c.Path, " "))})
		}
		if len(c.Path) == 0 {
			result.Errors = append(result.Errors, Diagnostic{Code: "command.empty_path", Message: fmt.Sprintf("%s has an empty path", c.ID)})
		}
		if _, ok := ids[c.ID]; c.ID != "" && !ok {
			result.Errors = append(result.Errors, Diagnostic{Code: "command.missing_metadata", Message: fmt.Sprintf("%s has no .agent.yaml metadata", c.ID)})
		}
		seenFlags := map[string]bool{}
		seenPositions := map[int]bool{}
		for _, p := range c.Parameters {
			switch p.Type {
			case "string", "integer", "number", "boolean", "enum", "path", "duration", "string[]", "path[]":
			default:
				result.Errors = append(result.Errors, Diagnostic{Code: "parameter.invalid_type", Message: fmt.Sprintf("%s parameter %s has invalid type %q", c.ID, p.Name, p.Type)})
			}
			if p.Type == "enum" && len(p.Values) == 0 {
				result.Errors = append(result.Errors, Diagnostic{Code: "parameter.enum_without_values", Message: fmt.Sprintf("%s parameter %s is enum without values", c.ID, p.Name)})
			}
			if p.Flag != "" {
				if seenFlags[p.Flag] {
					result.Errors = append(result.Errors, Diagnostic{Code: "parameter.duplicate_flag", Message: fmt.Sprintf("%s duplicates flag %s", c.ID, p.Flag)})
				}
				seenFlags[p.Flag] = true
			}
			if p.Position != nil {
				if seenPositions[*p.Position] {
					result.Errors = append(result.Errors, Diagnostic{Code: "parameter.duplicate_position", Message: fmt.Sprintf("%s duplicates positional index %d", c.ID, *p.Position)})
				}
				seenPositions[*p.Position] = true
			}
		}
	}
	return result
}
