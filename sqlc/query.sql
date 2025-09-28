-- Events
-- name: CreateEvent :one
INSERT INTO events (title, description, type, started_date, end_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
