package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kokos46/tripgo/internal/domain/trip"
	"github.com/kokos46/tripgo/internal/infra/postgres/transaction"
	"github.com/kokos46/tripgo/internal/model"
)

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	return &Repository{pool: pool, queryTimeout: queryTimeout}
}
func (r *Repository) executor(ctx context.Context) transaction.Executor {
	if executor, ok := transaction.ExecutorFromContext(ctx); ok {
		return executor
	}
	return poolExecutor{pool: r.pool}
}

type poolExecutor struct{ pool *pgxpool.Pool }

func (e poolExecutor) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := e.pool.Exec(ctx, sql, args...)
	return err
}

func (e poolExecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return e.pool.QueryRow(ctx, sql, args...)
}

func (r *Repository) Create(ctx context.Context, value model.Trip) error {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	if err := trip.ValidateCreate(value); err != nil {
		return err
	}
	query, args, err := squirrel.Insert("trips").Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").Values(value.ID, value.UserID, value.DriverID, value.StartPoint.Latitude, value.StartPoint.Longitude, value.EndPoint.Latitude, value.EndPoint.Longitude, value.Price, value.Status, value.StartedAt).PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build trip insert: %w", err)
	}
	if err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		if isDriverBusy(err) {
			return trip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}
func (r *Repository) CreateStatusHistory(ctx context.Context, tripID, fromStatus, toStatus string) error {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var from any
	if fromStatus != "" {
		from = fromStatus
	}
	query, args, err := squirrel.Insert("trip_status_history").Columns("trip_id", "from_status", "to_status").Values(tripID, from, toStatus).PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build history insert: %w", err)
	}
	if err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}
func (r *Repository) Get(ctx context.Context, id string) (model.Trip, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var value model.Trip
	var finishedAt *time.Time
	query, args, err := squirrel.Select(tripColumns).From("trips").Where(squirrel.Eq{"id": id}).PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build trip select: %w", err)
	}
	err = r.executor(ctx).QueryRow(ctx, query, args...).Scan(&value.ID, &value.UserID, &value.DriverID, &value.StartPoint.Latitude, &value.StartPoint.Longitude, &value.EndPoint.Latitude, &value.EndPoint.Longitude, &value.Price, &value.Status, &value.StartedAt, &finishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, trip.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	value.FinishedAt = finishedAt
	return value, nil
}
func (r *Repository) Finish(ctx context.Context, id string, at time.Time) (model.Trip, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var value model.Trip
	query, args, err := squirrel.Update("trips").
		Set("status", trip.StatusCompleted).
		Set("finished_at", at).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"status": trip.StatusActive}).
		Suffix("RETURNING " + tripColumns).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build trip update: %w", err)
	}
	err = r.executor(ctx).QueryRow(ctx, query, args...).Scan(&value.ID, &value.UserID, &value.DriverID, &value.StartPoint.Latitude, &value.StartPoint.Longitude, &value.EndPoint.Latitude, &value.EndPoint.Longitude, &value.Price, &value.Status, &value.StartedAt, &value.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := r.Get(ctx, id)
		if getErr != nil {
			return model.Trip{}, getErr
		}
		if existing.Status == trip.StatusCompleted {
			return model.Trip{}, trip.ErrTripCompleted
		}
		return model.Trip{}, trip.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return value, nil
}
func isDriverBusy(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_driver_active_idx"
}

func (r *Repository) withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.queryTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.queryTimeout)
}

const tripColumns = "id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at"
