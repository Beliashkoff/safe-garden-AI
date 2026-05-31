// Command seed loads the fertilizer catalog from a CSV file into the
// fertilizers table (ROADMAP §5.1). It is idempotent: rows are upserted by slug,
// so re-running with an updated CSV is safe. Until the customer delivers the real
// catalog (SPEC Q2) the CSV is a placeholder with a single demo row used by tests.
//
// Usage (from backend/): go run ./cmd/seed --csv ../infra/data/fertilizers.csv
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/config"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const runTimeout = 2 * time.Minute

// arraySep separates multiple values inside a single CSV cell (problems/plants),
// since the comma is the CSV field delimiter.
const arraySep = ";"

// expected CSV header (column order is fixed; the header row is validated).
var header = []string{
	"slug", "name", "short_desc", "long_desc", "image_url", "deeplink_url",
	"category", "problems", "plants", "priority", "active",
}

func main() {
	csvPath := flag.String("csv", "../infra/data/fertilizers.csv", "path to the fertilizer catalog CSV")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Env, cfg.LogLevel)
	slog.SetDefault(logger)

	rows, err := readCSV(*csvPath)
	if err != nil {
		slog.Error("read csv failed", "path", *csvPath, "err", err)
		os.Exit(1)
	}
	if len(rows) == 0 {
		slog.Warn("csv has no data rows — nothing to seed", "path", *csvPath)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	store, err := storage.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("storage init failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// Upsert all rows in one transaction so a malformed row aborts the whole
	// batch rather than leaving the catalog half-applied.
	err = store.ExecTx(ctx, func(q *db.Queries) error {
		for i, params := range rows {
			if _, err := q.UpsertFertilizerBySlug(ctx, params); err != nil {
				return fmt.Errorf("row %d (slug %q): %w", i+2, params.Slug, err)
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}

	slog.Info("catalog seeded", "rows", len(rows), "path", *csvPath)
}

// readCSV parses the catalog file into upsert params, validating the header and
// each row's required fields.
func readCSV(path string) ([]db.UpsertFertilizerBySlugParams, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = len(header)
	r.TrimLeadingSpace = true

	first, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if err := validateHeader(first); err != nil {
		return nil, err
	}

	var out []db.UpsertFertilizerBySlugParams
	for line := 2; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		params, err := rowToParams(rec)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		out = append(out, params)
	}
	return out, nil
}

func validateHeader(got []string) error {
	if len(got) != len(header) {
		return fmt.Errorf("header has %d columns, want %d", len(got), len(header))
	}
	for i, name := range header {
		if !strings.EqualFold(strings.TrimSpace(got[i]), name) {
			return fmt.Errorf("header column %d is %q, want %q", i+1, got[i], name)
		}
	}
	return nil
}

func rowToParams(rec []string) (db.UpsertFertilizerBySlugParams, error) {
	var p db.UpsertFertilizerBySlugParams

	p.Slug = strings.TrimSpace(rec[0])
	p.Name = strings.TrimSpace(rec[1])
	p.ShortDesc = strings.TrimSpace(rec[2])
	p.Category = strings.TrimSpace(rec[6])

	if p.Slug == "" || p.Name == "" || p.ShortDesc == "" || p.Category == "" {
		return p, errors.New("slug, name, short_desc and category are required")
	}

	p.LongDesc = optText(rec[3])
	p.ImageUrl = optText(rec[4])
	p.DeeplinkUrl = optText(rec[5])

	p.Problems = splitList(rec[7])
	if len(p.Problems) == 0 {
		return p, errors.New("problems must list at least one value")
	}
	p.Plants = splitList(rec[8]) // nil → NULL (universal)

	priority, err := optInt(rec[9])
	if err != nil {
		return p, fmt.Errorf("priority: %w", err)
	}
	p.Priority = priority

	p.Active, err = parseBool(rec[10])
	if err != nil {
		return p, fmt.Errorf("active: %w", err)
	}

	return p, nil
}

// splitList parses a ';'-separated cell into a trimmed, non-empty slice; an empty
// cell yields nil (stored as SQL NULL for the nullable plants column).
func splitList(cell string) []string {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return nil
	}
	parts := strings.Split(cell, arraySep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func optText(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	return pgtype.Text{String: s, Valid: s != ""}
}

func optInt(s string) (pgtype.Int4, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Int4{}, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return pgtype.Int4{}, err
	}
	return pgtype.Int4{Int32: int32(n), Valid: true}, nil
}

// parseBool defaults an empty cell to true (active by default, matching the
// table default).
func parseBool(s string) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return true, nil
	}
	return strconv.ParseBool(s)
}
