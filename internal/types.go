package internal

import (
	"go/ast"
	"path/filepath"
	"strings"
)

// A dependency set is a set of paths (useful cause we can go to/from proto paths using this)
type DependencySet map[string]*Path

// Path is a super useful struct for dealing with all the confusing path stuff
type Path struct {
	FilePath       *string
	Path           *string
	GlobalFilePath *string
	GlobalPath     *string
}

// ToProtoFilePath uses the global path to make the equivalent collapsed proto file path
func (p *Path) ToProtoFilePath() (string, error) {
	r, err := filepath.Rel("code.justin.tv/safety/go2proto", *p.Path)
	if err != nil {
		return "", err
	}

	return r, nil
}

func (p *Path) ToProtoPackageFilePath() (string, error) {
	r, err := filepath.Rel("code.justin.tv/safety/go2proto", *p.Path)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(r, "/", "."), nil
}

type PodTypedef struct {
	Path    Path
	Package string
	Name    string
	Type    string
}

type EnumAssignment struct {
	Package        string
	Path           Path
	Name           string
	FuncName       string       // eg for A = B("C") this would be B
	Decl           *ast.GenDecl // use this to determine which block each assignment belongs to
	UnderlyingType string       // int or string
}

type Field struct {
	Repeated bool
	Path     Path
	Package  string
	Name     string
	Type     string // either a POD type or some other struct type
	Optional bool
	Selector bool // needed to know if this field contains a reference to another package
}

func (f *Field) ProtoType(protoPackageFilePath *string) string {
	if f.Type == "time.Time" || f.Type == "Time" {
		return "google.protobuf.Timestamp"
	} else if f.Type == "time.Duration" || f.Type == "Duration" {
		return "google.protobuf.Duration"
	} else if f.Type == "float64" {
		return "double"
	} else if f.Type == "int" {
		return "int32"
	} else if f.Type == "float32" {
		return "float"
	} else if f.Type == "interface{}" || f.Type == "interface" {
		return "google.protobuf.Value"
	}

	// Since proto file path is not nil we implicitly have a selector
	if protoPackageFilePath != nil {
		return *protoPackageFilePath + "." + f.ComputeSelector().Name
	}
	return f.Type
}

// A selector on a field
type Selector struct {
	Package string
	Name    string
}

type Struct struct {
	Path    Path
	Package string
	Name    string
	Fields  []*Field
}

type Function struct {
	Name        string
	Fields      []*Field
	ReturnTypes []*Field
}

// Takes in a field and returns true if it overrode the type
// Used for applying type aliases
type FieldTypeOverride func(f *Field, parentFunc *Function, parentStruct *Struct) bool
type EnumOverride func(e *EnumAssignment) bool

func applyOverrides(fields []*Field, overrides []FieldTypeOverride, parentFunc *Function, parentStruct *Struct) {
	for _, f := range fields {
		for _, override := range overrides {
			override(f, parentFunc, parentStruct)
		}
	}
}

func (s *Struct) ApplyOverrides(overrides []FieldTypeOverride) {
	applyOverrides(s.Fields, overrides, nil, s)
}

func (f *Function) ApplyOverrides(overrides []FieldTypeOverride) {
	applyOverrides(f.Fields, overrides, f, nil)
	applyOverrides(f.ReturnTypes, overrides, f, nil)
}

// Returns the raw selector type from the field type
func (f *Field) ComputeSelector() *Selector {
	if !f.Selector {
		return nil
	}

	if strings.Contains(f.Type, ".") { // generic way of prepending protobuf package prefix
		s := strings.Split(f.Type, ".")
		return &Selector{
			Package: s[0],
			Name:    s[1],
		}
	}
	return nil
}

func (e *EnumAssignment) ApplyOverrides(overrides []EnumOverride) {
	for _, override := range overrides {
		override(e)
	}
}
