-- name: InsertEnrichedClick :exec
INSERT INTO clicks (short_code, clicked_at, country_code, device_type, traffic_source)
VALUES (?, ?, ?, ?, ?);

-- name: CountClicksInRange :one
SELECT COUNT(*) as total FROM clicks
WHERE short_code = ? AND clicked_at >= ? AND clicked_at <= ?;

-- name: CountByCountryInRange :many
SELECT country_code, COUNT(*) as count FROM clicks
WHERE short_code = ? AND clicked_at >= ? AND clicked_at <= ?
GROUP BY country_code ORDER BY count DESC;

-- name: CountByDeviceInRange :many
SELECT device_type, COUNT(*) as count FROM clicks
WHERE short_code = ? AND clicked_at >= ? AND clicked_at <= ?
GROUP BY device_type ORDER BY count DESC;

-- name: CountBySourceInRange :many
SELECT traffic_source, COUNT(*) as count FROM clicks
WHERE short_code = ? AND clicked_at >= ? AND clicked_at <= ?
GROUP BY traffic_source ORDER BY count DESC;

-- name: GetClickDetails :many
SELECT id, short_code, clicked_at, country_code, device_type, traffic_source
FROM clicks
WHERE short_code = ? AND clicked_at < ?
ORDER BY clicked_at DESC
LIMIT ?;

-- name: CountClicksByShortCode :one
SELECT COUNT(*) as total_clicks FROM clicks WHERE short_code = ?;

-- name: DeleteClicksByShortCode :exec
DELETE FROM clicks WHERE short_code = ?;
