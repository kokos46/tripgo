package trip

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	domaintrip "github.com/kokos46/tripgo/internal/domain/trip"
	api "github.com/kokos46/tripgo/internal/generated"
	"github.com/kokos46/tripgo/internal/model"
	"github.com/kokos46/tripgo/internal/usecase/trip/create"
	"github.com/kokos46/tripgo/internal/usecase/trip/finish"
	"github.com/kokos46/tripgo/internal/usecase/trip/get"
)

type handlerTx struct{}

func (handlerTx) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type handlerRepository struct{ value model.Trip }

func (r *handlerRepository) Create(_ context.Context, value model.Trip) error {
	r.value = value
	return nil
}

func (*handlerRepository) CreateStatusHistory(context.Context, string, string, string) error {
	return nil
}

func (r *handlerRepository) Get(context.Context, string) (model.Trip, error) {
	return r.value, nil
}

func (r *handlerRepository) Finish(_ context.Context, _ string, at time.Time) (model.Trip, error) {
	r.value.Status = domaintrip.StatusCompleted
	r.value.FinishedAt = &at
	return r.value, nil
}

func TestCreateTripReturnsTripAndLocation(t *testing.T) {
	repo := &handlerRepository{}
	handler := New(create.New(handlerTx{}, repo), get.New(repo), finish.New(handlerTx{}, repo), nil)
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", recorder.Header().Get("Content-Type"))
	}
	if got := recorder.Header().Get("Location"); got != "/api/v1/trips/"+repo.value.ID {
		t.Fatalf("Location = %q", got)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, field := range []string{"id", "user_id", "driver_id", "start_point", "end_point", "price", "status", "started_at", "finished_at", "last_position_at"} {
		if _, ok := response[field]; !ok {
			t.Errorf("response missing %q", field)
		}
	}
	if string(response["finished_at"]) != "null" || string(response["last_position_at"]) != "null" {
		t.Fatalf("nullable fields = finished_at %s, last_position_at %s", response["finished_at"], response["last_position_at"])
	}
}

func TestInvalidCreateBodyReturnsProblemJSON(t *testing.T) {
	repo := &handlerRepository{}
	handler := New(create.New(handlerTx{}, repo), get.New(repo), finish.New(handlerTx{}, repo), nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(`{"unknown":true}`))
	recorder := httptest.NewRecorder()

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var problem api.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "invalid_request" || problem.Instance == nil || *problem.Instance != "/api/v1/trips" {
		t.Fatalf("unexpected problem: %+v", problem)
	}
}

func TestMissingRequiredCreateFieldReturnsProblemJSON(t *testing.T) {
	repo := &handlerRepository{}
	handler := New(create.New(handlerTx{}, repo), get.New(repo), finish.New(handlerTx{}, repo), nil)
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusBadRequest || recorder.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status/content type = %d/%q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
}

func TestGeneratedInvalidPathReturnsProblemJSON(t *testing.T) {
	repo := &handlerRepository{}
	handler := New(create.New(handlerTx{}, repo), get.New(repo), finish.New(handlerTx{}, repo), nil)
	server := api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter: chi.NewRouter(),
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			WriteProblem(w, r, &domaintrip.ValidationError{Field: "tripId", Reason: "is invalid"})
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest || recorder.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status/content type = %d/%q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
}

func TestReadyRequiresDatabasePing(t *testing.T) {
	tests := []struct {
		name string
		ping func() error
		want int
	}{
		{name: "ready", ping: func() error { return nil }, want: http.StatusOK},
		{name: "database unavailable", ping: func() error { return errors.New("db down") }, want: http.StatusServiceUnavailable},
		{name: "missing ping dependency", ping: nil, want: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerRepository{}
			handler := New(create.New(handlerTx{}, repo), get.New(repo), finish.New(handlerTx{}, repo), test.ping)
			request := httptest.NewRequest(http.MethodGet, "/ready", nil)
			recorder := httptest.NewRecorder()

			handler.Ready(recorder, request)

			if recorder.Code != test.want || recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("status/content type = %d/%q, want %d/application/json", recorder.Code, recorder.Header().Get("Content-Type"), test.want)
			}
		})
	}
}
