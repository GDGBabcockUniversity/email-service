# GDG Email Service

[![Keyway Secrets](https://www.keyway.sh/badge.svg?repo=GDGBabcockUniversity/email-service)](https://www.keyway.sh/vaults/GDGBabcockUniversity/email-service)

Go/Gin HTTP API, PostgreSQL/GORM, Redis Streams, and SMTP. The API and one email consumer run together.

## Run

Copy `.env.example` to `.env` and fill in `DB_PASSWORD`, `ADMIN_API_KEY`, `UNSUBSCRIBE_SECRET`, and the SMTP settings. Credentials are not included.

```sh
cp .env.example .env
# Edit .env with your configuration.
docker compose up --build -d
docker compose logs -f app
```

SMTP uses `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `SMTP_FROM`. Set `SMTP_TLS_MODE=starttls` for STARTTLS (default port 587), or `tls` for implicit TLS (set port 465). Supply your Gmail app password or Brevo SMTP credentials and an authorized sender address. `none` is supported only with empty SMTP username/password for local test servers.

`UNSUBSCRIBE_SECRET` (`openssl rand -hex 32`) signs the unsubscribe tokens, and `PUBLIC_BASE_URL` is the origin used to build the unsubscribe links inside emails. Emails link to `<PUBLIC_BASE_URL>/unsubscribe/<token>`, a confirmation page served by the public website rather than by this service, so set it to that site's origin in production — a `localhost` value passes validation but ships links no recipient can open. Rotating the secret invalidates every unsubscribe link already in recipients' inboxes.

Compose supplies the database hostname and local Redis URL. To use Upstash, set `REDIS_URL` to its Redis protocol `rediss://...` URL, not its REST URL. PostgreSQL and Redis data use named volumes.

For execution outside Docker, export the environment variables before running `go run ./cmd/server`; the binary does not load `.env` itself. Go 1.27.1 is required by the existing module.

## Admin dashboard

The service serves a browser console at `http://localhost:8080/admin`. It tracks delivery
progress, sends a project's templates, manages each project's audience, and edits email
templates. Sign in with the same
`ADMIN_API_KEY` the management API uses; the key is kept in the browser's local storage and
sent as `X-Admin-Key` on every request, so the page itself holds no secrets and serves
unauthenticated.

The front end lives in `web/` and is plain HTML, CSS, and JavaScript with no build step:

```
web/
  embed.go              Go embed directives and the fs.FS the router mounts
  index.html            the single page
  static/css/           stylesheet
  static/js/            dashboard logic
  static/img/           GDG Babcock logo (light variant for dark surfaces) and favicon
```

`go:embed` compiles the whole tree into the binary, so the dashboard ships with the service
and needs no extra files at runtime. Editing anything under `web/` takes effect on the next
`go build`.

The console follows the chapter website's dark surface language. `gdg-logo.png` is the original
lockup; `gdg-logo-light.png` is the same artwork with its black wordmark inverted to white so it
reads on the dark background, and is what the page loads. The four Google colours are reserved
for delivery status, so a colour always means the same thing.

## API

Management requests require `X-Admin-Key: <ADMIN_API_KEY>`. Sending requires `x-gdg_email-apikey: <project-key>`.

Open `http://localhost:8080/swagger/index.html` for interactive Swagger UI. The generated Swagger 2.0 document is available at `http://localhost:8080/swagger/doc.json`; the generated YAML and JSON are also checked in under `cmd/server/docs`. The Swagger UI assets are bundled with the Go dependencies, so the docs work without a CDN.

In Swagger UI, use **Authorize** to enter `ADMIN_API_KEY` for management operations or the project API key for `POST /api/email`.

The API docs come from Swag annotations on the Gin handlers. After changing a handler or request/response model, regenerate the checked-in files:

```sh
go install github.com/swaggo/swag/cmd/swag@v1.16.6
swag init -g cmd/server/main.go -o cmd/server/docs
```

| Method | Path | Action |
| --- | --- | --- |
| POST / GET | `/api/projects` | Create / list projects |
| GET / PUT / DELETE | `/api/projects/:projectId` | Get / update / soft-delete project |
| POST / GET | `/api/projects/:projectId/templates` | Create / list templates |
| GET / PUT / DELETE | `/api/projects/:projectId/templates/:id` | Get / update / soft-delete template |
| POST / GET | `/api/projects/:projectId/api-keys` | Generate key / list metadata |
| DELETE | `/api/projects/:projectId/api-keys/:id` | Revoke key |
| POST / GET | `/api/projects/:projectId/audience` | Add recipients / list audience |
| DELETE | `/api/projects/:projectId/audience/:id` | Remove a recipient |
| POST | `/api/projects/:projectId/emails` | Send a template with the admin key |
| GET | `/api/projects/:projectId/emails` | List recent emails (`status`, `limit` query params) |
| GET | `/api/projects/:projectId/emails/stats` | Count emails per delivery status |
| POST | `/api/email` | Queue email |
| GET / POST | `/api/unsubscribe?token=...` | Unsubscribe a recipient (public, no key) |
| POST | `/api/resubscribe?token=...` | Resubscribe the same recipient (public, no key) |

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

Sends reference a template by its `name` (unique within the project), so keep the name above as `TEMPLATE_NAME`. `GET`, `PUT`, and `DELETE /templates/:id` address the template by its `data.id` UUID instead. Variables are case-sensitive string-map keys. The saved `data` map is example metadata; each send supplies its own complete runtime data. Missing referenced keys fail validation. HTML values are escaped, and subject values cannot inject headers.

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
  -d "{\"to\":\"recipient@example.com\",\"template\":\"$TEMPLATE_NAME\",\"data\":{\"name\":\"Alice\"}}"
```

Response:

```json
{"message":"Email accepted","data":{"id":"<email UUID>","status":"PENDING"}}
```

`template` is the template's name, resolved within the authenticated key's project, so two projects can reuse the same template name without reaching each other's content. `POST /api/projects/:projectId/emails` queues the same way but authenticates with the admin key instead of a project key, which is how the dashboard sends without holding a project key in the browser. Set `to` to `all` to send to every address in the project's audience; otherwise one recipient is accepted per request. Request bodies are limited to 1 MiB.

### Unsubscribes

Every audience entry gets one unsubscribe token, derived from `UNSUBSCRIBE_SECRET` plus the project and address. Only its SHA-256 hash is stored; the token itself exists in the recipient's link. Templates receive the recipient's link as `{{ .UnsubscribeURL }}`:

```sh
curl -X POST "http://localhost:8080/api/projects/$PROJECT_ID/templates" \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"name":"news","subject":"GDG news","content":"<p>Hello.</p><p><a href=\"{{ .UnsubscribeURL }}\">Unsubscribe</a></p>"}'
```

Every template gets `<p><a href="{{ .UnsubscribeURL }}">Unsubscribe</a></p>` appended to its body on create and update, unless the body already references `{{ .UnsubscribeURL }}` somewhere — move it to place the link yourself. This happens in the service, so it applies to the dashboard and the API alike; there is no way to store a template without the link.

Clicking the link opens `<PUBLIC_BASE_URL>/unsubscribe/<token>`, a page on the public website that asks the reader to confirm before anything changes, so a link prefetched by a mail client never opts someone out on its own. The token rides in the path because an ESP's click tracking can drop the query string when it rewrites the link — Brevo does — which would strand the reader on the page with nothing identifying their subscription. The site still accepts `/unsubscribe?token=...` for links already sitting in inboxes. That page posts the token back to `POST /api/unsubscribe?token=...` here, which hashes the token, finds the subscription it belongs to, and sets `subscribed` to false:

```json
{"message":"Unsubscribed","data":{"email":"recipient@example.com","project_id":"<project UUID>","subscribed":false}}
```

The audience entry stays in place. `to: "all"` sends only to subscribed addresses, and a single-recipient send to an address that unsubscribed is skipped the same way: the call still returns 202 with an empty `data` list, since honouring the opt-out is the request succeeding and there is nothing for the caller to retry. A single-recipient send to an address that is not on the audience yet adds it (subscribed, with its token hash) before rendering, so the recipient gets a working unsubscribe link rather than an empty one.

`POST /api/resubscribe?token=...` turns the same entry back on, and re-adding the address through `POST /api/projects/:projectId/audience` does the same: an address already on the list keeps its row and only has `subscribed` set back to true, so a project never collects a second entry for one address. Repeating either call is harmless; an unknown token returns 404. Both endpoints are public by design, since recipients click them from their inbox with no credentials.

## Delivery behavior

- Persist rendered subject/body as `PENDING`, then enqueue the email ID. Return 202 only after both succeed.
- Set `PROCESSING` when delivery starts. One initial SMTP attempt plus three retries uses 5, 15, and 45 second delays.
- SMTP acceptance sets `SENT` and `sent_at`; exhaustion sets `FAILED`. Sent means the SMTP server accepted the message, not inbox delivery.
- Persist attempts and next retry time. Recover abandoned Redis entries after five minutes idle; acknowledge and remove entries after terminal state is saved.
- A Redis publish failure returns 503 and attempts to mark the row failed. There is no transactional outbox: a crash between database save and enqueue can leave a pending row without a queue entry. Inspect the database to investigate such rows.
- Delivery is at least once. SMTP acceptance followed by a crash before status persistence can cause a duplicate. Client resubmissions are separate emails.
- Project/template deletion is soft. Project deletion revokes its keys. Already accepted emails retain their rendered content and can finish delivery. Deleted names remain reserved by unique constraints.
- One internal owner is bootstrapped for project ownership. There is no registration/login; the admin key is the only management credential. Run one app instance for this MVP.

Delivery status is visible on the dashboard at `/admin`, or straight from the database:

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
