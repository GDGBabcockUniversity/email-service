# GDG Email Service

Go/Gin HTTP API, PostgreSQL/GORM, Redis Streams, and SMTP. The API and one email consumer run together.

## Run

Copy `.env.example` to `.env` and fill in `DB_PASSWORD`, `ADMIN_API_KEY`, and the SMTP settings. Credentials are not included.

```sh
cp .env.example .env
# Edit .env with your configuration.
docker compose up --build -d
docker compose logs -f app
```

SMTP uses `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `SMTP_FROM`. Set `SMTP_TLS_MODE=starttls` for STARTTLS (default port 587), or `tls` for implicit TLS (set port 465). Supply your Gmail app password or Brevo SMTP credentials and an authorized sender address. `none` is supported only with empty SMTP username/password for local test servers.

Compose supplies the database hostname and local Redis URL. To use Upstash, set `REDIS_URL` to its Redis protocol `rediss://...` URL, not its REST URL. PostgreSQL and Redis data use named volumes.

For execution outside Docker, export the environment variables before running `go run ./cmd/server`; the binary does not load `.env` itself. Go 1.27.1 is required by the existing module.

## API

Management requests require `X-Admin-Key: <ADMIN_API_KEY>`. Sending requires `x-gdg_email-apikey: <project-key>`.

Open `http://localhost:8080/swagger/index.html` for interactive Swagger UI. The generated Swagger 2.0 document is available at `http://localhost:8080/swagger/doc.json`; the generated YAML and JSON are also checked in under `internal/delivery/http/docs`. The Swagger UI assets are bundled with the Go dependencies, so the docs work without a CDN.

In Swagger UI, use **Authorize** to enter `ADMIN_API_KEY` for management operations or the project API key for `POST /api/email`.

The API docs come from Swag annotations on the Gin handlers. After changing a handler or request/response model, regenerate the checked-in files:

```sh
go install github.com/swaggo/swag/cmd/swag@v1.16.6
swag init -g cmd/server/main.go --parseInternal -o internal/delivery/http/docs --generatedTime=false
```

| Method | Path | Action |
| --- | --- | --- |
| POST / GET | `/api/projects` | Create / list projects |
| GET / PUT / DELETE | `/api/projects/:projectId` | Get / update / soft-delete project |
| POST / GET | `/api/projects/:projectId/templates` | Create / list templates |
| GET / PUT / DELETE | `/api/projects/:projectId/templates/:id` | Get / update / soft-delete template |
| POST / GET | `/api/projects/:projectId/api-keys` | Generate key / list metadata |
| DELETE | `/api/projects/:projectId/api-keys/:id` | Revoke key |
| POST | `/api/email` | Queue email |

Responses follow `{"message":"...","data":...}`. Creates return 201, management reads/updates/deletes return 200, and accepted emails return 202. Errors use 400 (input), 401 (authentication), 404 (missing/inaccessible resource), 409 (conflict), 503 (dependency unavailable), or 500 (unexpected failure).

### Create a project

```sh
curl -X POST http://localhost:8080/api/projects \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"name":"GDG","description":"Community emails"}'
```

Use `data.id` as `PROJECT_ID`. Project PUT accepts the same body; it replaces name and description.

### Create a template

```sh
curl -X POST "http://localhost:8080/api/projects/$PROJECT_ID/templates" \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"name":"welcome","subject":"Welcome, {{ .name }}","content":"<h2>Hello {{ .name }}</h2><p>Welcome to GDG.</p>","data":{"name":"Example"}}'
```

Use `data.id` as `TEMPLATE_ID`. Template PUT uses the same body. Variables are case-sensitive string-map keys. The saved `data` map is example metadata; each send supplies its own complete runtime data. Missing referenced keys fail validation. HTML values are escaped, and subject values cannot inject headers.

### Generate a project key

```sh
curl -X POST "http://localhost:8080/api/projects/$PROJECT_ID/api-keys" \
  -H "X-Admin-Key: $ADMIN_API_KEY"
```

Save `data.key` as `PROJECT_API_KEY`: it is returned once. Only its SHA-256 hash is stored. Key lists contain metadata, not secrets or hashes.

### Submit an email

```sh
curl -X POST http://localhost:8080/api/email \
  -H "x-gdg_email-apikey: $PROJECT_API_KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"to\":\"recipient@example.com\",\"template\":\"$TEMPLATE_ID\",\"data\":{\"name\":\"Alice\"}}"
```

Response:

```json
{"message":"Email accepted","data":{"id":"<email UUID>","status":"PENDING"}}
```

The template must belong to the authenticated key's project. One recipient is accepted per request; request bodies are limited to 1 MiB.

## Delivery behavior

- Persist rendered subject/body as `PENDING`, then enqueue the email ID. Return 202 only after both succeed.
- Set `PROCESSING` when delivery starts. One initial SMTP attempt plus three retries uses 5, 15, and 45 second delays.
- SMTP acceptance sets `SENT` and `sent_at`; exhaustion sets `FAILED`. Sent means the SMTP server accepted the message, not inbox delivery.
- Persist attempts and next retry time. Recover abandoned Redis entries after five minutes idle; acknowledge and remove entries after terminal state is saved.
- A Redis publish failure returns 503 and attempts to mark the row failed. There is no transactional outbox: a crash between database save and enqueue can leave a pending row without a queue entry. Inspect the database to investigate such rows.
- Delivery is at least once. SMTP acceptance followed by a crash before status persistence can cause a duplicate. Client resubmissions are separate emails.
- Project/template deletion is soft. Project deletion revokes its keys. Already accepted emails retain their rendered content and can finish delivery. Deleted names remain reserved by unique constraints.
- One internal owner is bootstrapped for project ownership. No registration/login or email-status endpoint is included. Run one app instance for this MVP.

Inspect recent delivery status locally:

```sh
docker compose exec postgres psql -U postgres -d email_service \
  -c 'SELECT id, status, attempts, last_error, sent_at FROM emails ORDER BY created_at DESC LIMIT 20;'
```

## Tests

Unit tests and local SMTP tests require no provider credentials:

```sh
go test ./...
```

The integration suite uses disposable PostgreSQL/Redis services and a local SMTP sink. It exercises CRUD, authentication, project isolation, JSON persistence, migrations, enqueue failure, delivery, and abandoned-job recovery.

```sh
docker compose -p gdg-email-mvp-test -f docker-compose.test.yml up -d --wait
export TEST_DB_PORT=$(docker compose -p gdg-email-mvp-test -f docker-compose.test.yml port postgres 5432 | cut -d: -f2)
export TEST_REDIS_URL="redis://$(docker compose -p gdg-email-mvp-test -f docker-compose.test.yml port redis 6379)/0"
go test -race ./...
docker compose -p gdg-email-mvp-test -f docker-compose.test.yml down
```

The integration database is fixed to `email_service_test` with disposable test credentials. Never point these test variables at production services.
