package disputes

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

func ptrStatus(s models.DisputeStatus) *models.DisputeStatus { return &s }
func ptrUUID(u uuid.UUID) *uuid.UUID                          { return &u }

// Получение списка споров: без фильтров — нет доп. условий и пагинации.
func TestBuildListDisputesQuery_NoFilters(t *testing.T) {
	query, args := buildListDisputesQuery(DisputeFilter{})

	if len(args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(args))
	}
	if !strings.Contains(query, "WHERE 1=1") {
		t.Errorf("query must contain base WHERE clause")
	}
	if strings.Contains(query, "d.status =") {
		t.Errorf("query must not contain status filter when none provided")
	}
	if strings.Contains(query, "LIMIT") {
		t.Errorf("query must not contain LIMIT when limit is 0")
	}
	if !strings.Contains(query, "ORDER BY d.created_at DESC") {
		t.Errorf("query must be ordered by created_at DESC")
	}
}

// Фильтр по статусу: один аргумент и плейсхолдер $1.
func TestBuildListDisputesQuery_StatusFilter(t *testing.T) {
	query, args := buildListDisputesQuery(DisputeFilter{Status: ptrStatus(models.DisputeNew)})

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != models.DisputeNew {
		t.Errorf("expected status arg %q, got %v", models.DisputeNew, args[0])
	}
	if !strings.Contains(query, "d.status = $1") {
		t.Errorf("query must contain 'd.status = $1', got: %s", query)
	}
}

// Все фильтры + пагинация: корректные плейсхолдеры и порядок аргументов.
func TestBuildListDisputesQuery_AllFiltersAndPagination(t *testing.T) {
	providerID := uuid.New()
	casinoID := uuid.New()

	query, args := buildListDisputesQuery(DisputeFilter{
		Status:     ptrStatus(models.DisputeUnderReview),
		ProviderID: ptrUUID(providerID),
		CasinoID:   ptrUUID(casinoID),
		Limit:      25,
		Offset:     50,
	})

	if len(args) != 5 {
		t.Fatalf("expected 5 args (status, provider, casino, limit, offset), got %d", len(args))
	}
	for _, frag := range []string{
		"d.status = $1",
		"d.provider_id = $2",
		"d.casino_id = $3",
		"LIMIT $4 OFFSET $5",
	} {
		if !strings.Contains(query, frag) {
			t.Errorf("query must contain %q, got: %s", frag, query)
		}
	}
	if args[3] != 25 || args[4] != 50 {
		t.Errorf("expected limit=25 offset=50, got limit=%v offset=%v", args[3], args[4])
	}
}

// Обновление спора: терминальные статусы фиксируют разрешение, рабочие — нет.
func TestIsResolvedStatus(t *testing.T) {
	cases := map[models.DisputeStatus]bool{
		models.DisputeNew:                      false,
		models.DisputeUnderReview:              false,
		models.DisputeAwaitingProviderResponse: false,
		models.DisputeMerchantWon:              true,
		models.DisputeProviderWon:              true,
		models.DisputeClosed:                   true,
	}
	for status, want := range cases {
		if got := isResolvedStatus(status); got != want {
			t.Errorf("isResolvedStatus(%q) = %v, want %v", status, got, want)
		}
	}
}
