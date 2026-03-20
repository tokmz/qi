package openapi

import (
	"reflect"
	"sync"
)

type CacheKey struct {
	Type reflect.Type
	Mode AnalyzeMode
}

type Cache struct {
	mu      sync.RWMutex
	schemas map[CacheKey]*SchemaNode
	params  map[CacheKey][]ParameterSpec
}

func NewCache() *Cache {
	return &Cache{
		schemas: make(map[CacheKey]*SchemaNode),
		params:  make(map[CacheKey][]ParameterSpec),
	}
}

func (c *Cache) GetSchema(key CacheKey) (*SchemaNode, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.schemas[key]
	return v, ok
}

func (c *Cache) SetSchema(key CacheKey, node *SchemaNode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.schemas[key] = node
}

func (c *Cache) GetParams(key CacheKey) ([]ParameterSpec, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.params[key]
	if !ok {
		return nil, false
	}
	out := make([]ParameterSpec, len(v))
	copy(out, v)
	return out, true
}

func (c *Cache) SetParams(key CacheKey, params []ParameterSpec) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ParameterSpec, len(params))
	copy(out, params)
	c.params[key] = out
}
