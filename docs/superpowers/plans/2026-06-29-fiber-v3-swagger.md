# Fiber v3 and Swagger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate MOEX Gateway from `net/http` to Fiber v3 and expose generated Swagger UI.

**Architecture:** Keep MOEX/CBR service interfaces unchanged. Replace only HTTP routing, handlers, and server lifecycle; generate Swagger 2.0 from handler annotations and serve it through Fiber contrib.

**Tech Stack:** Go 1.25, Fiber v3, `gofiber/contrib/v3/swaggo`, `swaggo/swag`, Go testing.

---

### Task 1: Fiber HTTP transport

**Files:** `go.mod`, `go.sum`, `internal/httpserver/*.go`, `internal/httpserver/router_test.go`

- [ ] Convert router tests to `fiber.App.Test`, preserving every existing contract assertion; run `go test ./internal/httpserver` and confirm RED before production edits.
- [ ] Replace `net/http` router and handlers with Fiber v3 equivalents. Preserve routes, validation, JSON bodies, status codes, `Allow`, service calls, and standard request-context propagation.
- [ ] Run `go test ./internal/httpserver`; expect PASS.

### Task 2: Swagger and lifecycle

**Files:** `cmd/gateway/main.go`, `internal/app/app.go`, `internal/app/app_test.go`, `internal/httpserver/router.go`, `Makefile`, `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`

- [ ] Add failing tests for `/swagger/index.html`, required generated paths/schemas, and Fiber graceful shutdown.
- [ ] Add Swagger metadata/handler annotations, `make swagger`, generated docs import, `/swagger/*`, and Fiber `Listen`/five-second shutdown lifecycle.
- [ ] Run `make swagger` and `go test ./...`; expect PASS.

### Task 3: Documentation and verification

**Files:** `README.md`

- [ ] Document Go 1.25, Fiber, `/swagger/index.html`, and `make swagger`.
- [ ] Run `make swagger`, require no generated-doc diff, then run `go test ./...`, `go vet ./...`, and `go build ./...`; all must pass.
