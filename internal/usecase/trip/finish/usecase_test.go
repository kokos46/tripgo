package finish

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kokos46/tripgo/internal/model"
)

type fakeTx struct {
	calls int
}

func (tx *fakeTx) Do(ctx context.Context, fn func(context.Context) error) error {
	tx.calls++
	return fn(ctx)
}

type fakeRepository struct {
	trip         model.Trip
	historyCalls int
	historyErr   error
}

func (r *fakeRepository) Finish(context.Context, string, time.Time) (model.Trip, error) {
	return r.trip, nil
}

func (r *fakeRepository) CreateStatusHistory(context.Context, string, string, string) error {
	r.historyCalls++
	return r.historyErr
}

func TestExecuteFinishesTripAndWritesHistory(t *testing.T) {
	tx := &fakeTx{}
	repo := &fakeRepository{trip: model.Trip{ID: "trip-id", Status: "completed"}}
	got, err := New(tx, repo).Execute(context.Background(), "trip-id", time.Now())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ID != "trip-id" || got.Status != "completed" || tx.calls != 1 || repo.historyCalls != 1 {
		t.Fatalf("unexpected result: trip=%+v tx=%d history=%d", got, tx.calls, repo.historyCalls)
	}
}

func TestExecuteReturnsHistoryError(t *testing.T) {
	wantErr := errors.New("history insert failed")
	tx := &fakeTx{}
	repo := &fakeRepository{trip: model.Trip{ID: "trip-id"}, historyErr: wantErr}
	_, err := New(tx, repo).Execute(context.Background(), "trip-id", time.Now())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
