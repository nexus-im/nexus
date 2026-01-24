# Repository Guidelines

## Project Structure & Module Organization
This repository implements the Go-based Instant Messaging (IM) server. 
- `main.go` is the entrypoint.
- `internal/` contains core logic (e.g., authentication).
- `store/` handles data access (e.g., `store/user/`).
- `doc/` contains architectural and API documentation.
- `docker-compose.yml` provisions a local Postgres instance.

## Build, Test, and Development Commands
Run commands from the repository root:
- `make build` builds `bin/backend_server`.
- `make run` builds and starts the server (default `:8081` per Makefile).
- `make test` runs `go test -v ./...`.
- `make lint` uses `golangci-lint` if available, otherwise `go vet ./...`.
For the database, use `docker-compose up -d`. The server reads `DATABASE_URL` or falls back to `postgres://nexus_user:nexus_password@localhost:5432/nexus?sslmode=disable`.

## Coding Style & Naming Conventions
Use standard Go formatting (`gofmt`) and idiomatic naming (CamelCase for exported, lowerCamelCase for unexported, `*_test.go` for tests). Keep packages small and grouped by domain (e.g., `store/user`). Prefer short, explicit handler names like `handleLogin`.

## Testing Guidelines
Tests use Go’s standard `testing` package. Name files `*_test.go` and tests `TestXxx`. Run the full suite with `make test`.

## Commit & Pull Request Guidelines
Follow Conventional Commits with scopes, e.g. `feat(auth): implement JWT authentication`. Use types like `feat`, `fix`, `chore`, etc.

## Security & Configuration Tips
Avoid hard-coded secrets. Use `DATABASE_URL` for database configuration and keep credentials in local env files or your shell config.