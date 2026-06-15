package fertilizer

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

type fakeStore struct {
	rows      []db.RecommendFertilizersRow
	gotParams db.RecommendFertilizersParams
	diag      db.InsertDiagEventParams
}

func (f *fakeStore) RecommendFertilizers(_ context.Context, arg db.RecommendFertilizersParams) ([]db.RecommendFertilizersRow, error) {
	f.gotParams = arg
	return f.rows, nil
}

func (f *fakeStore) InsertDiagEvent(_ context.Context, arg db.InsertDiagEventParams) error {
	f.diag = arg
	return nil
}

func TestRecommend_MapsRowsAndNullableFields(t *testing.T) {
	id := uuid.New()
	store := &fakeStore{rows: []db.RecommendFertilizersRow{
		{
			ID:        id,
			Slug:      "k-boost",
			Name:      "Калий-Буст",
			ShortDesc: "desc",
			ImageUrl:  pgtype.Text{String: "https://img", Valid: true},
			// DeeplinkUrl left NULL → should map to "".
		},
	}}
	svc := NewService(store)

	out, err := svc.Recommend(context.Background(), llm.FertilizerToolArgs{Problem: "potassium_deficiency", Plant: "tomato"})
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 product, got %d", len(out))
	}
	p := out[0]
	if p.ID != id.String() || p.Slug != "k-boost" || p.ImageURL != "https://img" {
		t.Fatalf("unexpected product mapping: %+v", p)
	}
	if p.DeeplinkURL != "" {
		t.Fatalf("NULL deeplink should map to empty string, got %q", p.DeeplinkURL)
	}
}

func TestRecommend_PassesPlantFilter(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)

	if _, err := svc.Recommend(context.Background(), llm.FertilizerToolArgs{Problem: "wilting", Plant: "cucumber"}); err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if store.gotParams.Problem != "wilting" {
		t.Fatalf("problem not passed: %+v", store.gotParams)
	}
	if !store.gotParams.Plant.Valid || store.gotParams.Plant.String != "cucumber" {
		t.Fatalf("plant filter not passed: %+v", store.gotParams.Plant)
	}
}

func TestRecommend_NoPlantLeavesFilterNull(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)

	if _, err := svc.Recommend(context.Background(), llm.FertilizerToolArgs{Problem: "wilting"}); err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if store.gotParams.Plant.Valid {
		t.Fatalf("plant should be NULL when not provided, got %+v", store.gotParams.Plant)
	}
}

func TestRecommend_EmptyProblemErrors(t *testing.T) {
	svc := NewService(&fakeStore{})
	if _, err := svc.Recommend(context.Background(), llm.FertilizerToolArgs{}); err == nil {
		t.Fatalf("expected error for empty problem")
	}
}

func TestRecommend_EmptyResultIsEmptySlice(t *testing.T) {
	svc := NewService(&fakeStore{rows: nil})
	out, err := svc.Recommend(context.Background(), llm.FertilizerToolArgs{Problem: "wilting"})
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty result, got %+v", out)
	}
}
