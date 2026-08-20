package requestcoalescingcache

/*
QUESTIONS AND ANSWERS — CHANNEL VERSION

1. Why define FetchFunc?

	type FetchFunc func(key string) (string, error)

FetchFunc is a named function type.

Any function with this signature can be passed to NewCoalescer:

	func(string) (string, error)

Examples:

	func fetchFromAPI(key string) (string, error)
	func fetchFromDB(key string) (string, error)
	func mockFetch(key string) (string, error)

It is not mandatory. We could directly write:

	fetch func(string) (string, error)

The named type mainly improves readability and reuse.


2. Why use *Coalescer instead of Coalescer?

	func (c *Coalescer) Get(key string) (string, error)

*Coalescer means Get receives a pointer to the original Coalescer.

Therefore, all goroutines use the same:

	- mutex
	- cache map
	- inflight map
	- fetch function

Using a value receiver would copy the Coalescer, including sync.Mutex.

A mutex should not be copied after it starts being used.


3. Why use &call?

	currentCall := &call{
		done: make(chan struct{}),
	}

&call{} creates a call object and returns its address.

Therefore, currentCall has type:

	*call

The inflight map stores:

	map[string]*call

All goroutines requesting the same key share the exact same:

	- done channel
	- val
	- err


4. Why use chan struct{}?

	done chan struct{}

The channel is used only as a completion signal.

It does not need to carry any real data.

struct{} is preferred because it has no meaningful payload.

Other possible channel types are:

	chan bool
	chan int
	chan string

But their values are unnecessary here.


5. What happens at this line?

	<-currentCall.done

Possible conditions:

	1. The channel is open:
	   The goroutine blocks and waits.

	2. The leader closes the channel:
	   The goroutine wakes up.

	3. The channel is already closed:
	   The receive returns immediately.

It does not receive "value:a" or "value:b".

The actual result is read from:

	currentCall.val
	currentCall.err


6. What value is received from a closed chan struct{}?

The zero value of struct{} is:

	struct{}{}

But in this code, the value is discarded because it is not assigned:

	<-currentCall.done

This line is used only for waiting.


7. Why close the channel instead of sending a value?

	close(currentCall.done)

Closing the channel wakes all waiting goroutines.

Sending:

	currentCall.done <- struct{}{}

normally wakes only one receiver.

For example, if 199 goroutines are waiting for key "a", one send would
wake only one goroutine.

Closing the channel broadcasts completion to all waiters.


8. What are the possible values of v?

The fetch function returns:

	return "value:" + key, nil

Therefore:

	key = "a" -> v = "value:a"
	key = "b" -> v = "value:b"

The seen map eventually contains:

	map[string]bool{
		"value:a": true,
		"value:b": true,
	}


9. Why does seen need a mutex when there are only two possible values?

All 250 goroutines still execute:

	seen[v] = true

For example, 200 goroutines may execute:

	seen["value:a"] = true

Even though they write the same key and same value, every assignment is
still a map write.

Go maps are not safe for concurrent writes.

Without the mutex, the program may fail with:

	fatal error: concurrent map writes

The race detector may also report a data race:

	go run -race main.go


10. Why pass &calls to atomic.AddInt64?

	atomic.AddInt64(&calls, 1)

&calls means the memory address of calls.

Its type is:

	*int64

atomic.AddInt64 needs the address so it can safely modify the original
variable.

Conceptually, it performs:

	calls = calls + 1

but does so atomically across concurrent goroutines.


11. Why is fetch executed outside c.mu?

	val, err := c.fetch(key)

The upstream fetch takes around 400 milliseconds.

If c.mu remained locked during the fetch, another key could not start its
fetch.

Running fetch outside the mutex allows:

	fetch("a")
	fetch("b")

to execute concurrently.

Therefore, total time is around 400 ms instead of 800 ms.


12. Why is currentCall placed in inflight before unlocking?

	c.inflight[key] = currentCall
	c.mu.Unlock()

The leader must register the active fetch before releasing the mutex.

Otherwise, another goroutine could enter Get, fail to find the key in
inflight, and start another upstream request for the same key.


13. Why store val and err before closing done?

	currentCall.val = val
	currentCall.err = err
	close(currentCall.done)

Closing done wakes all followers.

Followers immediately read:

	currentCall.val
	currentCall.err

Therefore, the result must be stored before the channel is closed.


14. Why delete the key from inflight?

	delete(c.inflight, key)

Once the fetch completes, it is no longer in progress.

Future calls should either:

	- return the fresh cache value
	- or start a new fetch after the cache expires


15. Why are there only two upstream calls?

There are 200 requests for key "a" and 50 requests for key "b".

Request coalescing allows one fetch per key:

	one fetch for "a"
	one fetch for "b"

Therefore:

	upstream calls = 2


16. Why is the elapsed time around 400 ms?

The two upstream fetches for "a" and "b" run concurrently.

Each takes around 400 ms.

Therefore, the total elapsed time is approximately:

	400 ms

not:

	800 ms


17. Why does the final Get("a") not increase calls?

The cache TTL is five seconds:

	NewCoalescer(fetch, 5*time.Second)

The final Get("a") happens shortly after the first fetch, so the cached
entry is still fresh.

Therefore, it returns from cache and does not call fetch again.
*/
