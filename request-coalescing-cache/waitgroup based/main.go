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
	wg  sync.WaitGroup
	val string
	err error
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

func (c *Coalescer) Get(key string) (string, error) {
	c.mu.Lock()

	// 1. Return the cached value when it is still fresh.
	if cached, exists := c.cache[key]; exists &&
		time.Now().Before(cached.expiresAt) {

		c.mu.Unlock()
		return cached.val, nil
	}

	// 2. Another goroutine is already fetching this key.
	if currentCall, exists := c.inflight[key]; exists {
		c.mu.Unlock()

		currentCall.wg.Wait()

		return currentCall.val, currentCall.err
	}

	// 3. This goroutine becomes the leader for this key.
	currentCall := &call{}
	currentCall.wg.Add(1)

	c.inflight[key] = currentCall
	c.mu.Unlock()

	// Slow fetch happens outside the mutex.
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

	currentCall.wg.Done()

	c.mu.Unlock()

	return val, err
}

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

/*
QUESTIONS AND ANSWERS — WAITGROUP VERSION

1. Why define FetchFunc?

	type FetchFunc func(key string) (string, error)

FetchFunc is a named function type.

Any function matching this signature can be passed to NewCoalescer:

	func(string) (string, error)

It is not compulsory, but it improves readability and reuse.


2. Why use *Coalescer instead of Coalescer?

	func (c *Coalescer) Get(key string) (string, error)

All goroutines must use the same:

	- mutex
	- cache map
	- inflight map

A value receiver would copy the Coalescer, including sync.Mutex.

Synchronization primitives should not be copied after use.


3. Why use &call?

	currentCall := &call{}

&call{} creates one call object and returns its address.

The inflight map stores:

	map[string]*call

This ensures the leader and followers share the same:

	- WaitGroup
	- val
	- err


4. What replaces the channel?

The channel version uses:

	done chan struct{}

The WaitGroup version uses:

	wg sync.WaitGroup

Followers wait with:

	currentCall.wg.Wait()

The leader releases them with:

	currentCall.wg.Done()


5. Why call Add(1)?

	currentCall.wg.Add(1)

The WaitGroup counter represents one unfinished fetch.

After Add(1):

	counter = 1

Followers calling Wait() block while the counter is greater than zero.


6. What happens when Done() is called?

	currentCall.wg.Done()

Done decreases the counter by one:

	1 -> 0

When the counter reaches zero, every goroutine blocked in Wait() wakes up.


7. Why must Add(1) happen before adding currentCall to inflight?

Correct order:

	currentCall := &call{}
	currentCall.wg.Add(1)
	c.inflight[key] = currentCall

If currentCall were added to inflight before Add(1), another goroutine
could find it and call Wait() while the counter was still zero.

Wait() would return immediately even though the fetch had not finished.


8. Why store val and err before calling Done()?

Correct order:

	currentCall.val = val
	currentCall.err = err
	currentCall.wg.Done()

Done() releases all followers.

Followers then read:

	currentCall.val
	currentCall.err

Therefore, the result must be written before Done() is called.


9. Why use *call in the WaitGroup version?

sync.WaitGroup must not be copied after use.

Using:

	map[string]*call

ensures every goroutine waits on the same WaitGroup object.

It also ensures they share the same val and err fields.


10. Why are there two different WaitGroups?

The call struct contains:

	currentCall.wg

This WaitGroup waits for one upstream fetch for one key.

main contains:

	var wg sync.WaitGroup

This WaitGroup waits for all 250 request goroutines to finish.

They solve different synchronization problems.


11. What are the possible values of v?

The fetch function returns:

	return "value:" + key, nil

Therefore:

	key = "a" -> v = "value:a"
	key = "b" -> v = "value:b"


12. Why does seen need a mutex?

All 250 goroutines execute:

	seen[v] = true

Go maps are not safe for concurrent writes, even when goroutines write the
same key and same value.

Without the mutex, the program may fail with:

	fatal error: concurrent map writes


13. Why pass &calls?

	atomic.AddInt64(&calls, 1)

&calls passes the address of the original int64 variable.

atomic.AddInt64 needs a *int64 so it can safely modify the original value
across concurrent goroutines.


14. Why execute fetch outside c.mu?

The fetch takes around 400 milliseconds.

Executing it outside the lock allows different keys to fetch concurrently.

Therefore:

	fetch("a")
	fetch("b")

can run at the same time.


15. Why put currentCall into inflight before unlocking?

	c.inflight[key] = currentCall
	c.mu.Unlock()

This ensures any later goroutine requesting the same key finds the active
call and waits on it instead of starting another fetch.


16. Why delete the key from inflight?

	delete(c.inflight, key)

After the fetch completes, it is no longer active.

A future request should use the cache or start a new fetch after expiry.


17. Why are there only two upstream calls?

There are two unique keys:

	"a"
	"b"

Request coalescing runs one fetch per key.

Therefore:

	upstream calls = 2


18. Why is elapsed time around 400 ms?

The fetch for "a" and the fetch for "b" execute concurrently.

Each takes around 400 ms.

Therefore, total time is approximately 400 ms rather than 800 ms.


19. Why does the cache hit not increase the call count?

The cache TTL is five seconds.

The final Get("a") happens before the cached value expires.

Therefore, Get returns the cached value without calling fetch again.


20. Channel versus WaitGroup

Channel version:

	Follower waits:
		<-currentCall.done

	Leader releases:
		close(currentCall.done)

WaitGroup version:

	Follower waits:
		currentCall.wg.Wait()

	Leader releases:
		currentCall.wg.Done()

Both versions release all followers and store the actual result in:

	currentCall.val
	currentCall.err
*/
