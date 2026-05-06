package service

import (
	"container/list"
	"io"
	"net/http"
	"sync"
	"time"
)

// GlobalTSCache is the global TS segment cache instance.
var GlobalTSCache = NewTSCache(200*1024*1024, 30*time.Second)

type tsCacheItem struct {
	key    string
	mutex  sync.RWMutex
	waitCh chan struct{}

	chunks [][]byte
	bytes  int64
	err    error

	expireAt time.Time
	element  *list.Element
	closed   bool
}

type TSCache struct {
	mu sync.RWMutex

	maxBytes int64
	curBytes int64

	ttl time.Duration

	ll    *list.List
	items map[string]*tsCacheItem

	cleanupDone chan struct{}
}

func NewTSCache(maxBytes int64, ttl time.Duration) *TSCache {
	c := &TSCache{
		maxBytes:    maxBytes,
		ttl:         ttl,
		ll:          list.New(),
		items:       make(map[string]*tsCacheItem),
		cleanupDone: make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

func (c *TSCache) GetOrCreate(key string) (*tsCacheItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if it, ok := c.items[key]; ok {
		if time.Now().After(it.expireAt) {
			c.removeItem(it)
			return c.createItem(key), true
		}
		it.expireAt = time.Now().Add(c.ttl)
		c.ll.MoveToFront(it.element)
		return it, false
	}

	return c.createItem(key), true
}

func (c *TSCache) createItem(key string) *tsCacheItem {
	it := &tsCacheItem{
		key:      key,
		waitCh:   make(chan struct{}, 1),
		expireAt: time.Now().Add(c.ttl),
	}
	it.element = c.ll.PushFront(it)
	c.items[key] = it
	return it
}

func (c *TSCache) WriteChunk(item *tsCacheItem, data []byte) {
	if data == nil || item == nil {
		return
	}

	item.mutex.Lock()
	if item.closed {
		item.mutex.Unlock()
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	item.chunks = append(item.chunks, cp)
	item.bytes += int64(len(cp))
	item.mutex.Unlock()

	// 通知等待的读取者
	select {
	case item.waitCh <- struct{}{}:
	default:
	}

	// 更新字节计数并淘汰旧项
	c.mu.Lock()
	c.curBytes += int64(len(data))
	for c.curBytes > c.maxBytes && c.ll.Back() != nil {
		least := c.ll.Back()
		if least == nil {
			break
		}
		c.removeItem(least.Value.(*tsCacheItem))
	}
	c.mu.Unlock()
}

func (item *tsCacheItem) ReadAll(dst io.Writer, done <-chan struct{}) error {
	seq := 1
	for {
		item.mutex.RLock()
		if seq <= len(item.chunks) {
			data := item.chunks[seq-1]
			item.mutex.RUnlock()
			if len(data) > 0 {
				if _, err := dst.Write(data); err != nil {
					return err
				}
				if f, ok := dst.(http.Flusher); ok {
					select {
					case <-done:
						return nil
					default:
						f.Flush()
					}
				}
			}
			seq++
			continue
		}
		closed := item.closed
		retErr := item.err
		item.mutex.RUnlock()

		if closed {
			return retErr
		}

		select {
		case <-item.waitCh:
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

func (item *tsCacheItem) Seal(err error) {
	item.mutex.Lock()
	if item.closed {
		item.mutex.Unlock()
		return
	}
	item.closed = true
	item.err = err
	ch := item.waitCh
	item.mutex.Unlock()

	safeCloseChan(ch)
}

func safeCloseChan(ch chan struct{}) {
	if ch == nil {
		return
	}
	defer func() { recover() }()
	close(ch)
}

func (c *TSCache) removeItem(it *tsCacheItem) {
	delete(c.items, it.key)
	c.ll.Remove(it.element)
	c.curBytes -= it.bytes
	it.Seal(nil)
}

func (c *TSCache) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for e := c.ll.Back(); e != nil; {
				it := e.Value.(*tsCacheItem)
				next := e.Prev()
				if now.After(it.expireAt) {
					c.removeItem(it)
				}
				e = next
			}
			c.mu.Unlock()
		case <-c.cleanupDone:
			return
		}
	}
}

func (c *TSCache) UpdateMaxBytes(newMax int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxBytes = newMax
	for c.curBytes > c.maxBytes && c.ll.Back() != nil {
		least := c.ll.Back()
		if least == nil {
			break
		}
		c.removeItem(least.Value.(*tsCacheItem))
	}
}

func (c *TSCache) Close() {
	close(c.cleanupDone)
	c.mu.Lock()
	defer c.mu.Unlock()
	for e := c.ll.Front(); e != nil; {
		next := e.Next()
		item := e.Value.(*tsCacheItem)
		item.Seal(nil)
		delete(c.items, item.key)
		c.ll.Remove(e)
		e = next
	}
	c.curBytes = 0
}

func (item *tsCacheItem) IsSealed() bool {
	item.mutex.RLock()
	defer item.mutex.RUnlock()
	return item.closed
}

func (c *TSCache) CurBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.curBytes
}

func (c *TSCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if it, ok := c.items[key]; ok {
		c.removeItem(it)
	}
}
