package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDependencyTree(t *testing.T) {
	parentPkg, err := ResolveGoTree("code.justin.tv/safety/go2proto/dummy/interface.go", "code.justin.tv/safety/go2proto")
	assert.NoError(t, err)
	assert.NotNil(t, parentPkg)
	assert.Equal(t, 6, len(parentPkg.UniqueLocalPaths()))

	assert.NotNil(t, parentPkg.Path.FilePath)
	assert.Equal(t, "code.justin.tv/safety/go2proto/dummy/interface.go", *parentPkg.Path.FilePath)
	assert.Equal(t, parentPkg.GoFiles, []string{"interface.go"})
	assert.Equal(t, 3, len(parentPkg.Imports))

	assert.NotNil(t, parentPkg.Imports[0].GoNode.Path.FilePath)
	assert.Equal(t, "code.justin.tv/safety/go2proto/dummy/pkg1/const.go", *parentPkg.Imports[0].GoNode.Path.FilePath)
	assert.Equal(t, 3, len(parentPkg.Imports[0].GoNode.Imports))

	assert.True(t, len(parentPkg.ImportsForPath("test")) == 0)
	assert.Equal(t, 3, len(parentPkg.ImportsForPath("code.justin.tv/safety/go2proto/dummy/pkg1/const.go")))

	for _, j := range parentPkg.Imports {
		assert.NotNil(t, j.Alias)
	}
	assert.NotNil(t, parentPkg.Imports[1].Alias)
	assert.NotNil(t, parentPkg.Imports[0].Alias)
	assert.Equal(t, "nestpkg", *parentPkg.Imports[1].Alias)
	assert.Equal(t, "pkg1", *parentPkg.Imports[0].Alias)
}
