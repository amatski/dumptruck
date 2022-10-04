package writers

import (
	"fmt"
	"go/ast"
	"os"
	"strings"

	"code.justin.tv/safety/go2proto/internal"
)

func WriteEnumConverters(assignments []internal.EnumAssignment, pods []internal.PodTypedef, pkgPrefixSlash string) {
	pkgFiles := map[string]*strings.Builder{} // each converter is 1:1 with the package it belongs to

	// Write header of converter file
	for _, enum := range assignments {
		if _, ok := pkgFiles[enum.Package]; !ok {
			pkgFiles[enum.Package] = &strings.Builder{}

			sb := pkgFiles[enum.Package]
			sb.WriteString(fmt.Sprintf("package %s\n\n", enum.Package))
			sb.WriteString("import (\n")

			pkg := enum.Package
			sb.WriteString(fmt.Sprintf("    pb%s \"%s/%s\"\n", pkg, pkgPrefixSlash, pkg))
			sb.WriteString(fmt.Sprintf("    %s \"code.justin.tv/safety/datastore/v5/%s\"\n", pkg, *enum.Path.Path))

			sb.WriteString(")\n\n")
		}
	}

	podGoTypes := map[string]internal.PodTypedef{}
	for _, pod := range pods {
		podGoTypes[fmt.Sprintf("%s.%s", pod.Package, pod.Name)] = pod
	}

	enumsByDecl := map[*ast.GenDecl][]internal.EnumAssignment{}
	// Populate enums per decl block
	for _, enum := range assignments {
		if _, ok := enumsByDecl[enum.Decl]; !ok {
			enumsByDecl[enum.Decl] = []internal.EnumAssignment{}
		}
		enumsByDecl[enum.Decl] = append(enumsByDecl[enum.Decl], enum)
	}

	for _, enums := range enumsByDecl {
		if len(enums) > 0 {
			e := enums[0]
			funcName := e.FuncName
			nullValue := "\"\""
			if e.UnderlyingType == "int" || e.UnderlyingType == "int32" || e.UnderlyingType == "int64" {
				nullValue = "-1"
			}
			sb := pkgFiles[e.Package]
			goType := fmt.Sprintf("%s.%s", e.Package, funcName)
			/*
				// check if its a pod type
				// if go type in a map of pkg + name then return that pod type
				if pod, ok := podGoTypes[goType]; ok {
					goType = pod.Type
				}*/

			sb.WriteString(fmt.Sprintf("func %sFromPb(e pb%s) %s {\n", funcName, goType, goType))
			sb.WriteString("        switch e{\n")
			for _, e := range enums {
				sb.WriteString(fmt.Sprintf("            case pb%s_%s:\n", goType, e.Name))
				sb.WriteString(fmt.Sprintf("                return %s.%s\n", e.Package, e.Name))
			}
			sb.WriteString("        }\n")
			sb.WriteString(fmt.Sprintf("        return %s(%s)\n", goType, nullValue))
			sb.WriteString("}\n\n")

			sb.WriteString(fmt.Sprintf("func %sFromPbPtr(e *pb%s) *%s {\n", funcName, goType, goType))
			sb.WriteString("        if e == nil{\n")
			sb.WriteString("            return nil\n")
			sb.WriteString("        }\n")
			sb.WriteString("        switch *e{\n")
			for _, e := range enums {
				sb.WriteString(fmt.Sprintf("            case pb%s.%s_%s:\n", e.Package, funcName, e.Name))
				sb.WriteString(fmt.Sprintf("                var ret %s = %s.%s\n", goType, e.Package, e.Name))
				sb.WriteString("                return &ret\n")
			}
			sb.WriteString("        }\n")
			sb.WriteString("        return nil\n")
			sb.WriteString("}\n\n")

			sb.WriteString(fmt.Sprintf("func %sFromGo(e %s) pb%s {\n", funcName, goType, goType))
			sb.WriteString("        switch e{\n")
			for _, e := range enums {
				sb.WriteString(fmt.Sprintf("            case %s.%s:\n", e.Package, e.Name))
				sb.WriteString(fmt.Sprintf("                return pb%s_%s\n", goType, e.Name))
			}
			sb.WriteString("        }\n")
			sb.WriteString(fmt.Sprintf("        return pb%s(%s)\n", goType, "-1"))
			sb.WriteString("}\n\n")

			sb.WriteString(fmt.Sprintf("func %sFromGoPtr(e *%s) *pb%s {\n", funcName, goType, goType))
			sb.WriteString("        if e == nil{\n")
			sb.WriteString("            return nil\n")
			sb.WriteString("        }\n")
			sb.WriteString("        switch *e{\n")
			for _, e := range enums {
				sb.WriteString(fmt.Sprintf("            case %s.%s:\n", e.Package, e.Name))
				sb.WriteString(fmt.Sprintf("                var ret pb%s = pb%s_%s\n", goType, goType, e.Name))
				sb.WriteString("                return &ret\n")
			}
			sb.WriteString("        }\n")
			sb.WriteString("        return nil\n")
			sb.WriteString("}\n\n")
		}
	}

	for pkg, sb := range pkgFiles {
		if _, err := os.Stat(fmt.Sprintf("converters/%s", pkg)); os.IsNotExist(err) {
			os.Mkdir(fmt.Sprintf("converters/%s", pkg), 0755)
		}
		WriteFile(fmt.Sprintf("converters/%s/enum.go", pkg), []byte(sb.String()))
	}
}
