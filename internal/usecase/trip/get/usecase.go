package get

import (
	"context"

	"github.com/kokos46/tripgo/internal/model"
)

type UseCase struct{ repository tripRepository }

func New(repository tripRepository) *UseCase { return &UseCase{repository: repository} }
func (u *UseCase) Execute(ctx context.Context, id string) (model.Trip, error) {
	return u.repository.Get(ctx, id)
}
