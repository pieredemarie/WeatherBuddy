package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"scheduler-service/internal/model"

	"github.com/jackc/pgx/v5/pgtype"
)

type SchedulerRepo struct {
	DB *sql.DB
}

func NewSchedulerRepo(db *sql.DB) *SchedulerRepo {
	return &SchedulerRepo{DB: db}
}

func (r *SchedulerRepo) DueUsers(ctx context.Context) ([]model.User, error) {
	const query = `
		SELECT
			u.id, u.contact_type, u.contact_value, u.city,
			u.latitude, u.longitude, u.timezone, u.notify_time,
			u.is_active, u.created_at
		FROM users u
		WHERE u.is_active
		  AND to_char(now() AT TIME ZONE u.timezone, 'HH24:MI') = to_char(u.notify_time, 'HH24:MI')
		  AND NOT EXISTS (
		      SELECT 1 FROM sent_log sl
		      WHERE sl.user_id = u.id
		        AND sl.sent_date = (now() AT TIME ZONE u.timezone)::date
		  )
	`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query due users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var (
			u           model.User
			contactType string
			notifyTime  pgtype.Time
		)
		if err := rows.Scan(
			&u.ID, &contactType, &u.ContactValue, &u.City,
			&u.Latitude, &u.Longitude, &u.Timezone, &notifyTime,
			&u.Active, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}

		u.ContactType = model.ContactType(contactType)
		u.NotifyTime = fromPGTime(notifyTime)
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due users: %w", err)
	}

	return users, nil
}
