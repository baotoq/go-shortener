-- name: InsertClick :exec
INSERT INTO clicks (short_code, clicked_at)
VALUES (?, ?);

-- name: CountClicksByShortCode :one
SELECT COUNT(*) as total_clicks FROM clicks WHERE short_code = ?;
