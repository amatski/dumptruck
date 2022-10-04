package writers

import (
	"os"
	"path/filepath"
	"testing"

	"code.justin.tv/safety/go2proto/internal"
	"code.justin.tv/safety/go2proto/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestProtoWriters(t *testing.T) {
	transpilerConfig := internal.GetTranspilerConfig()
	goNode, err := ast.ResolveGoTree("code.justin.tv/safety/go2proto/dummy/interface.go", transpilerConfig.GoProjectPath)
	assert.NotNil(t, goNode)
	assert.NoError(t, err)

	gopath := os.Getenv("GOPATH")
	assert.NotEmpty(t, gopath)
	goSrcDir := gopath + "/src/"

	paths := goNode.UniqueLocalFilePaths()
	for idx := range paths {
		paths[idx] = goSrcDir + filepath.Dir(paths[idx])
	}

	result := ast.Parse(paths, goSrcDir)

	pkgToProtoFiles := ToProtoFiles(goNode, result.Structs, result.Enums, transpilerConfig.PkgPrefixSlash)
	assert.NotNil(t, pkgToProtoFiles)
	pkgNames := []string{}
	for s := range pkgToProtoFiles {
		pkgNames = append(pkgNames, s)
	}
	assert.Equal(t, []string{"nest", "pkg1", "pkg3", "pkg4", "meta"}, pkgNames)
}
