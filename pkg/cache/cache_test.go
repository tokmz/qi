package cache

import (
	"context"
	"testing"
	"time"
)

func TestNew_MemoryDriver(t *testing.T) {
	c, err := New(&Config{
		Driver: DriverMemory,
		Memory: &MemoryConfig{MaxSize: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	c.Set(ctx, "k", "v", 0)
	var v string
	if err := c.Get(ctx, "k", &v); err != nil {
		t.Fatal(err)
	}
	if v != "v" {
		t.Fatalf("want v, got %s", v)
	}
}

func TestNew_NilConfig_UsesDefault(t *testing.T) {
	c, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", 1, 0); err != nil {
		t.Fatal(err)
	}
}

func TestNew_UnknownDriver(t *testing.T) {
	_, err := New(&Config{Driver: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}

func TestNew_RedisDriverMissingConfig(t *testing.T) {
	_, err := New(&Config{Driver: DriverRedis})
	if err == nil {
		t.Fatal("expected error when redis config is nil")
	}
}

func TestNew_MultiLevelDriverMissingRedis(t *testing.T) {
	_, err := New(&Config{
		Driver: DriverMultiLevel,
		Memory: &MemoryConfig{MaxSize: 10},
	})
	if err == nil {
		t.Fatal("expected error when redis config is nil for multilevel")
	}
}

func TestNew_WithPenetration(t *testing.T) {
	c, err := New(&Config{
		Driver: DriverMemory,
		Memory: &MemoryConfig{MaxSize: 100},
		Penetration: &PenetrationConfig{
			EnableBloom: true,
			BloomN:      1000,
			BloomFP:     0.01,
			NullTTL:     time.Minute,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 验证装饰器生效：未写入的 key 不触发 bloom（bloom 初始为空，TestString 返回 false）
	ctx := context.Background()
	var v string
	if err := c.Get(ctx, "unknown", &v); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestNew_WithTracing(t *testing.T) {
	c, err := New(&Config{
		Driver:         DriverMemory,
		Memory:         &MemoryConfig{MaxSize: 100},
		TracingEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// tracing 装饰器不影响正常操作
	ctx := context.Background()
	c.Set(ctx, "k", "v", 0)
	var v string
	if err := c.Get(ctx, "k", &v); err != nil {
		t.Fatal(err)
	}
}

// ===== Config setDefaults =====

func TestSetDefaults_Serializer(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()
	if cfg.Serializer == nil {
		t.Fatal("serializer should default to JSON")
	}
	if _, ok := cfg.Serializer.(JSONSerializer); !ok {
		t.Fatal("default serializer should be JSONSerializer")
	}
}

func TestSetDefaults_MemoryCleanupInterval(t *testing.T) {
	cfg := &Config{
		Memory: &MemoryConfig{},
	}
	cfg.setDefaults()
	if cfg.Memory.CleanupInterval != time.Minute {
		t.Fatalf("default CleanupInterval should be 1m, got %v", cfg.Memory.CleanupInterval)
	}
}

func TestSetDefaults_PenetrationDefaults(t *testing.T) {
	cfg := &Config{
		Penetration: &PenetrationConfig{
			EnableBloom: true,
		},
	}
	cfg.setDefaults()
	if cfg.Penetration.NullTTL != 60*time.Second {
		t.Fatalf("default NullTTL should be 60s, got %v", cfg.Penetration.NullTTL)
	}
	if cfg.Penetration.BloomN != 100_000 {
		t.Fatalf("default BloomN should be 100000, got %d", cfg.Penetration.BloomN)
	}
	if cfg.Penetration.BloomFP != 0.01 {
		t.Fatalf("default BloomFP should be 0.01, got %f", cfg.Penetration.BloomFP)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Driver != DriverMemory {
		t.Fatalf("default driver should be memory, got %s", cfg.Driver)
	}
	if cfg.Memory == nil {
		t.Fatal("default Memory config should not be nil")
	}
	if cfg.Memory.MaxSize != 10_000 {
		t.Fatalf("default MaxSize should be 10000, got %d", cfg.Memory.MaxSize)
	}
}
