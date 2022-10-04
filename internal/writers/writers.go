package writers

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.justin.tv/safety/go2proto/internal"
	astt "code.justin.tv/safety/go2proto/internal/ast"
)

func isPodType(t string) bool {
	return t == "interface" || t == "time.Time" || t == "time.Duration" || t == "int64" || t == "int32" || t == "int" || t == "double" || t == "float" || t == "string" || t == "bool" || t == "float64" || t == "float32" || t == "uint" || t == "uin32" || t == "uint64"
}

func WriteFile(filename string, data []byte) error {
	path := filepath.Dir(filename)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func writeProtoHeader(sb *strings.Builder, pkgName string, pkgPrefixSlash string) {
	sb.WriteString("syntax = \"proto3\";\n")
	sb.WriteString(fmt.Sprintf("package %s;\n", pkgName))
	sb.WriteString(fmt.Sprintf("option go_package = \"%s/%s\";\n\n", pkgPrefixSlash, pkgName))
	sb.WriteString("import \"google/protobuf/timestamp.proto\";\nimport \"google/protobuf/struct.proto\";\nimport \"google/protobuf/duration.proto\";\n\n")
}

func writeProtoHeaderForProtofile(p *ProtoFile, pkgName string, pkgPrefixSlash string) {
	p.GetSb().WriteString("syntax = \"proto3\";\n")
	p.GetSb().WriteString(fmt.Sprintf("package %s;\n", pkgName))
	p.GetSb().WriteString(fmt.Sprintf("option go_package = \"%s/%s\";\n\n", pkgPrefixSlash, pkgName))
	p.GetSb().WriteString("import \"google/protobuf/timestamp.proto\";\nimport \"google/protobuf/struct.proto\";\nimport \"google/protobuf/duration.proto\";\n\n")
}

func writeDepImports(sb *strings.Builder, deps internal.DependencySet) {
	sortedDeps := []*internal.Path{}
	for dep := range deps {
		sortedDeps = append(sortedDeps, deps[dep])
	}
	sort.Slice(sortedDeps, func(i, j int) bool {
		return *sortedDeps[i].FilePath < *sortedDeps[j].FilePath
	})

	for _, dep := range sortedDeps {
		protoFilePath, err := dep.ToProtoFilePath()
		if err != nil {
			panic(err)
		}
		sb.WriteString(fmt.Sprintf("import \"%s/const.proto\";\n", protoFilePath))
	}
	if len(sortedDeps) > 0 {
		sb.WriteString("\n")
	}
}

// writeField outputs package dependency types as strings
func writeField(parentNode *astt.GoNode, field *internal.Field, idx int, sb *strings.Builder) internal.DependencySet {
	opt := ""
	repeated := ""
	deps := internal.DependencySet{}
	var protoFilePathPtr *string

	// If the field has a selector use the import tree to figure out the converted proto file path
	if field.ComputeSelector() != nil { // generic way of prepending protobuf package prefix
		// TODO: refactor this
		importAlias := strings.Split(field.Type, ".")[0]
		if parentNode != nil {
			// get the imports for the current file
			imp := astt.FindImport(importAlias, parentNode.ImportsForPath(*field.Path.FilePath))
			if imp != nil {
				// importAlias is either an aliased import or a package name, find that matching import
				// and then use it
				protoFilePath, err := imp.GoNode.Path.ToProtoPackageFilePath()
				if err != nil {
					panic(err)
				}
				deps[protoFilePath] = &imp.GoNode.Path
				protoFilePathPtr = &protoFilePath
				// Override the type with the package path
				log.Println("overrided import ", importAlias, "with package name", imp.GoNode.PackageName)
			} else {
				log.Println("import not found for import alias", importAlias)
			}
		} else {
			deps[importAlias] = nil // non relative go import
		}
	}

	if field.Repeated {
		repeated = "repeated "
	} else if field.Optional {
		opt = "optional "
	}
	sb.WriteString(fmt.Sprintf("    %s%s%s %s = %d;\n", repeated, opt, field.ProtoType(protoFilePathPtr), field.Name, idx))
	return deps
}

func addDependencies(src internal.DependencySet, dst internal.DependencySet) internal.DependencySet {
	// add the dependencies to this dependency set
	for dep := range src {
		dst[dep] = src[dep]
	}
	return dst
}

func WriteServer(parentNode *astt.GoNode, funcs []internal.Function, rootPkgName, pkgPrefixSlash, outDir string) {
	var sb strings.Builder
	writeProtoHeader(&sb, rootPkgName, pkgPrefixSlash)

	// Write all request types
	deps := internal.DependencySet{}
	tmpSb := &strings.Builder{}

	// Build deps before in a tmp sb
	for _, f := range funcs {
		if len(f.Fields) > 1 {
			for idx, e := range f.Fields[1:] {
				addDependencies(writeField(parentNode, e, idx+1, tmpSb), deps)
			}
		}
		for idx, e := range f.ReturnTypes {
			if e.Type != "error" {
				e.Name = fmt.Sprintf("Field%d", idx+1)
				addDependencies(writeField(parentNode, e, idx+1, tmpSb), deps)
			}
		}
	}

	// Write the deps
	writeDepImports(&sb, deps)

	// Write all the real messages out for the server
	for _, f := range funcs {
		// Write request
		sb.WriteString(fmt.Sprintf("message %s {\n", f.Name+"Request"))
		if len(f.Fields) > 1 {
			for idx, e := range f.Fields[1:] {
				writeField(parentNode, e, idx+1, &sb)
			}
		}

		sb.WriteString("}\n\n")
		// Write response
		sb.WriteString(fmt.Sprintf("message %s {\n", f.Name+"Response"))
		for idx, e := range f.ReturnTypes {
			if e.Type != "error" {
				e.Name = fmt.Sprintf("Field%d", idx+1)
				writeField(parentNode, e, idx+1, &sb)
			}
		}

		sb.WriteString("}\n\n")
	}

	sb.WriteString("service Leviathan {\n")
	for _, f := range funcs {
		rets := "("
		for idx, r := range f.ReturnTypes {
			rets += r.Type
			if idx != len(f.ReturnTypes)-1 {
				rets += ","
			}
		}
		rets += ")"
		sb.WriteString(fmt.Sprintf("     rpc %s(%s) returns (%s);\n", f.Name, f.Name+"Request", f.Name+"Response"))
	}
	sb.WriteString("}\n")
	WriteFile(fmt.Sprintf("%s/server.proto", outDir), []byte(sb.String()))
}

func convertEnumsByDecl(assignments []internal.EnumAssignment) [][]internal.EnumAssignment {
	enumsByDecl := map[*ast.GenDecl][]internal.EnumAssignment{}
	// Populate enums per decl block
	for _, enum := range assignments {
		if _, ok := enumsByDecl[enum.Decl]; !ok {
			enumsByDecl[enum.Decl] = []internal.EnumAssignment{}
		}
		enumsByDecl[enum.Decl] = append(enumsByDecl[enum.Decl], enum)
	}

	enumsFlat := []([]internal.EnumAssignment){}

	for _, enums := range enumsByDecl {
		if len(enums) > 0 {
			enumsFlat = append(enumsFlat, enums)
		}
	}
	sort.Slice(enumsFlat, func(i, j int) bool {
		return enumsFlat[i][0].Name < enumsFlat[j][0].Name
	})
	return enumsFlat
}

func ToProtoFiles(parentNode *astt.GoNode, structs []internal.Struct, assignments []internal.EnumAssignment, pkgPrefixSlash string) map[string]*ProtoFile {
	protoFiles := map[string]*ProtoFile{}

	enumsFlat := convertEnumsByDecl(assignments)

	// First figure out all dependencies in that package if we were to write all the messages (structs)
	// and then when we want to write proto header we can write those imports in as well
	// Write import table + deps from struct fields
	writtenDeps := map[string]struct{}{}
	writtenProtoHeader := map[string]struct{}{}

	// Add any missing packages for packages that just contain enums
	for _, enum := range assignments {
		if _, ok := protoFiles[enum.Package]; !ok {
			protoFiles[enum.Package] = NewProtoFile(filepath.Dir(*enum.Path.GlobalFilePath) + "/const.go")
			// TODO: change this so it calculates the go_option package name
			protoPkg, err := enum.Path.ToProtoPackageFilePath()
			if err != nil {
				panic(err)
			}
			if _, ok := writtenProtoHeader[enum.Package]; !ok {
				writeProtoHeaderForProtofile(protoFiles[enum.Package], protoPkg, pkgPrefixSlash)
			}
			writtenProtoHeader[enum.Package] = struct{}{}
		}
	}

	// Write dep imports using a temp sb (pkg name -> deps)
	for _, s := range structs {
		tmpSb := &strings.Builder{}
		for idx, f := range s.Fields {
			fieldDeps := writeField(parentNode, f, idx+1, tmpSb)
			if _, ok := protoFiles[s.Package]; !ok {
				// Bugfix: when we see a const.go and a const2.go it'll sometimes write
				// const.proto and const2.proto unnecessarily so collapse them into const.go
				// for all structs
				protoFiles[s.Package] = NewProtoFile(filepath.Dir(*s.Path.GlobalFilePath) + "/const.go")
			}
			addDependencies(fieldDeps, protoFiles[s.Package].GetDeps())
		}
	}

	for _, s := range structs {
		if _, ok := writtenDeps[s.Package]; !ok {
			if _, ok2 := writtenProtoHeader[s.Package]; !ok2 {
				protoPkg, err := s.Path.ToProtoPackageFilePath()
				if err != nil {
					panic(err)
				}
				writeProtoHeaderForProtofile(protoFiles[s.Package], protoPkg, pkgPrefixSlash)
			}
			writeDepImports(protoFiles[s.Package].GetSb(), protoFiles[s.Package].GetDeps())
			writtenDeps[s.Package] = struct{}{}
		}
	}

	// all of the enums that are in the same package
	for _, enums := range enumsFlat {
		if len(enums) > 0 {
			sb := protoFiles[enums[0].Package].GetSb()
			enumName := enums[0].FuncName
			sb.WriteString(fmt.Sprintf("enum %s {\n", enumName)) // assumes every single one is the same in the enum which is ok
			for idx, enum := range enums {
				sb.WriteString(fmt.Sprintf("     %s = %d;\n", enum.Name, idx))
			}
			sb.WriteString(fmt.Sprintf("}\n\n"))
		}
	}

	for idx, s := range structs {
		var sb *strings.Builder = protoFiles[s.Package].GetSb()
		// Write the messages
		sb.WriteString(fmt.Sprintf("message %s {\n", s.Name))
		for idx, f := range s.Fields {
			writeField(parentNode, f, idx+1, sb)
		}
		sb.WriteString("}")
		if idx != len(structs)-1 {
			sb.WriteString("\n\n")
		}
	}
	return protoFiles
}
