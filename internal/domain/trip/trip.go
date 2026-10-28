package trip

import (
	"errors"
	"time"

	"github.com/kokos46/tripgo/internal/model"
)

const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip already completed")
	ErrDriverBusy    = errors.New("driver already has an active trip")
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Reason }

func ValidateCreate(t model.Trip) error {
	if t.UserID == "" {
		return &ValidationError{Field: "user_id", Reason: "is required"}
	}
	if t.DriverID == "" {
		return &ValidationError{Field: "driver_id", Reason: "is required"}
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Reason: "must be non-negative"}
	}
	if t.StartPoint.Latitude < -90 || t.StartPoint.Latitude > 90 {
		return &ValidationError{Field: "start_point.latitude", Reason: "must be between -90 and 90"}
	}
	if t.StartPoint.Longitude < -180 || t.StartPoint.Longitude > 180 {
		return &ValidationError{Field: "start_point.longitude", Reason: "must be between -180 and 180"}
	}
	if t.EndPoint.Latitude < -90 || t.EndPoint.Latitude > 90 {
		return &ValidationError{Field: "end_point.latitude", Reason: "must be between -90 and 90"}
	}
	if t.EndPoint.Longitude < -180 || t.EndPoint.Longitude > 180 {
		return &ValidationError{Field: "end_point.longitude", Reason: "must be between -180 and 180"}
	}
	return nil
}

func Finish(t *model.Trip, at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}
