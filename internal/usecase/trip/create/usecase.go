package create

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kokos46/tripgo/internal/domain/trip"
	"github.com/kokos46/tripgo/internal/model"
)

type UseCase struct {
	tx         txManager
	repository tripRepository
}

func New(tx txManager, repository tripRepository) *UseCase {
	return &UseCase{tx: tx, repository: repository}
}
func (u *UseCase) Execute(ctx context.Context, value model.Trip) (model.Trip, error) {
	if value.ID == "" {
		value.ID = uuid.NewString()
	}
	if value.StartedAt.IsZero() {
		value.StartedAt = time.Now()
	}
	value.Status = trip.StatusActive
	if err := trip.ValidateCreate(value); err != nil {
		return model.Trip{}, err
	}
	if err := u.tx.Do(ctx, func(txCtx context.Context) error {
		if err := u.repository.Create(txCtx, value); err != nil {
			return err
		}
		return u.repository.CreateStatusHistory(txCtx, value.ID, "", trip.StatusActive)
	}); err != nil {
		return model.Trip{}, err
	}
	return value, nil
}
