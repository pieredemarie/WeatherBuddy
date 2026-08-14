package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func fromPGTime(t pgtype.Time) time.Time {
	if !t.Valid {
		return time.Time{}
	}

	return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(t.Microseconds) * time.Microsecond)
}
