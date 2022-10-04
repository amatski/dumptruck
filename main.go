package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"code.justin.tv/safety/go2proto/internal"
	"code.justin.tv/safety/go2proto/internal/ast"
	"code.justin.tv/safety/go2proto/internal/writers"
)

// WriteFile creates a new file given a path and if the directories don't exist will create it
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

func main() {
	fieldOverrides := []internal.FieldTypeOverride{
		func(f *internal.Field, parentFunc *internal.Function, parentStruct *internal.Struct) bool {
			// replace wizard path fields w string array
			if f.Type == "WizardPath" || f.Type == "ContentTags" {
				f.Type = "StringArray"
				return true
			}
			return false
		},
	}
	enumOverrides := []internal.EnumOverride{
		func(e *internal.EnumAssignment) bool {
			// Hack: Force assign the name of the sort type enum to SortType
			if e.Name == "SortAscending" || e.Name == "SortDescending" {
				e.FuncName = "SortType"
				return true
			}
			return false
		},
	}

	transpilerConfig := internal.GetTranspilerConfig()

	goNode, err := ast.ResolveGoTree("code.justin.tv/safety/go2proto/dummy/interface.go", transpilerConfig.GoProjectPath)
	if err != nil {
		panic(err)
	}

	gopath := os.Getenv("GOPATH")
	goSrcDir := gopath + "/src/"

	paths := goNode.UniqueLocalFilePaths()
	for idx := range paths {
		paths[idx] = goSrcDir + filepath.Dir(paths[idx])
	}

	result := ast.Parse(paths, goSrcDir)

	result.ApplyOverrides(fieldOverrides, enumOverrides)

	// After parsing we want to map the selectors to the separate protobuf types
	// and actually rename our headers tbh to update the . separated pkg path
	// But then how does a proto file import another file that is in another directory not relative to it? I think it
	// requires a flat directory structure -> no it just requires go_package to be specified
	// https://jbrandhorst.com/post/go-protobuf-tips/

	pkgToProtoFiles := writers.ToProtoFiles(goNode, result.Structs, result.Enums, transpilerConfig.PkgPrefixSlash)
	for _, protoFile := range pkgToProtoFiles {
		rel, err := filepath.Rel(goSrcDir+transpilerConfig.GoProjectPath, protoFile.GetPackagePath())
		if err != nil {
			panic(err)
		}
		err = WriteFile(fmt.Sprintf("%s/%s/%s.proto", transpilerConfig.OutDir, rel, protoFile.GetFileNameWithoutExtension()), []byte(protoFile.GetSb().String()))
		if err != nil {
			panic(err)
		}
	}

	writers.WriteServer(goNode, result.Funcs, transpilerConfig.RootPkgName, transpilerConfig.PkgPrefixSlash, transpilerConfig.OutDir)

	// Write all convertors for each struct and fields recursively
	// Convert the enums to and from golang type
	writers.WriteEnumConverters(result.Enums, result.PodTypedefs, transpilerConfig.PkgPrefixSlash)

	// use the deps from when we write the structs/rpc types to figure out deps for converters
	//writers.WriteStructConverters(result.Structs, deps, pkgPrefixSlash)
	log.Println("Done writing")
}
