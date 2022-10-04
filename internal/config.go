package internal

type TranspilerConfig struct {
	GoProjectPath  string
	PkgPrefix      string
	PkgPrefixSlash string
	RootPkgName    string
	OutDir         string
	InputDir       string
}

// GetTranspilerConfig returns the config for the transpiler
func GetTranspilerConfig() TranspilerConfig {
	return TranspilerConfig{
		GoProjectPath:  "code.justin.tv/safety/go2proto",
		PkgPrefix:      "code.justin.tv.safety.gateway.testserver",
		PkgPrefixSlash: "code.justin.tv/safety/gateway/testserver/rpc/testserver/gen",
		RootPkgName:    "root",
		OutDir:         "out",
		InputDir:       "models/",
	}
}
