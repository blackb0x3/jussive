package jussive

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	ExitSuccess          = 0
	ExitCommandFailed    = 1
	ExitInvalidUsage     = 2
	ExitValidationFailed = 3
	ExitNotFound         = 4
	ExitUnsafeBlocked    = 5
)

var ErrInvalidUsage = errors.New("invalid usage")

type Handler func(context.Context, Args) error

type Args struct {
	Positionals []string
	Flags       map[string]string
	Bools       map[string]bool
	Raw         []string
}

func (a Args) String(name string) string {
	if v, ok := a.Flags[name]; ok {
		return v
	}
	return ""
}

func (a Args) Bool(name string) bool {
	return a.Bools[name]
}

type Command struct {
	ID                  string
	Path                []string
	Summary             string
	Parameters          []Parameter
	AllowRunnableParent bool
	Run                 Handler
}

func (c Command) DisplayPath() string {
	return strings.Join(c.Path, " ")
}

type Parameter struct {
	Name         string   `json:"name" yaml:"name"`
	Type         string   `json:"type" yaml:"type"`
	Required     bool     `json:"required,omitempty" yaml:"required,omitempty"`
	Position     *int     `json:"position,omitempty" yaml:"position,omitempty"`
	Flag         string   `json:"flag,omitempty" yaml:"flag,omitempty"`
	Description  string   `json:"description,omitempty" yaml:"description,omitempty"`
	Values       []string `json:"values,omitempty" yaml:"values,omitempty"`
	Default      any      `json:"default,omitempty" yaml:"default,omitempty"`
	Deprecated   bool     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	ReplacedBy   string   `json:"replaced_by,omitempty" yaml:"replaced_by,omitempty"`
	RemovalAfter string   `json:"removal_after,omitempty" yaml:"removal_after,omitempty"`
}

func String(name string) ParameterBuilder    { return newParameter(name, "string") }
func Integer(name string) ParameterBuilder   { return newParameter(name, "integer") }
func Number(name string) ParameterBuilder    { return newParameter(name, "number") }
func Bool(name string) ParameterBuilder      { return newParameter(name, "boolean") }
func Enum(name string) ParameterBuilder      { return newParameter(name, "enum") }
func PathParam(name string) ParameterBuilder { return newParameter(name, "path") }
func Duration(name string) ParameterBuilder  { return newParameter(name, "duration") }
func StringArray(name string) ParameterBuilder {
	return newParameter(name, "string[]")
}
func PathArray(name string) ParameterBuilder {
	return newParameter(name, "path[]")
}

type ParameterBuilder struct {
	p Parameter
}

func newParameter(name, typ string) ParameterBuilder {
	return ParameterBuilder{p: Parameter{Name: name, Type: typ}}
}

func (b ParameterBuilder) Required() ParameterBuilder {
	b.p.Required = true
	return b
}

func (b ParameterBuilder) Position(n int) ParameterBuilder {
	b.p.Position = &n
	return b
}

func (b ParameterBuilder) Flag(name string) ParameterBuilder {
	b.p.Flag = name
	return b
}

func (b ParameterBuilder) Description(text string) ParameterBuilder {
	b.p.Description = text
	return b
}

func (b ParameterBuilder) Values(values ...string) ParameterBuilder {
	b.p.Values = append([]string(nil), values...)
	return b
}

func (b ParameterBuilder) Default(value any) ParameterBuilder {
	b.p.Default = value
	return b
}

func (b ParameterBuilder) Deprecated(replacedBy, removalAfter string) ParameterBuilder {
	b.p.Deprecated = true
	b.p.ReplacedBy = replacedBy
	b.p.RemovalAfter = removalAfter
	return b
}

func (b ParameterBuilder) Build() Parameter {
	return b.p
}

func Path(text string) []string {
	fields := strings.Fields(text)
	return append([]string(nil), fields...)
}

type BuildInfo struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Commit  string `json:"commit" yaml:"commit"`
	Dirty   bool   `json:"dirty" yaml:"dirty"`
	BuiltAt string `json:"built_at" yaml:"built_at"`
}

type Envelope struct {
	OK       bool         `json:"ok" yaml:"ok"`
	Data     any          `json:"data" yaml:"data"`
	Warnings []Diagnostic `json:"warnings" yaml:"warnings"`
	Errors   []Diagnostic `json:"errors" yaml:"errors"`
}

type Diagnostic struct {
	Code    string `json:"code" yaml:"code"`
	Message string `json:"message" yaml:"message"`
	File    string `json:"file,omitempty" yaml:"file,omitempty"`
}

func UsageError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidUsage, fmt.Sprintf(format, args...))
}
