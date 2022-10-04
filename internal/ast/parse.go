package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"code.justin.tv/safety/go2proto/internal"
)

type ParseResult struct {
	Funcs       []internal.Function
	Structs     []internal.Struct
	PodTypedefs []internal.PodTypedef
	Enums       []internal.EnumAssignment
}

func (r *ParseResult) ApplyOverrides(fieldOverrides []internal.FieldTypeOverride, enumOverrides []internal.EnumOverride) {
	for idx := range r.Structs {
		r.Structs[idx].ApplyOverrides(fieldOverrides)
	}

	for idx := range r.Funcs {
		r.Funcs[idx].ApplyOverrides(fieldOverrides)
	}

	for idx := range r.Enums {
		r.Enums[idx].ApplyOverrides(enumOverrides)
	}
}

func Parse(paths []string, goSrcDir string) ParseResult {
	functions := []internal.Function{}
	structs := []internal.Struct{}
	podTypedefs := []internal.PodTypedef{}
	assignments := []internal.EnumAssignment{}
	parsedDirs := map[string]struct{}{}

	for _, globalPath := range paths {
		path, err := filepath.Rel(goSrcDir, globalPath)
		if err != nil {
			panic(err)
		}
		//Create a FileSet to work with
		// note this is a bunch of files so that's why we have to index them
		fset := token.NewFileSet()

		// Bugfix: skip already parsed dirs
		if _, ok := parsedDirs[globalPath]; ok {
			continue
		}

		//Parse the file and create an AST
		pkgs, err := parser.ParseDir(fset, globalPath, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		parsedDirs[globalPath] = struct{}{}
		for pkgName, pkg := range pkgs {
			for globalFilePath, file := range pkg.Files {
				filePath, err := filepath.Rel(goSrcDir, globalFilePath)
				if err != nil {
					panic(err)
				}

				pathObj := internal.Path{
					Path:           &path,
					FilePath:       &filePath,
					GlobalFilePath: &globalFilePath,
					GlobalPath:     &globalPath,
				}

				for _, n := range file.Decls {
					switch n.(type) {
					case *ast.GenDecl:
						genDecl := n.(*ast.GenDecl)
						for _, spec := range genDecl.Specs {
							switch spec.(type) {
							case *ast.ValueSpec:
								value := spec.(*ast.ValueSpec)
								funcName := "Type"

								if j, ok := value.Type.(*ast.Ident); ok {
									funcName = j.Name
								}

								enum := internal.EnumAssignment{
									Path:           pathObj,
									UnderlyingType: "int", // always int by default
									Package:        pkgName,
									FuncName:       funcName,
									Name:           value.Names[0].Name,
									Decl:           genDecl,
								}

								// If values[0] is ast.Ident OR nil then we just append it default
								if value.Values != nil {
									if len(value.Values) != 1 {
										panic("Length of enum block values should be 1")
									}
									if t, ok := value.Type.(*ast.Ident); ok {
										enum.FuncName = t.Name
									}

									if v, ok := value.Values[0].(*ast.Ident); ok {
										if v.Name == "iota" {
											assignments = append(assignments, enum)
										}
									}
								} else if value.Values == nil {
									assignments = append(assignments, enum)
								}
								for _, expr := range value.Values {
									if call, ok := expr.(*ast.CallExpr); ok {
										if ident, ok := call.Fun.(*ast.Ident); ok {
											enum.FuncName = ident.Name
											// skip make
											if enum.FuncName != "make" {
												if len(call.Args) > 0 {
													if lit, ok := call.Args[0].(*ast.BasicLit); ok {
														enum.UnderlyingType = strings.ToLower(lit.Kind.String())
													}
												}
												assignments = append(assignments, enum)
											}
										}
									} else if lit, ok := expr.(*ast.BasicLit); ok {
										enum.UnderlyingType = strings.ToLower(lit.Kind.String())
										assignments = append(assignments, enum)
									}
								}

							case *ast.TypeSpec:
								typeSpec := spec.(*ast.TypeSpec)

								switch typeSpec.Type.(type) {
								case *ast.InterfaceType:
									interfaces := typeSpec.Type.(*ast.InterfaceType)
									for _, field := range interfaces.Methods.List {
										if fun, ok := field.Type.(*ast.FuncType); ok {
											funcName := field.Names[0].Name
											funcImpl := internal.Function{Name: funcName}

											funcImpl.Fields = internal.ProcessFields(fun.Params.List, pkgName, pathObj)
											funcImpl.ReturnTypes = internal.ProcessFields(fun.Results.List, pkgName, pathObj)
											functions = append(functions, funcImpl)
										} else {
											panic("unexpected non func in interface")
										}
									}

								case *ast.StructType:
									structType := typeSpec.Type.(*ast.StructType)
									structImpl := internal.Struct{Path: pathObj, Package: pkgName, Name: typeSpec.Name.Name}
									structImpl.Fields = internal.ProcessFields(structType.Fields.List, pkgName, pathObj)
									structs = append(structs, structImpl)
								case *ast.Ident:
									// found a type assignment that is an identifier not a struct
									ident := typeSpec.Type.(*ast.Ident)
									podTypedefs = append(podTypedefs, internal.PodTypedef{
										Package: pkgName,
										Path:    pathObj,
										Name:    typeSpec.Name.Name,
										Type:    ident.Name,
									})
								case *ast.ArrayType:
									arr := typeSpec.Type.(*ast.ArrayType)
									// this type is an alias on an array type, treat it like an array struct
									structImpl := internal.Struct{Path: pathObj, Package: pkgName, Name: typeSpec.Name.Name}
									/*
																			Repeated bool
										Path     string
										Package  string
										Name     string
										Type     string // either a POD type or some other struct type
										Optional bool

									*/

									if iden, ok := arr.Elt.(*ast.Ident); ok {
										structImpl.Fields = []*internal.Field{
											{
												Path:     pathObj,
												Package:  pkgName,
												Repeated: true,
												Name:     "Elements",
												Type:     iden.Name,
											},
										}
									} else {
										panic("unimplemented")
									}
									structs = append(structs, structImpl)
								default:
									/*structImpl := internal.Struct{Path: path, Package: pkgName, Name: typeSpec.Name.Name}
									structImpl.Fields = append(structImpl.Fields, internal.Field{
										Name: "unknown_name",
										Type: "unknown_type",
									})
									structs = append(structs, structImpl)*/
									log.Println("Ignoring type ", globalPath, pkgName, typeSpec.Name.Name)
								}
							}
						}
					}
				}
			}
		}
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

	sort.Slice(structs, func(i, j int) bool {
		return structs[i].Name < structs[j].Name
	})

	sort.Slice(assignments, func(i, j int) bool {
		return assignments[i].Name < assignments[j].Name
	})

	sort.Slice(podTypedefs, func(i, j int) bool {
		return podTypedefs[i].Name < podTypedefs[j].Name
	})

	return ParseResult{
		Funcs:       functions,
		Structs:     structs,
		PodTypedefs: podTypedefs,
		Enums:       assignments,
	}
}
