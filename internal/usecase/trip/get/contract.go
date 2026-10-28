package get

import (
	"context"
	"github.com/kokos46/tripgo/internal/model"
)

//go:generate go run github.com/golang/mock/mockgen -source contract.go -destination mocks_test.go -package get

type tripRepository interface {
	Get(context.Context, string) (model.Trip, error)
}
