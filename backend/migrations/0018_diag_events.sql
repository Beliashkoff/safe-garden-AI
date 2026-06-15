-- +goose Up
-- Diagnosis-problem analytics for the catalog. Each recommend_fertilizer tool
-- call records the problem key the model queried and whether the catalog had a
-- matching product. No user_id / PII — problem is a closed enum
-- (llm.FertilizerProblemKeys). Powers the "problem distribution + catalog
-- miss-rate" admin view: which plant problems users hit, and which have no
-- fertilizer in the catalog yet (a map of assortment gaps).
CREATE TABLE diag_events (
    id         BIGSERIAL PRIMARY KEY,
    problem    TEXT NOT NULL,
    matched    BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX diag_events_created_idx ON diag_events (created_at);

-- +goose Down
DROP TABLE IF EXISTS diag_events;
