package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	maxNameLen      = 200
	maxSlugLen      = 100
	maxShortDescLen = 500
	maxLongDescLen  = 4000
	maxCategoryLen  = 100
	maxURLLen       = 1000
	maxPlants       = 30
	// maxImageBytes caps the catalog image upload (server-side PUT, small files
	// only — this is not the user media path).
	maxImageBytes = 5 * 1024 * 1024
)

// imageContentTypes maps allowed upload types to their key extension.
var imageContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var problemKeySet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(llm.FertilizerProblemKeys))
	for _, k := range llm.FertilizerProblemKeys {
		m[k] = struct{}{}
	}
	return m
}()

// ProductInput is the create/update payload for one catalog item.
type ProductInput struct {
	Slug        string // optional on create — derived from Name when empty
	Name        string
	ShortDesc   string
	LongDesc    string
	ImageURL    string
	DeeplinkURL string
	Category    string
	Problems    []string
	Plants      []string
	Priority    int32
	Active      bool
	PriceRub    *int32
}

// ProductView is the transport-agnostic catalog item projection.
type ProductView struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	ShortDesc   string
	LongDesc    string
	ImageURL    string
	DeeplinkURL string
	Category    string
	Problems    []string
	Plants      []string
	Priority    int32
	Active      bool
	PriceRub    *int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toProductView(f db.Fertilizer) ProductView {
	v := ProductView{
		ID:          f.ID,
		Slug:        f.Slug,
		Name:        f.Name,
		ShortDesc:   f.ShortDesc,
		LongDesc:    f.LongDesc.String,
		ImageURL:    f.ImageUrl.String,
		DeeplinkURL: f.DeeplinkUrl.String,
		Category:    f.Category,
		Problems:    f.Problems,
		Plants:      f.Plants,
		Active:      f.Active,
		CreatedAt:   f.CreatedAt.Time,
		UpdatedAt:   f.UpdatedAt.Time,
	}
	if f.Priority.Valid {
		v.Priority = f.Priority.Int32
	}
	if f.PriceRub.Valid {
		price := f.PriceRub.Int32
		v.PriceRub = &price
	}
	if v.Plants == nil {
		v.Plants = []string{}
	}
	return v
}

// ListProducts returns the whole catalog, newest first.
func (s *Service) ListProducts(ctx context.Context) ([]ProductView, error) {
	rows, err := s.store.ListFertilizers(ctx)
	if err != nil {
		return nil, fmt.Errorf("admin: list fertilizers: %w", err)
	}
	out := make([]ProductView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProductView(r))
	}
	return out, nil
}

// GetProduct returns one catalog item.
func (s *Service) GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error) {
	row, err := s.store.GetFertilizerByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductView{}, ErrProductNotFound
	}
	if err != nil {
		return ProductView{}, fmt.Errorf("admin: get fertilizer: %w", err)
	}
	return toProductView(row), nil
}

// CreateProduct validates and inserts a catalog item. The model can recommend
// it on the very next chat turn (the tool callback reads the same table).
func (s *Service) CreateProduct(ctx context.Context, adminID uuid.UUID, in ProductInput, dev DeviceMeta) (ProductView, error) {
	norm, err := normalizeProduct(in)
	if err != nil {
		return ProductView{}, err
	}
	row, err := s.store.CreateFertilizer(ctx, db.CreateFertilizerParams{
		Slug:        norm.Slug,
		Name:        norm.Name,
		ShortDesc:   norm.ShortDesc,
		LongDesc:    textOrNull(norm.LongDesc),
		ImageUrl:    textOrNull(norm.ImageURL),
		DeeplinkUrl: textOrNull(norm.DeeplinkURL),
		Category:    norm.Category,
		Problems:    norm.Problems,
		Plants:      norm.Plants,
		Priority:    pgtype.Int4{Int32: norm.Priority, Valid: true},
		Active:      norm.Active,
		PriceRub:    int4OrNull(norm.PriceRub),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ProductView{}, ErrSlugTaken
		}
		return ProductView{}, fmt.Errorf("admin: create fertilizer: %w", err)
	}
	s.audit(ctx, adminID, "product_created", "fertilizer", row.Slug, dev.IP)
	return toProductView(row), nil
}

// UpdateProduct validates and updates a catalog item in full.
func (s *Service) UpdateProduct(ctx context.Context, adminID uuid.UUID, id uuid.UUID, in ProductInput, dev DeviceMeta) (ProductView, error) {
	norm, err := normalizeProduct(in)
	if err != nil {
		return ProductView{}, err
	}
	row, err := s.store.UpdateFertilizer(ctx, db.UpdateFertilizerParams{
		ID:          id,
		Slug:        norm.Slug,
		Name:        norm.Name,
		ShortDesc:   norm.ShortDesc,
		LongDesc:    textOrNull(norm.LongDesc),
		ImageUrl:    textOrNull(norm.ImageURL),
		DeeplinkUrl: textOrNull(norm.DeeplinkURL),
		Category:    norm.Category,
		Problems:    norm.Problems,
		Plants:      norm.Plants,
		Priority:    pgtype.Int4{Int32: norm.Priority, Valid: true},
		Active:      norm.Active,
		PriceRub:    int4OrNull(norm.PriceRub),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductView{}, ErrProductNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ProductView{}, ErrSlugTaken
		}
		return ProductView{}, fmt.Errorf("admin: update fertilizer: %w", err)
	}
	s.audit(ctx, adminID, "product_updated", "fertilizer", row.Slug, dev.IP)
	return toProductView(row), nil
}

// DeleteProduct removes a catalog item permanently. For a reversible
// "выключить" the panel uses Active=false instead.
func (s *Service) DeleteProduct(ctx context.Context, adminID, id uuid.UUID, dev DeviceMeta) error {
	n, err := s.store.DeleteFertilizer(ctx, id)
	if err != nil {
		return fmt.Errorf("admin: delete fertilizer: %w", err)
	}
	if n == 0 {
		return ErrProductNotFound
	}
	s.audit(ctx, adminID, "product_deleted", "fertilizer", id.String(), dev.IP)
	return nil
}

// UploadProductImage stores a catalog image publicly and returns its URL. The
// admin then saves the URL on the product.
func (s *Service) UploadProductImage(ctx context.Context, adminID uuid.UUID, contentType string, data []byte, dev DeviceMeta) (string, error) {
	declared := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(declared, ';'); i >= 0 {
		declared = strings.TrimSpace(declared[:i])
	}
	ext, ok := imageContentTypes[declared]
	if !ok {
		return "", fmt.Errorf("%w: unsupported image type", ErrValidation)
	}
	if len(data) == 0 || len(data) > maxImageBytes {
		return "", fmt.Errorf("%w: image size out of range", ErrValidation)
	}
	// Sniff the real bytes and require them to match the declared type, so a
	// non-image payload (HTML/SVG/JS) cannot be stored under an image MIME on a
	// public URL regardless of the client-supplied part header.
	sniffed := http.DetectContentType(data)
	if i := strings.IndexByte(sniffed, ';'); i >= 0 {
		sniffed = sniffed[:i]
	}
	if sniffed != declared {
		return "", fmt.Errorf("%w: file content does not match its type", ErrValidation)
	}
	key := "catalog/" + uuid.NewString() + ext
	url, err := s.objs.PutPublic(ctx, key, contentType, data)
	if err != nil {
		return "", fmt.Errorf("admin: put image: %w", err)
	}
	s.audit(ctx, adminID, "product_image_uploaded", "image", key, dev.IP)
	return url, nil
}

// normalizeProduct trims, derives the slug and validates every field.
func normalizeProduct(in ProductInput) (ProductInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(strings.ToLower(in.Slug))
	in.ShortDesc = strings.TrimSpace(in.ShortDesc)
	in.LongDesc = strings.TrimSpace(in.LongDesc)
	in.Category = strings.TrimSpace(in.Category)
	in.ImageURL = strings.TrimSpace(in.ImageURL)
	in.DeeplinkURL = strings.TrimSpace(in.DeeplinkURL)

	if in.Slug == "" {
		in.Slug = Slugify(in.Name)
	}
	switch {
	case in.Name == "" || len(in.Name) > maxNameLen:
		return in, fmt.Errorf("%w: name", ErrValidation)
	case in.Slug == "" || len(in.Slug) > maxSlugLen || !validSlug(in.Slug):
		return in, fmt.Errorf("%w: slug", ErrValidation)
	case in.ShortDesc == "" || len(in.ShortDesc) > maxShortDescLen:
		return in, fmt.Errorf("%w: short_desc", ErrValidation)
	case len(in.LongDesc) > maxLongDescLen:
		return in, fmt.Errorf("%w: long_desc", ErrValidation)
	case in.Category == "" || len(in.Category) > maxCategoryLen:
		return in, fmt.Errorf("%w: category", ErrValidation)
	case len(in.ImageURL) > maxURLLen || len(in.DeeplinkURL) > maxURLLen:
		return in, fmt.Errorf("%w: url too long", ErrValidation)
	case len(in.Problems) == 0:
		return in, fmt.Errorf("%w: at least one problem key required", ErrValidation)
	case len(in.Plants) > maxPlants:
		return in, fmt.Errorf("%w: too many plants", ErrValidation)
	case in.PriceRub != nil && (*in.PriceRub < 0 || *in.PriceRub > 1_000_000):
		return in, fmt.Errorf("%w: price_rub", ErrValidation)
	}
	seen := map[string]struct{}{}
	for _, p := range in.Problems {
		if _, ok := problemKeySet[p]; !ok {
			return in, fmt.Errorf("%w: unknown problem key %q", ErrValidation, p)
		}
		if _, dup := seen[p]; dup {
			return in, fmt.Errorf("%w: duplicate problem key %q", ErrValidation, p)
		}
		seen[p] = struct{}{}
	}
	cleanPlants := make([]string, 0, len(in.Plants))
	seenPlants := map[string]struct{}{}
	for _, p := range in.Plants {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		if _, dup := seenPlants[p]; dup {
			continue
		}
		seenPlants[p] = struct{}{}
		cleanPlants = append(cleanPlants, p)
	}
	if len(cleanPlants) == 0 {
		cleanPlants = nil // NULL = universal product (matches any plant)
	}
	in.Plants = cleanPlants
	return in, nil
}

// translit maps Cyrillic to Latin for slug derivation from Russian names.
var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// Slugify derives a URL-safe slug from a (possibly Russian) product name.
// Exported for tests.
func Slugify(name string) string {
	var b strings.Builder
	prevDash := true // suppress leading dash
	for _, r := range strings.ToLower(name) {
		var part string
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			part = string(r)
		default:
			if t, ok := translit[r]; ok {
				part = t
			}
		}
		if part == "" {
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
		b.WriteString(part)
		prevDash = false
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > maxSlugLen {
		s = strings.Trim(s[:maxSlugLen], "-")
	}
	return s
}

func validSlug(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func int4OrNull(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
