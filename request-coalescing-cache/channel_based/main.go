package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type FetchFunc func(key string) (string, error)

type entry struct {
	val       string
	expiresAt time.Time
}

type call struct {
	done chan struct{}
	val  string
	err  error
}

type Coalescer struct {
	ttl   time.Duration
	fetch FetchFunc

	mu       sync.Mutex
	cache    map[string]entry
	inflight map[string]*call
}

func NewCoalescer(fetch FetchFunc, ttl time.Duration) *Coalescer {
	return &Coalescer{
		fetch:    fetch,
		ttl:      ttl,
		cache:    make(map[string]entry),
		inflight: make(map[string]*call),
	}
}

// Get returns the value for key, serving from cache when fresh.
// Concurrent callers for the same key share a single fetch.
func (c *Coalescer) Get(key string) (string, error) {
	c.mu.Lock()

	// 1. Return immediately when the cached value is fresh.
	if cached, exists := c.cache[key]; exists &&
		time.Now().Before(cached.expiresAt) {

		c.mu.Unlock()
		return cached.val, nil
	}

	// 2. Another goroutine is already fetching this key.
	// Wait for its result instead of calling upstream again.
	if currentCall, exists := c.inflight[key]; exists {
		c.mu.Unlock()

		<-currentCall.done

		return currentCall.val, currentCall.err
	}

	// 3. This goroutine becomes the leader for this key.
	currentCall := &call{
		done: make(chan struct{}),
	}

	c.inflight[key] = currentCall
	c.mu.Unlock()

	// Important: never hold the global mutex during the slow fetch.
	// This allows key "a" and key "b" to fetch concurrently.
	val, err := c.fetch(key)

	c.mu.Lock()

	currentCall.val = val
	currentCall.err = err

	if err == nil {
		c.cache[key] = entry{
			val:       val,
			expiresAt: time.Now().Add(c.ttl),
		}
	}

	delete(c.inflight, key)

	// Closing the channel wakes every caller waiting on this fetch.
	close(currentCall.done)

	c.mu.Unlock()

	return val, err
}

// ---------- harness (don't edit) ----------

func main() {
	var calls int64

	fetch := func(key string) (string, error) {
		atomic.AddInt64(&calls, 1)
		time.Sleep(400 * time.Millisecond)
		return "value:" + key, nil
	}

	c := NewCoalescer(fetch, 5*time.Second)

	start := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[string]bool{}

	for i := 0; i < 250; i++ {
		key := "a"
		if i >= 200 {
			key = "b"
		}

		wg.Add(1)

		go func(k string) {
			defer wg.Done()

			v, err := c.Get(k)
			if err != nil {
				fmt.Println("err:", err)
				return
			}

			mu.Lock()
			seen[v] = true
			mu.Unlock()
		}(key)
	}

	wg.Wait()

	fmt.Println(
		"upstream calls:",
		atomic.LoadInt64(&calls),
		"(expect 2)",
	)

	fmt.Println(
		"distinct results:",
		len(seen),
		"(expect 2)",
	)

	fmt.Println(
		"elapsed:",
		time.Since(start).Round(10*time.Millisecond),
		"(expect ~400ms)",
	)

	if _, err := c.Get("a"); err != nil {
		fmt.Println("err:", err)
	}

	fmt.Println(
		"upstream calls after cache hit:",
		atomic.LoadInt64(&calls),
		"(expect 2)",
	)
}
