package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Cache
type Cache struct {
	m   sync.Map
	ttl time.Duration
}

func (c *Cache) Get(key any) (any, error) {
	v, ok := c.m.Load(key)
	if ok && !v.(*item).expired() {
		return v.(*item).data, nil
	}
	return nil, errors.New(fmt.Sprintf("Cache not found: %v", key))
}

func (c *Cache) Set(key any, value any) {
	c.m.Store(key, &item{
		data:     value,
		expireAt: time.Now().Add(c.ttl),
	})
}

func (c *Cache) Remove(key any) {
	c.m.Delete(key)
}

// item
type item struct {
	data     any
	expireAt time.Time
}

func (item *item) expired() bool {
	return item.expireAt.Before(time.Now())
}
