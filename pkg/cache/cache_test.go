package cache

import (
	"context"
	"testing"
	"time"
)

// TestMemoryCache 测试内存缓存
func TestMemoryCache(t *testing.T) {
	ctx := context.Background()

	// 创建内存缓存
	c, err := NewWithOptions(
		WithMemory(DefaultMemoryConfig()),
		WithKeyPrefix("test:"),
	)
	if err != nil {
		t.Fatalf("failed to create memory cache: %v", err)
	}
	defer c.Close()

	// 测试 Set/Get
	t.Run("Set/Get", func(t *testing.T) {
		type User struct {
			ID   int64
			Name string
		}

		user := User{ID: 123, Name: "Alice"}
		err := c.Set(ctx, "user:123", user, 10*time.Minute)
		if err != nil {
			t.Fatalf("failed to set cache: %v", err)
		}

		var cachedUser User
		err = c.Get(ctx, "user:123", &cachedUser)
		if err != nil {
			t.Fatalf("failed to get cache: %v", err)
		}

		if cachedUser.ID != user.ID || cachedUser.Name != user.Name {
			t.Errorf("cached user mismatch: got %+v, want %+v", cachedUser, user)
		}
	})

	// 测试 Delete
	t.Run("Delete", func(t *testing.T) {
		err := c.Set(ctx, "key1", "value1", 10*time.Minute)
		if err != nil {
			t.Fatalf("failed to set cache: %v", err)
		}

		err = c.Delete(ctx, "key1")
		if err != nil {
			t.Fatalf("failed to delete cache: %v", err)
		}

		var value string
		err = c.Get(ctx, "key1", &value)
		if err != ErrCacheNotFound {
			t.Errorf("expected ErrCacheNotFound, got %v", err)
		}
	})

	// 测试 Exists
	t.Run("Exists", func(t *testing.T) {
		err := c.Set(ctx, "key2", "value2", 10*time.Minute)
		if err != nil {
			t.Fatalf("failed to set cache: %v", err)
		}

		exists, err := c.Exists(ctx, "key2")
		if err != nil {
			t.Fatalf("failed to check existence: %v", err)
		}
		if !exists {
			t.Error("expected key to exist")
		}

		exists, err = c.Exists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("failed to check existence: %v", err)
		}
		if exists {
			t.Error("expected key to not exist")
		}
	})

	// 测试 Incr/Decr
	t.Run("Incr/Decr", func(t *testing.T) {
		val, err := c.Incr(ctx, "counter")
		if err != nil {
			t.Fatalf("failed to incr: %v", err)
		}
		if val != 1 {
			t.Errorf("expected 1, got %d", val)
		}

		val, err = c.IncrBy(ctx, "counter", 10)
		if err != nil {
			t.Fatalf("failed to incrby: %v", err)
		}
		if val != 11 {
			t.Errorf("expected 11, got %d", val)
		}

		val, err = c.Decr(ctx, "counter")
		if err != nil {
			t.Fatalf("failed to decr: %v", err)
		}
		if val != 10 {
			t.Errorf("expected 10, got %d", val)
		}
	})

	// 测试 TTL
	t.Run("TTL", func(t *testing.T) {
		err := c.Set(ctx, "ttl_key", "value", 5*time.Second)
		if err != nil {
			t.Fatalf("failed to set cache: %v", err)
		}

		ttl, err := c.TTL(ctx, "ttl_key")
		if err != nil {
			t.Fatalf("failed to get ttl: %v", err)
		}
		if ttl <= 0 || ttl > 5*time.Second {
			t.Errorf("unexpected ttl: %v", ttl)
		}
	})

	// 测试 MSet/MGet
	t.Run("MSet/MGet", func(t *testing.T) {
		items := map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		err := c.MSet(ctx, items, 10*time.Minute)
		if err != nil {
			t.Fatalf("failed to mset: %v", err)
		}

		var values []string
		err = c.MGet(ctx, []string{"key1", "key2", "key3"}, &values)
		if err != nil {
			t.Fatalf("failed to mget: %v", err)
		}

		if len(values) != 3 {
			t.Errorf("expected 3 values, got %d", len(values))
		}
	})
}

// TestRemember 测试 Remember 模式
func TestRemember(t *testing.T) {
	ctx := context.Background()

	c, err := NewWithOptions(WithMemory(DefaultMemoryConfig()))
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "computed_value", nil
	}

	// 第一次调用，应该执行函数
	val, err := Remember(ctx, c, "remember_key", 10*time.Minute, fn)
	if err != nil {
		t.Fatalf("failed to remember: %v", err)
	}
	if val != "computed_value" {
		t.Errorf("expected 'computed_value', got %s", val)
	}
	if callCount != 1 {
		t.Errorf("expected fn to be called once, got %d", callCount)
	}

	// 第二次调用，应该从缓存获取
	val, err = Remember(ctx, c, "remember_key", 10*time.Minute, fn)
	if err != nil {
		t.Fatalf("failed to remember: %v", err)
	}
	if val != "computed_value" {
		t.Errorf("expected 'computed_value', got %s", val)
	}
	if callCount != 1 {
		t.Errorf("expected fn to be called once, got %d", callCount)
	}
}

// TestGetTyped 测试泛型 API
func TestGetTyped(t *testing.T) {
	ctx := context.Background()

	c, err := NewWithOptions(WithMemory(DefaultMemoryConfig()))
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	type User struct {
		ID   int64
		Name string
	}

	user := User{ID: 456, Name: "Bob"}
	err = SetTyped(ctx, c, "typed_user", user, 10*time.Minute)
	if err != nil {
		t.Fatalf("failed to set typed: %v", err)
	}

	cachedUser, err := GetTyped[User](ctx, c, "typed_user")
	if err != nil {
		t.Fatalf("failed to get typed: %v", err)
	}

	if cachedUser.ID != user.ID || cachedUser.Name != user.Name {
		t.Errorf("cached user mismatch: got %+v, want %+v", cachedUser, user)
	}
}
