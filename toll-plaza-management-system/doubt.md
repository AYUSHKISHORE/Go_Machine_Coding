
# Go Doubts — Pointers, Methods & Mutex

## 1. Why `*Struct`?

```go
type Booth struct {
    Fare int
}

b1 := Booth{Fare: 100}
b2 := b1          // Value copy
b2.Fare = 200

fmt.Println(b1.Fare) // 100

b3 := &b1         // Pointer to original
b3.Fare = 300

fmt.Println(b1.Fare) // 300
```

**Conclusion:** Value copies are independent. Pointers allow modification of the same object.

## 2. Why `(t *TollService)`?

```go
type TollService struct {
    Count int
}

func (t TollService) Way1() {
    t.Count++
}

func (t *TollService) Way2() {
    t.Count++
}
```

Example:

```go
ts := TollService{}

ts.Way1()
fmt.Println(ts.Count) // 0

ts.Way2()
fmt.Println(ts.Count) // 1
```

**Conclusion:** A value receiver copies the struct. A pointer receiver modifies the original.

### Special case: `Events []Event`

```go
func (t TollService) RecordEvent(e Event) {
    t.Events = append(t.Events, e)
}
```

This does not update the original slice length because the slice descriptor was copied.

Use `(t *TollService)` instead.

## 3. Why Mutex During Append?

Without mutex:

```go
go ts.RecordEvent(event1)
go ts.RecordEvent(event2)
```

Both goroutines may modify `ts.Events` simultaneously, causing a data race or lost events.

With mutex:

```go
func (t *TollService) RecordEvent(e Event) {
    t.mu.Lock()
    defer t.mu.Unlock()

    t.Events = append(t.Events, e)
}
```

Only one goroutine appends at a time.

**Remember:**
- Sequential calls: Mutex not needed if no other concurrent access.
- Concurrent shared writes: Synchronization needed.
- `[]*Event` does not prevent a race.
- A struct containing a used mutex must not be copied.
- Alternative: Use a channel with one event collector goroutine.
