-- name: CreateUser :one
INSERT INTO users (id, username, username_idempotent, universal, email_unsubscriptions)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;