package create

import (
	"context"

	"github.com/kokos46/tripgo/internal/model"
)

//go:generate go run github.com/golang/mock/mockgen -source contract.go -destination mocks_test.go -package create

type tripRepository interface {
	Create(context.Context, model.Trip) error
	CreateStatusHistory(context.Context, string, string, string) error
}
type txManager interface {
	Do(context.Context, func(context.Context) error) error
}
