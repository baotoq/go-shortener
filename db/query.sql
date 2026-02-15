-- name: CreateURL :one
INSERT INTO urls (short_code, original_url)
VALUES (?, ?)
RETURNING *;

-- name: FindByShortCode :one
SELECT * FROM urls WHERE short_code = ? LIMIT 1;

-- name: FindByOriginalURL :one
SELECT * FROM urls WHERE original_url = ? LIMIT 1;

-- name: ListURLs :many
SELECT * FROM urls
WHERE
  (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
  AND (sqlc.narg('search') IS NULL OR original_url LIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListURLsAsc :many
SELECT * FROM urls
WHERE
  (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
  AND (sqlc.narg('search') IS NULL OR original_url LIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountURLs :one
SELECT COUNT(*) FROM urls
WHERE
  (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
  AND (sqlc.narg('search') IS NULL OR original_url LIKE '%' || sqlc.narg('search') || '%');

-- name: DeleteURL :exec
DELETE FROM urls WHERE short_code = ?;
