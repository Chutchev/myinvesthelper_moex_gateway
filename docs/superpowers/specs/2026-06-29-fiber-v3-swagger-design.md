# Fiber v3 and Swagger Design

## Goal

Replace the gateway's `net/http` transport with Fiber v3 while preserving its
public API, and publish generated Swagger documentation without requiring
manual OpenAPI editing.

## Runtime and dependencies

- Raise the module's minimum Go version from 1.22.4 to 1.25 because Fiber v3
  requires Go 1.25 or newer.
- Use `github.com/gofiber/fiber/v3` for routing and server lifecycle.
- Use `github.com/swaggo/swag` to generate the Swagger specification from Go
  annotations.
- Use `github.com/gofiber/contrib/v3/swaggo` to serve Swagger UI and the
  generated specification.
- Permit Go's toolchain mechanism to download Go 1.25 when the host does not
  already provide it.

## HTTP architecture

`internal/httpserver.NewRouter` will construct and return a `*fiber.App`.
Handlers will use Fiber v3 contexts and return errors through Fiber's handler
contract. Domain service interfaces in `internal/moex` and `internal/cbr` stay
unchanged; handlers will pass a request-scoped standard `context.Context` to
them.

The existing API remains stable:

- `GET /health`
- `GET /v1/bonds/{isin}`
- `GET /v1/bonds?limit=40`
- `GET /v1/cbr/rates`

Successful payloads, validation rules, default values, status codes, and error
messages remain unchanged. Unknown routes return 404. A known route called
with an unsupported method returns 405 and advertises `GET` in the `Allow`
header.

## Swagger workflow

API metadata and handler annotations are the source of truth. Running
`make swagger` invokes `swag init` and writes generated artifacts under
`docs/`. Generated files are committed so builds and runtime do not require the
generator binary.

Swagger UI is available at `/swagger/index.html`, with supporting routes under
`/swagger/*`. Documentation covers all four endpoints, path and query
parameters, response models, and the 200, 400, 500, and 501 responses applicable
to each endpoint.

## Server lifecycle

`internal/app` owns the Fiber app and listening address. Startup uses Fiber's
listener. Cancellation triggers graceful shutdown with the existing five-second
limit. Normal shutdown returns nil; startup and shutdown failures retain useful
error wrapping.

## Error handling

Handlers return explicitly serialized JSON responses. Known
`apperrors.ErrNotImplemented` values map to 501. Unexpected service or encoding
failures map to stable 500 JSON without exposing internal error text. Invalid
ISIN and limit values map to the existing 400 payloads.

## Testing and acceptance

Router tests use Fiber's `app.Test` helper and continue asserting status,
headers, bodies, validation, service calls, and request-context propagation.
New tests verify that Swagger UI loads and that the generated specification
contains every public path and required schema.

Completion requires:

- `make swagger` followed by a clean generated-doc diff;
- `go test ./...`;
- `go vet ./...`;
- `go build ./...`.

No MOEX/CBR business logic, endpoint additions, authentication, persistence, or
frontend changes are included.
