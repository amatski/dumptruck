package pkg1

import (
	"code.justin.tv/safety/go2proto/dummy/pkg3"
	"code.justin.tv/safety/go2proto/meta"
)

type A struct {
	Message string
	Flag    bool
	Alt     *string
}

type Q struct {
	F meta.FF
}

type C struct {
	Value pkg3.B
}
