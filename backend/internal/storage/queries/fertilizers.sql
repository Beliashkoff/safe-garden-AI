-- name: RecommendFertilizers :many
-- ARCH §6.4. plant = NULL → no crop filter; universal items (plants IS NULL) always
-- match. Ranked by priority (highest first), at most 3.
SELECT id, slug, name, short_desc, image_url, deeplink_url, price_rub
FROM fertilizers
WHERE active
  AND problems @> ARRAY[sqlc.arg('problem')::text]
  AND (
    sqlc.narg('plant')::text IS NULL
    OR plants IS NULL
    OR plants @> ARRAY[sqlc.narg('plant')::text]
  )
ORDER BY priority DESC NULLS LAST
LIMIT 3;

-- name: GetFertilizerBySlug :one
SELECT * FROM fertilizers WHERE slug = $1;

-- name: UpsertFertilizerBySlug :one
-- Idempotent catalog seeding (Stage 5). Updates everything but id/created_at and
-- bumps updated_at on conflict.
INSERT INTO fertilizers (
    slug, name, short_desc, long_desc, image_url, deeplink_url,
    category, problems, plants, priority, active
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    short_desc = EXCLUDED.short_desc,
    long_desc = EXCLUDED.long_desc,
    image_url = EXCLUDED.image_url,
    deeplink_url = EXCLUDED.deeplink_url,
    category = EXCLUDED.category,
    problems = EXCLUDED.problems,
    plants = EXCLUDED.plants,
    priority = EXCLUDED.priority,
    active = EXCLUDED.active,
    updated_at = NOW()
RETURNING *;

-- Admin panel catalog CRUD. The catalog is small (tens of items), so the list
-- is unpaginated and filtered client-side.

-- name: ListFertilizers :many
SELECT * FROM fertilizers ORDER BY created_at DESC;

-- name: GetFertilizerByID :one
SELECT * FROM fertilizers WHERE id = $1;

-- name: CreateFertilizer :one
INSERT INTO fertilizers (
    slug, name, short_desc, long_desc, image_url, deeplink_url,
    category, problems, plants, priority, active, price_rub
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateFertilizer :one
UPDATE fertilizers SET
    slug = $2,
    name = $3,
    short_desc = $4,
    long_desc = $5,
    image_url = $6,
    deeplink_url = $7,
    category = $8,
    problems = $9,
    plants = $10,
    priority = $11,
    active = $12,
    price_rub = $13,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteFertilizer :execrows
DELETE FROM fertilizers WHERE id = $1;

-- name: CountFertilizers :one
SELECT
    COUNT(*)                                  AS total,
    COUNT(*) FILTER (WHERE active)::bigint    AS active
FROM fertilizers;
