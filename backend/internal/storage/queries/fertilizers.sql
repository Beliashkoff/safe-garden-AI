-- name: RecommendFertilizers :many
-- ARCH §6.4. plant = NULL → no crop filter; universal items (plants IS NULL) always
-- match. Ranked by priority (highest first), at most 3.
SELECT id, slug, name, short_desc, image_url, deeplink_url
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
