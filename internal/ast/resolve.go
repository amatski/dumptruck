package ast

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"code.justin.tv/safety/go2proto/internal"
)

var (
	ErrMissingGoPath = errors.New("missing GOPATH")
	ErrNotAnImport   = errors.New("unexpected non import")
)

// GoNode is a single go file in a Go file import dependency tree
// it is useful for being able to figure out from an interface what
// files need to be processed to be transpiled (top down)
type GoNode struct {
	PackageName string // Default package name the Go node belongs to
	Path        internal.Path
	GoFiles     []string      // list of file names in the current path
	Imports     []*ImportNode // each of this package's imports
}

type ImportNode struct {
	Alias  *string // Optional import alias
	Path   string  // Optional import alias
	GoNode *GoNode // An import imports everything in that package as a GoNode
}

// ImportsForPath returns all of the ImportNodes for a given filepath
func (g *GoNode) ImportsForPath(filepath string) []*ImportNode {
	flattenedNodes := map[string]*GoNode{}
	crawl(g, flattenedNodes)
	for _, node := range flattenedNodes {
		if *node.Path.FilePath == filepath {
			return node.Imports
		}
	}
	return []*ImportNode{}
}

func FindImport(selector string, imports []*ImportNode) *ImportNode {
	for _, imp := range imports {
		if imp.Alias != nil && *imp.Alias == selector {
			return imp
		}
	}

	for _, imp := range imports {
		if imp.GoNode.PackageName == selector {
			return imp
		}
	}
	return nil
}

func (g *GoNode) AddImport(importNode *ImportNode) {
	g.Imports = append(g.Imports, importNode)
}

// UniqueLocalPaths returns a collapsed set of local file paths sorted alphabetically
func (g *GoNode) UniqueLocalFilePaths() []string {
	out := []string{}
	m := map[string]*GoNode{}

	crawl(g, m)
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// UniqueLocalPaths returns a collapsed set of local file paths
func (g *GoNode) UniqueLocalPaths() []string {
	out := []string{}
	m := map[string]*GoNode{}

	crawl(g, m)
	m2 := map[string]*GoNode{}
	for s := range m {
		m2[filepath.Dir(s)] = m[s]
	}
	for s := range m2 {
		out = append(out, s)
	}
	return out
}

func crawl(g *GoNode, m map[string]*GoNode) {
	if _, ok := m[*g.Path.FilePath]; !ok {
		m[*g.Path.FilePath] = g

		for _, e := range g.Imports {
			crawl(e.GoNode, m)
		}
	}
}

func goFilesInDir(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			out = append(out, file.Name())
		}
	}
	return out, nil
}

// ResolveGoTree builds a package tree (assuming the import tree is non cyclic)
// **Note that the path of inputFile is RELATIVE to your $GOPATH already and
// we deduce the global path from $GOPATH
func ResolveGoTree(inputFile string, rootPath string) (*GoNode, error) {
	return resolveGoTree(inputFile, rootPath, map[string]*GoNode{})
}

func resolveGoTree(inputFile string, rootPath string, cache map[string]*GoNode) (*GoNode, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return nil, ErrMissingGoPath
	}

	fset := token.NewFileSet()
	globalPath := fmt.Sprintf("%s/src/", gopath) // e.g /Users/gritaras/src/
	globalFilePath := globalPath + inputFile

	// Check the cache
	if _, ok := cache[globalFilePath]; ok {
		return cache[globalFilePath], nil
	}

	node, err := parser.ParseFile(fset, globalFilePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	currentPath := filepath.Dir(inputFile) // e.g code.justin.tv/safety/go2proto/dummy
	goFiles, err := goFilesInDir(globalPath + currentPath)
	if err != nil {
		return nil, err
	}

	globPath := filepath.Dir(globalFilePath)
	pathObj := internal.Path{
		Path:           &currentPath,
		GlobalPath:     &globPath,
		GlobalFilePath: &globalFilePath,
		FilePath:       &inputFile,
	}

	parentPackage := GoNode{
		PackageName: filepath.Base(*pathObj.Path),
		Path:        pathObj,
		GoFiles:     goFiles,
	}

	for _, decl := range node.Decls {
		if _, ok := decl.(*ast.GenDecl); ok {
			genDecl := decl.(*ast.GenDecl)
			if genDecl.Tok.String() == "import" {
				for _, spec := range genDecl.Specs {
					if astImport, ok := spec.(*ast.ImportSpec); ok {
						var importAlias *string = nil
						astImportPath := astImport.Path.Value[1 : len(astImport.Path.Value)-1]
						if astImport.Name != nil {
							importAlias = &astImport.Name.Name
						} else {
							// Hack (but reasonable?): the package name is always the last dir
							base := filepath.Base(astImportPath)
							importAlias = &base
						}

						// Check that the import is a relative import
						if strings.Contains(astImportPath, rootPath) {
							filesInImportPath, err := goFilesInDir(globalPath + astImportPath)
							if err != nil {
								return nil, err
							}

							// We'd think all the files in the same package are collapsed into a single file
							// and so are all of the imports
							// But it's a little different cause only aliases exist in files
							// and so we would almost consider that a "different package"

							// Recursively process all of the files in the import's package
							for _, fileName := range filesInImportPath {
								importNode, err := resolveGoTree(astImportPath+"/"+fileName, rootPath, cache)
								if err != nil {
									return nil, err
								}
								parentPackage.AddImport(&ImportNode{
									Alias:  importAlias,
									Path:   astImportPath,
									GoNode: importNode,
								})
							}
						}
					} else {
						return nil, ErrNotAnImport
					}

				}
			}
		}
	}

	// Store the node in the cache if it's not already there
	if _, ok := cache[globalFilePath]; !ok {
		cache[globalFilePath] = &parentPackage
	}

	return &parentPackage, nil
}
