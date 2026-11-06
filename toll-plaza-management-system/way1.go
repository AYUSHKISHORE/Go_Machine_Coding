package main

import (
	"fmt"
	"time"
)

// CreatePlaza adds a plaza with the specified number of booths.
func CreatePlaza(pid string, mBooth int, ts *TollService) *TollService {
	plz := &Plaza{
		Pid:           pid,
		DateBasedFare: make(map[int]int),
	}

	for i := 0; i < mBooth; i++ {
		booth := &Booth{
			Bid:           i + 1,
			Pid:           pid,
			DateBasedFare: make(map[int]int),
		}
		plz.Booths = append(plz.Booths, booth)
	}

	ts.Plazas = append(ts.Plazas, plz)
	return ts
}

func sampleVehicles() []Vehicle {
	return []Vehicle{
		{"v1", "car"},
		{"v2", "bike"},
		{"v3", "bus"},
		{"v4", "bike"},
		{"v5", "car"},
	}
}

// way1 is the original sequential implementation.
func way1() {
	fmt.Println("========== WAY 1: SEQUENTIAL ==========")

	ts := &TollService{}
	CreatePlaza("p1", 3, ts)

	ts.Initiate(sampleVehicles())
	ts.GenerateDailyReport(time.Now().Day())
}

// Initiate processes vehicles one by one.
func (t *TollService) Initiate(vehicles []Vehicle) {
	for i, vehicle := range vehicles {
		plaza := t.Plazas[i%len(t.Plazas)]
		booth := plaza.Booths[i%len(plaza.Booths)]

		optype := "entry"
		if i%2 != 0 {
			optype = "exit"
		}

		t.VehicleMovement(vehicle, plaza.Pid, booth.Bid, optype)
	}
}

// VehicleMovement is the original sequential version (no mutex).
func (t *TollService) VehicleMovement(vehicle Vehicle, pid string, bid int, optype string) {
	fare := calculateFare(vehicle, optype)
	evt := &Event{
		Vid:  vehicle.Vid,
		Pid:  pid,
		Bid:  bid,
		Type: optype,
		Time: time.Now(),
		Fare: fare,
	}

	// Store a pointer to the event; its Fare is already set.
	t.Events = append(t.Events, evt)
	date := evt.Time.Day()

	for _, plz := range t.Plazas {
		if plz.Pid != pid {
			continue
		}

		for _, bth := range plz.Booths {
			if bth.Bid == bid {
				bth.DateBasedFare[date] += fare
				break
			}
		}

		plz.DateBasedFare[date] += fare
		break
	}

	captureEvent(evt)
}

func captureEvent(evt *Event) {
	fmt.Printf("Event Recorded | vehicleId = %v | plazaId = %v | boothId = %v | movementType = %v | time = %v | fare = %v\n",
		evt.Vid, evt.Pid, evt.Bid, evt.Type, evt.Time.Format(time.RFC3339), evt.Fare)
}

func calculateFare(vehicle Vehicle, optype string) int {
	return basefares[vehicle.Type] + opFare[optype]
}

func (t *TollService) GenerateDailyReport(date int) {
	fmt.Println("\n========== DAILY REPORT ==========")
	for _, plz := range t.Plazas {
		for _, bth := range plz.Booths {
			fmt.Printf("Date = %v | Pid = %v | TotalPlazaFareCollected = %v | Bid = %v | TotalBoothFareCollected = %v\n",
				date, plz.Pid, plz.DateBasedFare[date], bth.Bid, bth.DateBasedFare[date])
		}
	}
	fmt.Println("Total Events:", len(t.Events))
	fmt.Println()
}
