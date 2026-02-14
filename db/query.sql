-- name: CreateURL :one
INSERT INTO urls (short_code, original_url)
VALUES (?, ?)
RETURNING *;

-- name: FindByShortCode :one
SELECT * FROM urls WHERE short_code = ? LIMIT 1;

-- name: FindByOriginalURL :one
SELECT * FROM urls WHERE original_url = ? LIMIT 1;
