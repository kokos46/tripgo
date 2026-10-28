package model

import "time"

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             string
	UserID         string
	DriverID       string
	StartPoint     Coordinates
	EndPoint       Coordinates
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}
