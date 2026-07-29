package repository

import (
	"context"
	"errors"
	"time"

	"github.com/whiterage/subscriptions-api/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	db *pgxpool.Pool
}

func NewPostgres(db *pgxpool.Pool) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) Create(ctx context.Context, input service.CreateSubscription) (service.Subscription, error) {
	const query = `
		insert into subscriptions (service_name, price, user_id, start_date, end_date)
		values ($1, $2, $3, $4, $5)
		returning id, service_name, price, user_id::text, start_date, end_date, created_at, updated_at`

	return scanSubscription(p.db.QueryRow(ctx, query,
		input.ServiceName,
		input.Price,
		input.UserID,
		input.StartDate.Date(),
		monthPtr(input.EndDate),
	))
}

func (p *Postgres) Get(ctx context.Context, id int64) (service.Subscription, error) {
	const query = `
		select id, service_name, price, user_id::text, start_date, end_date, created_at, updated_at
		from subscriptions
		where id = $1`

	return scanSubscription(p.db.QueryRow(ctx, query, id))
}

func (p *Postgres) Update(ctx context.Context, id int64, input service.UpdateSubscription) (service.Subscription, error) {
	const query = `
		update subscriptions
		set service_name = $2,
			price = $3,
			user_id = $4,
			start_date = $5,
			end_date = $6,
			updated_at = now()
		where id = $1
		returning id, service_name, price, user_id::text, start_date, end_date, created_at, updated_at`

	return scanSubscription(p.db.QueryRow(ctx, query,
		id,
		input.ServiceName,
		input.Price,
		input.UserID,
		input.StartDate.Date(),
		monthPtr(input.EndDate),
	))
}

func (p *Postgres) Delete(ctx context.Context, id int64) error {
	tag, err := p.db.Exec(ctx, "delete from subscriptions where id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (p *Postgres) List(ctx context.Context, filter service.ListFilter) ([]service.Subscription, error) {
	const query = `
		select id, service_name, price, user_id::text, start_date, end_date, created_at, updated_at
		from subscriptions
		where ($1 = '' or user_id = $1::uuid)
		  and ($2 = '' or service_name ilike '%' || $2 || '%')
		order by id desc
		limit $3 offset $4`

	rows, err := p.db.Query(ctx, query, filter.UserID, filter.ServiceName, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]service.Subscription, 0)
	for rows.Next() {
		item, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *Postgres) Total(ctx context.Context, filter service.TotalFilter) (int64, error) {
	const query = `
		select coalesce(sum(s.price), 0)::bigint
		from subscriptions s
		join generate_series($1::date, $2::date, interval '1 month') month_start on true
		where s.start_date <= month_start
		  and (s.end_date is null or s.end_date >= month_start)
		  and ($3 = '' or s.user_id = $3::uuid)
		  and ($4 = '' or s.service_name ilike '%' || $4 || '%')`

	var total int64
	err := p.db.QueryRow(ctx, query, filter.From.Date(), filter.To.Date(), filter.UserID, filter.ServiceName).Scan(&total)
	return total, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(row rowScanner) (service.Subscription, error) {
	var item service.Subscription
	var startDate time.Time
	var endDate *time.Time

	err := row.Scan(
		&item.ID,
		&item.ServiceName,
		&item.Price,
		&item.UserID,
		&startDate,
		&endDate,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Subscription{}, service.ErrNotFound
	}
	if err != nil {
		return service.Subscription{}, err
	}

	item.StartDate = service.Month{Time: normalizeMonth(startDate)}
	if endDate != nil {
		normalized := service.Month{Time: normalizeMonth(*endDate)}
		item.EndDate = &normalized
	}
	return item, nil
}

func normalizeMonth(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func monthPtr(value *service.Month) any {
	if value == nil {
		return nil
	}
	return value.Date()
}
