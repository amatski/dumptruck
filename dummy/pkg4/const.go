package pkg4

import (
	"time"

	"code.justin.tv/safety/go2proto/dummy/pkg1"
	"code.justin.tv/safety/go2proto/dummy/pkg2/nest"
)

type D struct {
	Country   nest.Country
	A         pkg1.A
	CreatedAt time.Time
}
