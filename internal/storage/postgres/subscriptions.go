package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/storage"
)

type SubscriptionStorage struct {
	pool *pgxpool.Pool
}

func NewSubscriptionStorage(pool *pgxpool.Pool) *SubscriptionStorage {
	return &SubscriptionStorage{pool: pool}
}

func (s *SubscriptionStorage) Create(ctx context.Context, sub *models.Subscription) error {
	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		sub.ServiceName, sub.Price, sub.UserID, sub.StartDate.Time, endDateArg(sub.EndDate),
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert subscription: %w", err)
	}

	return nil
}

func (s *SubscriptionStorage) GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	sub, err := scanSubscription(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("select subscription: %w", err)
	}

	return sub, nil
}

func (s *SubscriptionStorage) List(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions`

	var conditions []string
	var args []any

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if filter.ServiceName != nil {
		args = append(args, *filter.ServiceName)
		conditions = append(conditions, fmt.Sprintf("service_name = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	args = append(args, filter.Limit, filter.Offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select subscriptions: %w", err)
	}
	defer rows.Close()

	subs := make([]models.Subscription, 0)

	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, *sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriptions: %w", err)
	}

	return subs, nil
}

func (s *SubscriptionStorage) Update(ctx context.Context, sub *models.Subscription) error {
	query := `
		UPDATE subscriptions
		SET service_name = $1, price = $2, user_id = $3, start_date = $4, end_date = $5, updated_at = now()
		WHERE id = $6
		RETURNING created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		sub.ServiceName, sub.Price, sub.UserID, sub.StartDate.Time, endDateArg(sub.EndDate), sub.ID,
	).Scan(&sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("update subscription: %w", err)
	}

	return nil
}

func (s *SubscriptionStorage) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func endDateArg(endDate *models.MonthYear) *time.Time {
	if endDate == nil {
		return nil
	}

	return &endDate.Time
}

func scanSubscription(row pgx.Row) (*models.Subscription, error) {
	var sub models.Subscription
	var endDate *time.Time

	err := row.Scan(
		&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID,
		&sub.StartDate.Time, &endDate, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if endDate != nil {
		sub.EndDate = &models.MonthYear{Time: *endDate}
	}

	return &sub, nil
}
