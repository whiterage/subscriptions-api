package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const MaxTotalPeriodMonths = 36

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateSubscription) (Subscription, error) {
	if err := validateSubscription(input); err != nil {
		return Subscription{}, err
	}
	return s.repo.Create(ctx, normalizeSubscription(input))
}

func (s *Service) Get(ctx context.Context, id int64) (Subscription, error) {
	if id <= 0 {
		return Subscription{}, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateSubscription) (Subscription, error) {
	if id <= 0 {
		return Subscription{}, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	if err := validateSubscription(input); err != nil {
		return Subscription{}, err
	}
	return s.repo.Update(ctx, id, normalizeSubscription(input))
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Subscription, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.ServiceName = strings.TrimSpace(filter.ServiceName)
	if filter.UserID != "" && !uuidPattern.MatchString(filter.UserID) {
		return nil, fmt.Errorf("%w: user_id must be a valid UUID", ErrInvalidInput)
	}
	return s.repo.List(ctx, filter)
}

func (s *Service) Total(ctx context.Context, filter TotalFilter) (int64, error) {
	if filter.From.Date().After(filter.To.Date()) {
		return 0, fmt.Errorf("%w: from must be before or equal to to", ErrInvalidInput)
	}
	if monthsBetween(filter.From.Date(), filter.To.Date()) > MaxTotalPeriodMonths {
		return 0, fmt.Errorf("%w: period must not exceed %d months", ErrInvalidInput, MaxTotalPeriodMonths)
	}
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.ServiceName = strings.TrimSpace(filter.ServiceName)
	if filter.UserID != "" && !uuidPattern.MatchString(filter.UserID) {
		return 0, fmt.Errorf("%w: user_id must be a valid UUID", ErrInvalidInput)
	}
	return s.repo.Total(ctx, filter)
}

func validateSubscription(input CreateSubscription) error {
	if strings.TrimSpace(input.ServiceName) == "" {
		return fmt.Errorf("%w: service_name is required", ErrInvalidInput)
	}
	if input.Price <= 0 {
		return fmt.Errorf("%w: price must be positive", ErrInvalidInput)
	}
	if strings.TrimSpace(input.UserID) == "" {
		return fmt.Errorf("%w: user_id is required", ErrInvalidInput)
	}
	if !uuidPattern.MatchString(strings.TrimSpace(input.UserID)) {
		return fmt.Errorf("%w: user_id must be a valid UUID", ErrInvalidInput)
	}
	if input.EndDate != nil && input.EndDate.Date().Before(input.StartDate.Date()) {
		return fmt.Errorf("%w: end_date must be after or equal to start_date", ErrInvalidInput)
	}
	return nil
}

func monthsBetween(from, to time.Time) int {
	return (to.Year()-from.Year())*12 + int(to.Month()-from.Month()) + 1
}

func normalizeSubscription(input CreateSubscription) CreateSubscription {
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	input.UserID = strings.TrimSpace(input.UserID)
	return input
}
