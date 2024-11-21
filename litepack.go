package litepack

import (
	"context"

	"github.com/lucasvillarinho/litepack/cache"
)

func NewCache(ctx context.Context, opts ...cache.Option) (cache.Cache, error) {
	return cache.NewCache(ctx, opts...)
}
