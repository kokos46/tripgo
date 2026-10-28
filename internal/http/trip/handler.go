package trip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/kokos46/tripgo/internal/domain/trip"
	api "github.com/kokos46/tripgo/internal/generated"
	"github.com/kokos46/tripgo/internal/http/domainerror"
	"github.com/kokos46/tripgo/internal/model"
	"github.com/kokos46/tripgo/internal/usecase/trip/create"
	"github.com/kokos46/tripgo/internal/usecase/trip/finish"
	"github.com/kokos46/tripgo/internal/usecase/trip/get"
)

type Handler struct {
	api.Unimplemented
	create *create.UseCase
	get    *get.UseCase
	finish *finish.UseCase
	ping   func() error
}

func New(createUseCase *create.UseCase, getUseCase *get.UseCase, finishUseCase *finish.UseCase, ping func() error) *Handler {
	return &Handler{create: createUseCase, get: getUseCase, finish: finishUseCase, ping: ping}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
func (h *Handler) Ready(w http.ResponseWriter, _ *http.Request) {
	if h.ping == nil || h.ping() != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var body api.CreateTripJSONRequestBody
	if err := decodeCreateTrip(r, &body); err != nil {
		WriteProblem(w, r, &domaintrip.ValidationError{Field: "body", Reason: "is invalid"})
		return
	}
	if body.UserId == uuid.Nil || body.DriverId == uuid.Nil {
		WriteProblem(w, r, &domaintrip.ValidationError{Field: "user_id/driver_id", Reason: "must be a non-empty UUID"})
		return
	}
	value, err := h.create.Execute(r.Context(), model.Trip{UserID: body.UserId.String(), DriverID: body.DriverId.String(), StartPoint: model.Coordinates{Latitude: body.StartPoint.Latitude, Longitude: body.StartPoint.Longitude}, EndPoint: model.Coordinates{Latitude: body.EndPoint.Latitude, Longitude: body.EndPoint.Longitude}, Price: body.Price})
	if err != nil {
		WriteProblem(w, r, err)
		return
	}
	response, err := toAPITrip(value)
	if err != nil {
		WriteProblem(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+value.ID)
	writeJSON(w, http.StatusCreated, response)
}
func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	value, err := h.get.Execute(r.Context(), id.String())
	if err != nil {
		WriteProblem(w, r, err)
		return
	}
	response, err := toAPITrip(value)
	if err != nil {
		WriteProblem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	value, err := h.finish.Execute(r.Context(), id.String(), time.Now())
	if err != nil {
		WriteProblem(w, r, err)
		return
	}
	response, err := toAPITrip(value)
	if err != nil {
		WriteProblem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func decodeCreateTrip(r *http.Request, value *api.CreateTripJSONRequestBody) error {
	const maxRequestBody = 1 << 20
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody+1))
	if err != nil {
		return err
	}
	if len(data) > maxRequestBody {
		return errors.New("request body is too large")
	}
	var required struct {
		UserID     *json.RawMessage `json:"user_id"`
		DriverID   *json.RawMessage `json:"driver_id"`
		StartPoint *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"start_point"`
		EndPoint *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"end_point"`
		Price *int64 `json:"price"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&required); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}
		return err
	}
	if required.UserID == nil || required.DriverID == nil ||
		required.StartPoint == nil || required.StartPoint.Latitude == nil || required.StartPoint.Longitude == nil ||
		required.EndPoint == nil || required.EndPoint.Latitude == nil || required.EndPoint.Longitude == nil ||
		required.Price == nil {
		return errors.New("request is missing a required field")
	}
	typedDecoder := json.NewDecoder(bytes.NewReader(data))
	typedDecoder.DisallowUnknownFields()
	if err := typedDecoder.Decode(value); err != nil {
		return err
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func WriteProblem(w http.ResponseWriter, r *http.Request, err error) {
	status, body := domainerror.Problem(err)
	instance := r.URL.Path
	body.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func toAPITrip(value model.Trip) (tripResponse, error) {
	id, err := parseUUID(value.ID, "id")
	if err != nil {
		return tripResponse{}, err
	}
	userID, err := parseUUID(value.UserID, "user_id")
	if err != nil {
		return tripResponse{}, err
	}
	driverID, err := parseUUID(value.DriverID, "driver_id")
	if err != nil {
		return tripResponse{}, err
	}
	tripValue := api.Trip{Id: id, UserId: userID, DriverId: driverID, StartPoint: api.Coordinates{Latitude: value.StartPoint.Latitude, Longitude: value.StartPoint.Longitude}, EndPoint: api.Coordinates{Latitude: value.EndPoint.Latitude, Longitude: value.EndPoint.Longitude}, Price: value.Price, Status: api.TripStatus(value.Status), StartedAt: value.StartedAt, FinishedAt: value.FinishedAt, LastPositionAt: value.LastPositionAt}
	return tripResponse{Trip: tripValue, FinishedAt: tripValue.FinishedAt, LastPositionAt: tripValue.LastPositionAt}, nil
}

type tripResponse struct {
	api.Trip
	FinishedAt     *time.Time `json:"finished_at"`
	LastPositionAt *time.Time `json:"last_position_at"`
}

func parseUUID(value, field string) (api.TripId, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return api.TripId{}, fmt.Errorf("invalid %s in stored trip: %w", field, err)
	}
	return api.TripId(parsed), nil
}
