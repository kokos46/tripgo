package trip

import (
	"testing"
	"time"

	"github.com/kokos46/tripgo/internal/model"
)

func TestFinish(t *testing.T) {
	finishedAt := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		status  string
		wantErr error
	}{
		{name: "active", status: StatusActive},
		{name: "completed", status: StatusCompleted, wantErr: ErrTripCompleted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := model.Trip{Status: test.status}
			err := Finish(&value, finishedAt)
			if err != test.wantErr {
				t.Fatalf("Finish() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && (value.Status != StatusCompleted || value.FinishedAt == nil || !value.FinishedAt.Equal(finishedAt)) {
				t.Fatalf("trip was not completed: %+v", value)
			}
		})
	}
}

func TestValidateCreate(t *testing.T) {
	base := model.Trip{UserID: "user", DriverID: "driver", StartPoint: model.Coordinates{Latitude: 1, Longitude: 2}, EndPoint: model.Coordinates{Latitude: 3, Longitude: 4}, Price: 10}
	if err := ValidateCreate(base); err != nil {
		t.Fatalf("valid trip rejected: %v", err)
	}
	invalid := base
	invalid.Price = -1
	if err := ValidateCreate(invalid); err == nil {
		t.Fatal("negative price accepted")
	}
}
