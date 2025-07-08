// cmd/macbat/performance/optimizations.go
package main

import (
	"sync"
	"time"
)

// Cache представляет простой кэш для часто используемых данных
type Cache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

// CacheItem представляет элемент кэша с временем истечения
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// NewCache создает новый кэш
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
	}
}

// Set добавляет элемент в кэш с TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Get получает элемент из кэша
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Проверяем, не истек ли TTL
	if time.Now().After(item.Expiration) {
		// Удаляем истекший элемент (в отдельной горутине для неблокирующего удаления)
		go func() {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}()
		return nil, false
	}

	return item.Value, true
}

// LazyInitializer обеспечивает ленивую инициализацию ресурсов
type LazyInitializer struct {
	once     sync.Once
	initFunc func() (interface{}, error)
	value    interface{}
	err      error
}

// NewLazyInitializer создает новый lazy initializer
func NewLazyInitializer(initFunc func() (interface{}, error)) *LazyInitializer {
	return &LazyInitializer{
		initFunc: initFunc,
	}
}

// Get получает значение, инициализируя его при первом обращении
func (l *LazyInitializer) Get() (interface{}, error) {
	l.once.Do(func() {
		l.value, l.err = l.initFunc()
	})
	return l.value, l.err
}

// ConnectionPool представляет пул соединений для переиспользования
type ConnectionPool struct {
	mu          sync.Mutex
	connections chan interface{}
	factory     func() (interface{}, error)
	cleanup     func(interface{}) error
}

// NewConnectionPool создает новый пул соединений
func NewConnectionPool(size int, factory func() (interface{}, error), cleanup func(interface{}) error) *ConnectionPool {
	return &ConnectionPool{
		connections: make(chan interface{}, size),
		factory:     factory,
		cleanup:     cleanup,
	}
}

// Get получает соединение из пула или создает новое
func (p *ConnectionPool) Get() (interface{}, error) {
	select {
	case conn := <-p.connections:
		return conn, nil
	default:
		return p.factory()
	}
}

// Put возвращает соединение в пул
func (p *ConnectionPool) Put(conn interface{}) {
	select {
	case p.connections <- conn:
	default:
		// Пул полон, закрываем соединение
		if p.cleanup != nil {
			p.cleanup(conn)
		}
	}
}

// Close закрывает все соединения в пуле
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.connections)

	for conn := range p.connections {
		if p.cleanup != nil {
			if err := p.cleanup(conn); err != nil {
				return err
			}
		}
	}

	return nil
}
