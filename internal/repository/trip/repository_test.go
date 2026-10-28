package trip

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsDriverBusy(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "active driver unique constraint",
			err:  &pgconn.PgError{Code: "23505", ConstraintName: "trips_driver_active_idx"},
			want: true,
		},
		{
			name: "other unique constraint",
			err:  &pgconn.PgError{Code: "23505", ConstraintName: "trips_pkey"},
			want: false,
		},
		{
			name: "other PostgreSQL error",
			err:  &pgconn.PgError{Code: "23503", ConstraintName: "trips_driver_active_idx"},
			want: false,
		},
		{name: "non PostgreSQL error", err: errTest, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isDriverBusy(test.err); got != test.want {
				t.Fatalf("isDriverBusy() = %v, want %v", got, test.want)
			}
		})
	}
}

var errTest = testError("test")

type testError string

func (e testError) Error() string { return string(e) }
