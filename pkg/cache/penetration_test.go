package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func newTestPenetration(enableBloom bool) (*penetrationGuard, *memoryCache) {
	inner := newTestMemory(0, 0)
	cfg := &PenetrationConfig{
		EnableBloom: enableBloom,
		BloomN:      1_000,
		BloomFP:     0.01,
		NullTTL:     time.Minute,
	}
	g, _ := newPenetrationGuard(inner, cfg, JSONSerializer{})
	return g, inner
}

func TestPenetration_NullKey_BlocksGet(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	inner.Set(ctx, g.nullKey("k"), true, time.Minute)

	var v string
	if err := g.Get(ctx, "k", &v); !errors.Is(err, ErrNotFound) {
		t.Fatalf("null key should block get, got %v", err)
	}
}

func TestPenetration_GetOrSet_StoresNullKey(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	var v string
	err := g.GetOrSet(ctx, "k", &v, time.Hour, func() (any, error) {
		return nil, ErrNotFound
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if ok, _ := inner.Exists(ctx, g.nullKey("k")); !ok {
		t.Fatal("null key should be stored in inner cache")
	}
	// 后续 Get 被拦截
	if err := g.Get(ctx, "k", &v); !errors.Is(err, ErrNotFound) {
		t.Fatalf("subsequent get should return ErrNotFound, got %v", err)
	}
}

func TestPenetration_GetOrSet_NonErrNotFound_NoNullKey(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	dbErr := errors.New("db error")
	var v string
	err := g.GetOrSet(ctx, "k", &v, time.Hour, func() (any, error) {
		return nil, dbErr
	})
	if !errors.Is(err, dbErr) {
		t.Fatalf("want dbErr, got %v", err)
	}
	if ok, _ := inner.Exists(ctx, g.nullKey("k")); ok {
		t.Fatal("null key should NOT be stored for non-ErrNotFound errors")
	}
}

func TestPenetration_Set_ClearsNullKey(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	inner.Set(ctx, g.nullKey("k"), true, time.Minute)
	g.Set(ctx, "k", "real", time.Hour)

	if ok, _ := inner.Exists(ctx, g.nullKey("k")); ok {
		t.Fatal("null key should be cleared after Set")
	}
	var v string
	if err := g.Get(ctx, "k", &v); err != nil {
		t.Fatalf("should find real value, got %v", err)
	}
	if v != "real" {
		t.Fatalf("want real, got %s", v)
	}
}

func TestPenetration_Del_ClearsNullKey(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	inner.Set(ctx, g.nullKey("k"), true, time.Minute)
	g.Del(ctx, "k")

	if ok, _ := inner.Exists(ctx, g.nullKey("k")); ok {
		t.Fatal("null key should be cleared after Del")
	}
}

func TestPenetration_MSet_ClearsNullKeys(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	inner.Set(ctx, g.nullKey("a"), true, time.Minute)
	inner.Set(ctx, g.nullKey("b"), true, time.Minute)
	g.MSet(ctx, map[string]any{"a": 1, "b": 2}, time.Hour)

	for _, k := range []string{"a", "b"} {
		if ok, _ := inner.Exists(ctx, g.nullKey(k)); ok {
			t.Errorf("null key for %s should be cleared after MSet", k)
		}
	}
}

func TestPenetration_Bloom_BlocksUnknownKey(t *testing.T) {
	g, inner := newTestPenetration(true)
	defer inner.Close()
	ctx := context.Background()

	var v string
	if err := g.Get(ctx, "unknown", &v); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bloom should block unknown key, got %v", err)
	}
}

func TestPenetration_Bloom_AllowsKnownKey(t *testing.T) {
	g, inner := newTestPenetration(true)
	defer inner.Close()
	ctx := context.Background()

	g.Set(ctx, "known", "val", 0)

	var v string
	if err := g.Get(ctx, "known", &v); err != nil {
		t.Fatalf("known key should pass bloom, got %v", err)
	}
	if v != "val" {
		t.Fatalf("want val, got %s", v)
	}
}

func TestPenetration_Bloom_Flush_ClearsFilter(t *testing.T) {
	g, inner := newTestPenetration(true)
	defer inner.Close()
	ctx := context.Background()

	g.Set(ctx, "k", "v", 0)
	g.Flush(ctx)

	var v string
	if err := g.Get(ctx, "k", &v); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after flush bloom should block k, got %v", err)
	}
}

func TestPenetration_Bloom_ThreadSafety(t *testing.T) {
	g, inner := newTestPenetration(true)
	defer inner.Close()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		n := i
		go func() {
			key := string(rune('a' + n))
			g.Set(ctx, key, n, 0)
			var v int
			g.Get(ctx, key, &v)
		}()
	}
}

func TestPenetration_Exists_NullKeyBlocks(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	inner.Set(ctx, g.nullKey("k"), true, time.Minute)
	if ok, _ := g.Exists(ctx, "k"); ok {
		t.Fatal("Exists should return false when null key present")
	}
}

func TestPenetration_IncrBy_Passthrough(t *testing.T) {
	g, inner := newTestPenetration(false)
	defer inner.Close()
	ctx := context.Background()

	n, err := g.IncrBy(ctx, "cnt", 5)
	if err != nil || n != 5 {
		t.Fatalf("want 5, got %d, err %v", n, err)
	}
	if ok, _ := inner.Exists(ctx, "cnt"); !ok {
		t.Fatal("cnt should exist in inner")
	}
}
