package benchmarks

import (
	"context"
	"testing"

	"github.com/lucasvillarinho/litepack"
)

func BenchmarkSet(b *testing.B) {
	ctx := context.Background()

	lcache, err := litepack.NewCache(ctx)
	if err != nil {
		b.Fatalf("Failed to initialize cache: %v", err)
	}
	defer lcache.Destroy(ctx)

	for i := 0; i < b.N; i++ {
		err := lcache.Set(ctx, "key", "test", 10)
		if err != nil {
			b.Errorf("Expected to set cache entry without error, but got: %v", err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkGet(b *testing.B) {

	ctx := context.Background()

	lcache, err := litepack.NewCache(ctx)
	if err != nil {
		b.Fatalf("Failed to initialize cache: %v", err)
	}
	defer lcache.Destroy(ctx)

	_ = lcache.Set(ctx, "key", "test", 10)

	for i := 0; i < b.N; i++ {
		_, err := lcache.Get(ctx, "key")
		if err != nil {
			b.Errorf("Expected to get cache entry without error, but got: %v", err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkDel(b *testing.B) {

	ctx := context.Background()

	lcache, err := litepack.NewCache(ctx)
	if err != nil {
		b.Fatalf("Failed to initialize cache: %v", err)
	}
	defer lcache.Destroy(ctx)

	_ = lcache.Set(ctx, "key", "test", 10)

	for i := 0; i < b.N; i++ {
		err := lcache.Del(ctx, "key")
		if err != nil {
			b.Errorf("Expected to delete cache entry without error, but got: %v", err)
		}
	}

	b.ReportAllocs()
}
