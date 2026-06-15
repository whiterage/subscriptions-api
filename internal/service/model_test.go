package service

import (
	"errors"
	"testing"
)

func TestParseMonth(t *testing.T) {
	month, err := ParseMonth("07-2025")
	if err != nil {
		t.Fatalf("ParseMonth returned error: %v", err)
	}

	if got := month.Format(MonthLayout); got != "07-2025" {
		t.Fatalf("expected 07-2025, got %s", got)
	}
}

func TestParseMonthInvalid(t *testing.T) {
	_, err := ParseMonth("2025-07")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestValidateSubscriptionEndBeforeStart(t *testing.T) {
	start, err := ParseMonth("07-2025")
	if err != nil {
		t.Fatal(err)
	}
	end, err := ParseMonth("06-2025")
	if err != nil {
		t.Fatal(err)
	}

	err = validateSubscription(CreateSubscription{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      "60601fee-2bf1-4721-ae6f-7636e79a0cba",
		StartDate:   start,
		EndDate:     &end,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestValidateSubscriptionRejectsInvalidUUID(t *testing.T) {
	start, err := ParseMonth("07-2025")
	if err != nil {
		t.Fatal(err)
	}

	err = validateSubscription(CreateSubscription{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      "not-a-uuid",
		StartDate:   start,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTotalRejectsTooLargePeriodBeforeRepository(t *testing.T) {
	from, err := ParseMonth("01-2025")
	if err != nil {
		t.Fatal(err)
	}
	to, err := ParseMonth("01-2029")
	if err != nil {
		t.Fatal(err)
	}

	svc := New(nil)
	_, err = svc.Total(t.Context(), TotalFilter{From: from, To: to})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListRejectsInvalidUUIDBeforeRepository(t *testing.T) {
	svc := New(nil)
	_, err := svc.List(t.Context(), ListFilter{UserID: "not-a-uuid"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
