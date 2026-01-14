-- name: GetUserByEmail :one
SELECT id, email, name FROM users WHERE email=$1;
