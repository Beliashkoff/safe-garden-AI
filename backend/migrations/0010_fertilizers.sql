-- +goose Up
CREATE TABLE fertilizers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug         TEXT UNIQUE NOT NULL,
    name         TEXT NOT NULL,
    short_desc   TEXT NOT NULL,
    long_desc    TEXT,
    image_url    TEXT,
    deeplink_url TEXT,
    category     TEXT NOT NULL,
    problems     TEXT[] NOT NULL,
    plants       TEXT[],
    priority     INT,
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Array containment search for recommend_fertilizer (problems @> ARRAY[...], ARCH §6.4).
CREATE INDEX fertilizers_problems_gin_idx ON fertilizers USING GIN (problems);

-- +goose Down
DROP TABLE IF EXISTS fertilizers;
