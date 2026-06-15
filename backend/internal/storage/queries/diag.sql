-- name: InsertDiagEvent :exec
-- One row per recommend_fertilizer call: the problem key the model queried and
-- whether the catalog matched. No PII (problem is a closed enum). Written
-- best-effort from the tool callback (fertilizer.Service.Recommend).
INSERT INTO diag_events (problem, matched) VALUES ($1, $2);
