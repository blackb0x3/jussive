package jussive

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Name      string        `json:"name" yaml:"name"`
	Changelog string        `json:"changelog" yaml:"changelog"`
	Runtime   RuntimeConfig `json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Release   ReleaseConfig `json:"release" yaml:"release"`
}

type RuntimeConfig struct {
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Entrypoint string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
}

type ReleaseConfig struct {
	VersionSource string `json:"version_source" yaml:"version_source"`
	TagPrefix     string `json:"tag_prefix" yaml:"tag_prefix"`
}

type CommandMetadata struct {
	ID                     string          `json:"id" yaml:"id"`
	Name                   string          `json:"name" yaml:"name"`
	Summary                string          `json:"summary" yaml:"summary"`
	Tags                   []string        `json:"tags,omitempty" yaml:"tags,omitempty"`
	Risk                   string          `json:"risk,omitempty" yaml:"risk,omitempty"`
	RiskLevel              string          `json:"risk_level,omitempty" yaml:"risk_level,omitempty"`
	ReadOnly               bool            `json:"read_only" yaml:"read_only"`
	MutatesFiles           bool            `json:"mutates_files" yaml:"mutates_files"`
	MutatesExternalSystems bool            `json:"mutates_external_systems" yaml:"mutates_external_systems"`
	SupportsDryRun         bool            `json:"supports_dry_run" yaml:"supports_dry_run"`
	RequiresConfirmation   bool            `json:"requires_confirmation" yaml:"requires_confirmation"`
	Command                MetadataCommand `json:"command" yaml:"command"`
	Inputs                 []InputMetadata `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Examples               []string        `json:"examples,omitempty" yaml:"examples,omitempty"`
	WhenToUse              []string        `json:"when_to_use,omitempty" yaml:"when_to_use,omitempty"`
	WhenNotToUse           []string        `json:"when_not_to_use,omitempty" yaml:"when_not_to_use,omitempty"`
	File                   string          `json:"-" yaml:"-"`
}

type MetadataCommand struct {
	Path []string `json:"path" yaml:"path"`
}

type InputMetadata struct {
	Name        string `json:"name" yaml:"name"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

func LoadProjectConfig(path string) (ProjectConfig, error) {
	var cfg ProjectConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func LoadMetadata(root string) ([]CommandMetadata, error) {
	var out []CommandMetadata
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".agent.yaml") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var m CommandMetadata
		if err := yaml.Unmarshal(b, &m); err != nil {
			return err
		}
		m.File = path
		out = append(out, m)
		return nil
	})
	return out, err
}
