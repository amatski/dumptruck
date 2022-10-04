package models

import (
	"context"

	"code.justin.tv/safety/go2proto/dummy/pkg1"
	nestpkg "code.justin.tv/safety/go2proto/dummy/pkg2/nest"
	"code.justin.tv/safety/go2proto/dummy/pkg4"
)

type TestInterface interface {
	GreatFunction() ([]string, error)
	GreatFunction2(ctx context.Context, arg *pkg1.A) ([]string, error)
	Function3(ctx context.Context) ([]string, []*string, error)
	Function4(ctx context.Context, limit uint64) ([]string, []*string, error)
	Function5(ctx context.Context, j int64) ([]string, []*string, error)
	Function6(ctx context.Context, c nestpkg.Country) ([]string, []*string, error)
	Function7(ctx context.Context, d pkg4.D) error
}
