package finish

import (
	"context"
	"github.com/kokos46/tripgo/internal/model"
	"time"
)

//go:generate go run github.com/golang/mock/mockgen -source contract.go -destination mocks_test.go -package finish

type tripRepository interface {
	Finish(context.Context, string, time.Time) (model.Trip, error)
	CreateStatusHistory(context.Context, string, string, string) error
}
type txManager interface {
	Do(context.Context, func(context.Context) error) error
}
