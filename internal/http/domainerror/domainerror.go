package domainerror

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kokos46/tripgo/internal/domain/trip"
	api "github.com/kokos46/tripgo/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	status, code, title := http.StatusInternalServerError, "internal_error", "Internal Server Error"
	var validation *trip.ValidationError
	switch {
	case errors.Is(err, trip.ErrTripNotFound):
		status, code, title = http.StatusNotFound, "trip_not_found", "Trip Not Found"
	case errors.Is(err, trip.ErrTripCompleted):
		status, code, title = http.StatusConflict, "trip_completed", "Trip Already Completed"
	case errors.Is(err, trip.ErrDriverBusy):
		status, code, title = http.StatusConflict, "driver_busy", "Driver Busy"
	case errors.As(err, &validation):
		status, code, title = http.StatusBadRequest, "invalid_request", "Invalid Request"
	}
	typeURI := "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-")
	return status, api.Problem{Status: int32(status), Code: code, Title: title, Type: typeURI}
}
