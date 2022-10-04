package writers

import (
	"path/filepath"
	"strings"

	"code.justin.tv/safety/go2proto/internal"
)

type ProtoFile struct {
	packagePath string // the relative path of the package
	filePath    string
	sb          *strings.Builder
	deps        internal.DependencySet
}

func NewProtoFile(filePath string) *ProtoFile {
	return &ProtoFile{
		packagePath: filepath.Dir(filePath),
		filePath:    filePath,
		sb:          &strings.Builder{},
		deps:        internal.DependencySet{},
	}
}

func (p *ProtoFile) GetSb() *strings.Builder {
	return p.sb
}

func (p *ProtoFile) GetDeps() internal.DependencySet {
	return p.deps
}

func (p *ProtoFile) GetPackagePath() string {
	return p.packagePath
}

func (p *ProtoFile) GetFilePath() string {
	return p.filePath
}

func (p *ProtoFile) GetFileNameWithoutExtension() string {
	filename := p.GetFileName()
	if pos := strings.LastIndexByte(filename, '.'); pos != -1 {
		return filename[:pos]
	}
	return filename
}

// GetFileName returns the filename
func (p *ProtoFile) GetFileName() string {
	return filepath.Base(p.filePath)
}
