package domainerror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/kokos46/tripgo/internal/domain/trip"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{{"not found", trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"}, {"completed", trip.ErrTripCompleted, http.StatusConflict, "trip_completed"}, {"busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy"}, {"validation", &trip.ValidationError{Field: "price"}, http.StatusBadRequest, "invalid_request"}, {"internal", errors.New("secret database error"), http.StatusInternalServerError, "internal_error"}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, body := Problem(test.err)
			if status != test.status || body.Code != test.code || body.Detail != nil {
				t.Fatalf("Problem() = %d, %+v", status, body)
			}
		})
	}
}
