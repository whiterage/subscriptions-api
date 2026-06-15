package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MonthLayout = "01-2006"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("subscription not found")
)

type Subscription struct {
	ID          int64     `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      string    `json:"user_id"`
	StartDate   Month     `json:"start_date"`
	EndDate     *Month    `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Month struct {
	time.Time
}

func ParseMonth(value string) (Month, error) {
	parsed, err := time.Parse(MonthLayout, strings.TrimSpace(value))
	if err != nil {
		return Month{}, fmt.Errorf("%w: date must use MM-YYYY format", ErrInvalidInput)
	}
	return Month{Time: parsed}, nil
}

func (m Month) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.Format(MonthLayout) + `"`), nil
}

func (m *Month) UnmarshalJSON(data []byte) error {
	value := strings.Trim(string(data), `"`)
	parsed, err := ParseMonth(value)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func (m Month) Date() time.Time {
	return time.Date(m.Year(), m.Month(), 1, 0, 0, 0, 0, time.UTC)
}

type CreateSubscription struct {
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   Month  `json:"start_date"`
	EndDate     *Month `json:"end_date"`
}

type UpdateSubscription = CreateSubscription

type ListFilter struct {
	UserID      string
	ServiceName string
	Limit       int
	Offset      int
}

type TotalFilter struct {
	From        Month
	To          Month
	UserID      string
	ServiceName string
}

type Repository interface {
	Create(ctx context.Context, input CreateSubscription) (Subscription, error)
	Get(ctx context.Context, id int64) (Subscription, error)
	Update(ctx context.Context, id int64, input UpdateSubscription) (Subscription, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter ListFilter) ([]Subscription, error)
	Total(ctx context.Context, filter TotalFilter) (int64, error)
}
