export DATABASE_URL="postgres://postgres:@localhost:5432/investment_monitoring?sslmode=disable"

goose -dir migrations postgres "$DATABASE_URL" down