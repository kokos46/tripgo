package finish

import (
	"context"
	"time"

	"github.com/kokos46/tripgo/internal/model"
)

type UseCase struct {
	tx         txManager
	repository tripRepository
}

func New(tx txManager, repository tripRepository) *UseCase {
	return &UseCase{tx: tx, repository: repository}
}
func (u *UseCase) Execute(ctx context.Context, id string, at time.Time) (model.Trip, error) {
	var value model.Trip
	err := u.tx.Do(ctx, func(txCtx context.Context) error {
		updated, err := u.repository.Finish(txCtx, id, at)
		if err != nil {
			return err
		}
		value = updated
		return u.repository.CreateStatusHistory(txCtx, id, "active", "completed")
	})
	if err != nil {
		return model.Trip{}, err
	}
	return value, nil
}
