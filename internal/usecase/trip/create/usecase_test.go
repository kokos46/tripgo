package create

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kokos46/tripgo/internal/model"
)

type fakeRepository struct {
	createCalls  int
	historyCalls int
	historyErr   error
}

func (r *fakeRepository) Create(context.Context, model.Trip) error { r.createCalls++; return nil }
func (r *fakeRepository) CreateStatusHistory(context.Context, string, string, string) error {
	r.historyCalls++
	return r.historyErr
}
func (*fakeRepository) Get(context.Context, string) (model.Trip, error) { return model.Trip{}, nil }
func (*fakeRepository) Finish(context.Context, string, time.Time) (model.Trip, error) {
	return model.Trip{}, nil
}

type fakeTx struct{ calls int }

func (tx *fakeTx) Do(ctx context.Context, fn func(context.Context) error) error {
	tx.calls++
	return fn(ctx)
}

func TestExecuteCreatesTripAndHistory(t *testing.T) {
	repository := &fakeRepository{}
	tx := &fakeTx{}
	value, err := New(tx, repository).Execute(context.Background(), model.Trip{UserID: "user", DriverID: "driver"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if value.ID == "" || value.Status != "active" || tx.calls != 1 || repository.createCalls != 1 || repository.historyCalls != 1 {
		t.Fatalf("unexpected calls: trip=%+v tx=%d create=%d history=%d", value, tx.calls, repository.createCalls, repository.historyCalls)
	}
}
func TestExecuteReturnsHistoryError(t *testing.T) {
	expected := errors.New("history failed")
	repository := &fakeRepository{historyErr: expected}
	_, err := New(&fakeTx{}, repository).Execute(context.Background(), model.Trip{UserID: "user", DriverID: "driver"})
	if !errors.Is(err, expected) {
		t.Fatalf("Execute() error = %v, want %v", err, expected)
	}
}
