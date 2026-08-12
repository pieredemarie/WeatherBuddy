package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"registration-service/internal/model"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// PostgresRepo — репозиторий регистраций поверх таблицы users.
type PostgresRepo struct {
	DB *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{DB: db}
}

func (r *PostgresRepo) Save(ctx context.Context, u model.User) error {
	const query = `
		INSERT INTO users (contact_type, contact_value, city, latitude, longitude, timezone, notify_time, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		ON CONFLICT (contact_type, contact_value)
		DO UPDATE SET
			city        = EXCLUDED.city,
			latitude    = EXCLUDED.latitude,
			longitude   = EXCLUDED.longitude,
			timezone    = EXCLUDED.timezone,
			notify_time = EXCLUDED.notify_time,
			is_active   = true
	`

	_, err := r.DB.ExecContext(ctx, query,
		string(u.ContactType),
		u.ContactValue,
		u.City,
		u.Latitude,
		u.Longitude,
		u.Timezone,
		toPGTime(u.NotifyTime),
	)
	if err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func toPGTime(t time.Time) pgtype.Time {
	micros := int64(t.Hour())*3600e6 + int64(t.Minute())*60e6 + int64(t.Second())*1e6
	return pgtype.Time{Microseconds: micros, Valid: true}
}

func fromPGTime(t pgtype.Time) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration(t.Microseconds) * time.Microsecond)
}
