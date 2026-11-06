package main

import "time"

type Vehicle struct {
	Vid  string
	Type string
}

type Booth struct {
	Bid           int
	Pid           string
	DateBasedFare map[int]int
}

type Plaza struct {
	Pid           string
	DateBasedFare map[int]int
	Booths        []*Booth
}

type TollService struct {
	Plazas []*Plaza
	Events []*Event
}

type Event struct {
	Vid  string
	Bid  int
	Pid  string
	Type string
	Time time.Time
	Fare int
}

var basefares = map[string]int{
	"car":  10,
	"bike": 5,
	"bus":  15,
}

var opFare = map[string]int{
	"entry": 10,
	"exit":  5,
}
