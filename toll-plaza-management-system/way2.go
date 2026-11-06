package main

import (
	"fmt"
	"sync"
	"time"
)

// Way 2: TollService with mutex
type ConcurrentTollService struct {
	*TollService
	mu sync.Mutex
}

func way2() {
	fmt.Println("========== WAY 2: GOROUTINES + MUTEX ==========")

	ts := &ConcurrentTollService{
		TollService: &TollService{},
	}

	CreatePlaza("p1", 3, ts.TollService)

	vehicles := []Vehicle{
		{"v1", "car"},
		{"v2", "bike"},
		{"v3", "bus"},
		{"v4", "bike"},
		{"v5", "car"},
	}

	ts.InitiateConcurrent(vehicles)

	ts.GenerateDailyReport(time.Now().Day())
}

// Process vehicles concurrently
func (t *ConcurrentTollService) InitiateConcurrent(
	vehicles []Vehicle,
) {
	var wg sync.WaitGroup

	for i, vehicle := range vehicles {
		plaza := t.Plazas[i%len(t.Plazas)]
		booth := plaza.Booths[i%len(plaza.Booths)]

		optype := "entry"
		if i%2 != 0 {
			optype = "exit"
		}

		wg.Add(1)

		go func(v Vehicle, pid string, bid int, op string) {
			defer wg.Done()

			t.VehicleMovement(v, pid, bid, op)

		}(vehicle, plaza.Pid, booth.Bid, optype)
	}

	wg.Wait()
}

// Thread-safe VehicleMovement
func (t *ConcurrentTollService) VehicleMovement(
	vehicle Vehicle,
	pid string,
	bid int,
	optype string,
) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Reuse VehicleMovement from Way 1
	t.TollService.VehicleMovement(
		vehicle,
		pid,
		bid,
		optype,
	)
}
