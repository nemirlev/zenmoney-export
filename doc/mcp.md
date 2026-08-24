# Financial analytics MCP server

`zenmcp` is the read-only financial analytics runtime for the PostgreSQL database populated by
`zenexport`. It is a separate binary: it does not synchronize data, run migrations, or require a
ZenMoney API token.

For design decisions and exact report semantics, see [MCP architecture](mcp-architecture.md).

## Prerequisites

- PostgreSQL 15 through 18 with this checkout's migrations applied.
- At least one completed `zenexport sync` so the database contains the user, instruments, accounts,
  and transactions to analyze.
- A PostgreSQL connection URL. For production, use a dedicated role with `SELECT` only and TLS
  verification.

`zenmcp` never reads `ZEN_API_TOKEN`. `ZENMCP_DB_URL` is preferred over the shared `DB_URL` so the
MCP and synchronizer can use different database roles.

## Build

```bash
go build -o zenmcp ./cmd/zenmcp
```

The binary serves stateless Streamable HTTP; it is not a stdio MCP server.

## Local loopback mode

Local mode maps every request to the ZenMoney users discovered in the configured database and is
restricted to a loopback listen address. It is intended for one trusted developer on one machine.
`ZENMCP_USER_IDS` may narrow that catalog as a defense-in-depth allowlist, but is not required.

```bash
ZENMCP_DB_URL='postgres://mcp_reader:password@127.0.0.1:5432/zenmoney?sslmode=disable' \
ZENMCP_AUTH_MODE=local \
go run ./cmd/zenmcp
```

The disabled database TLS in this loopback example is for local development only. Use
`sslmode=verify-full` and a trusted CA for a remote database.

Defaults:

- MCP endpoint: `http://127.0.0.1:8080/mcp`
- liveness: `http://127.0.0.1:8080/healthz`
- readiness: `http://127.0.0.1:8080/readyz`

The liveness endpoint proves the process can answer HTTP. Readiness also pings PostgreSQL. Neither
health endpoint requires MCP authentication.

Do not set `ZENMCP_LISTEN_ADDRESS=0.0.0.0:...` in local mode; startup rejects non-loopback local
authentication.

## Connect a local MCP client

MCP client configuration varies by host. Configure a Streamable HTTP server whose URL is
`http://127.0.0.1:8080/mcp`; do not configure a command/stdio transport.

The official MCP Inspector can exercise the modern `2026-07-28` protocol. Save this as
`mcp-inspector.json`:

```json
{
  "mcpServers": {
    "zenmoney": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "protocolEra": "modern",
      "advertisedExtensions": {
        "io.modelcontextprotocol/ui": {
          "mimeTypes": ["text/html;profile=mcp-app"]
        }
      }
    }
  }
}
```

List tools:

```bash
npx @modelcontextprotocol/inspector --cli \
  --config ./mcp-inspector.json --server zenmoney --method tools/list
```

Call a report. Public `from` and `to` dates are both inclusive, and currency is deliberately not an
input—it comes from the selected ZenMoney users' profiles:

```bash
npx @modelcontextprotocol/inspector --cli \
  --config ./mcp-inspector.json --server zenmoney \
  --method tools/call --tool-name get_spending_summary \
  --tool-args-json '{"period":{"from":"2026-07-01","to":"2026-07-31","timezone":"Europe/Moscow"},"limit":20}'
```

Run the Inspector web client with the same configuration to inspect the embedded MCP App. Only
`render_finance_chart` has a UI resource; the five data tools intentionally remain analysis-first.

```bash
npx @modelcontextprotocol/inspector --config ./mcp-inspector.json
```

Inspector behavior evolves independently; consult the official
[MCP Inspector configuration reference](https://github.com/modelcontextprotocol/inspector/blob/main/docs/mcp-server-configuration.md)
if its command-line syntax changes.

## Remote bearer mode

Bearer mode is the safe bootstrap mode for a remote single principal. It still requires HTTPS; the
built-in server is plain HTTP, so terminate TLS at a trusted reverse proxy or load balancer. The
secret must contain at least 32 bytes; for example, generate 32 random bytes as 64 hexadecimal
characters with `openssl rand -hex 32`.

```bash
ZENMCP_DB_URL='postgres://mcp_reader:password@db.internal:5432/zenmoney?sslmode=verify-full' \
ZENMCP_LISTEN_ADDRESS='0.0.0.0:8080' \
ZENMCP_AUTH_MODE=bearer \
ZENMCP_BEARER_TOKEN='replace-with-a-long-random-secret' \
./zenmcp
```

Every MCP request must carry exactly:

```http
Authorization: Bearer replace-with-a-long-random-secret
```

The configured token is never included in errors or logs. The resolver stores a digest and compares
authorization values in constant time. Keep the token in a secret manager or protected environment
file, rotate it like any other access credential, and never put it in a client configuration that
will be committed. One token maps to one configured principal; by default that principal can
discover every ZenMoney user in the connected database. Set the optional `ZENMCP_USER_IDS`
allowlist when the database contains users that this deployment must never expose. This mode is
not multi-user OAuth.

For a browser-based MCP host, allow its exact origin (scheme, hostname, and optional port):

```bash
ZENMCP_ALLOWED_ORIGINS='https://assistant.example.com,https://admin.example.com'
```

Origin validation is defense in depth and does not replace bearer authentication. Non-browser MCP
clients normally send no `Origin`. Do not use `*`; only exact HTTP(S) origins are accepted.

A remote Inspector entry looks like this:

```json
{
  "mcpServers": {
    "zenmoney": {
      "type": "http",
      "url": "https://finance.example.com/mcp",
      "protocolEra": "modern",
      "headers": {
        "Authorization": "Bearer replace-with-a-long-random-secret"
      }
    }
  }
}
```

## Configuration reference

All `zenmcp` configuration is environment-based and independent from the `zenexport` CLI config.

| Variable | Default | Meaning |
| --- | --- | --- |
| `ZENMCP_DB_URL` | fallback `DB_URL` | Required PostgreSQL URL. Prefer a read-only role. |
| `ZENMCP_LISTEN_ADDRESS` | `127.0.0.1:8080` | HTTP host and port. Local auth requires loopback. |
| `ZENMCP_ENDPOINT` | `/mcp` | Clean absolute MCP path, distinct from health paths. |
| `ZENMCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. |
| `ZENMCP_AUTH_MODE` | `local` | `local` or `bearer`. |
| `ZENMCP_USER_IDS` | none | Optional comma-separated numeric allowlist that restricts which database users can be selected. Omit it to expose the full database user catalog. |
| `ZENMCP_BEARER_TOKEN` | none | Required only in bearer mode. |
| `ZENMCP_ALLOWED_ORIGINS` | none | Comma-separated exact trusted HTTP(S) browser origins. |
| `ZENMCP_REPORT_TIMEZONE` | `UTC` | Default IANA timezone for periods without one. |
| `ZENMCP_MAX_PERIOD_DAYS` | `3660` | Maximum inclusive report period. |
| `ZENMCP_DEFAULT_PAGE_SIZE` | `50` | Default bounded row/page size. |
| `ZENMCP_MAX_PAGE_SIZE` | `100` | Maximum spending/budget/search row size. |
| `ZENMCP_MAX_CHART_POINTS` | `400` | Cashflow and rendered chart point limit. |
| `ZENMCP_MAX_FILTER_VALUES` | `100` | Maximum IDs in each filter list. |
| `ZENMCP_MAX_REQUEST_BODY_BYTES` | `1048576` | HTTP request body limit. |
| `ZENMCP_STALE_AFTER` | `24h` | Duration after which the latest completed sync is stale. |
| `ZENMCP_REQUEST_TIMEOUT` | `30s` | Positive Go duration; cancellation propagates to report queries. |

Positive limits are validated at startup; the default page size cannot exceed the maximum. Unknown
auth modes, bearer secrets shorter than 32 bytes, invalid timezones, unsafe endpoints, malformed
origins, and invalid allowlist IDs fail closed before the server listens. Page and chart limits cannot
exceed the PostgreSQL adapter's hard safety ceiling of 500 rows/points.

## User catalog and report scope

With no `ZENMCP_USER_IDS`, the authenticated principal can use every ZenMoney user stored in the
database. `get_data_freshness` returns that catalog in `users` and `metadata.users`, and every
report repeats its effective users in `metadata.users`. Public stable keys have the form
`user:<numeric-id>`; use those keys in an optional request selector such as:

```json
{"users":{"userIds":["user:12345"]}}
```

The `users.userIds` field can only narrow the authenticated catalog; an unknown or disallowed key
is rejected. An empty selector means the complete available catalog. `get_data_freshness` itself
has no selector because it is the discovery entry point.

One aggregate report can combine several selected users only when all of them have the same
`user.currency`. When currencies differ, Codex should group the catalog by currency and make a
separate tool call for each user or same-currency group. The server never converts multiple primary
report currencies into an arbitrarily chosen common currency.

## Tool overview

- `get_spending_summary` — authoritative spending totals plus a bounded category list; optionally
  narrow the user catalog with `users.userIds`; inspect
  `hasMore`/`truncated` before describing the list as complete.
- `get_cashflow` — income, outcome, and net by day/week/month, optionally for selected users.
- `get_budget_progress` — budget versus allocated spending for complete calendar months; totals
  remain authoritative when the bounded rows report `hasMore`/`truncated`; user selection is
  optional.
- `search_transactions` — bounded cursor-paginated detail search, optionally for selected users.
- `get_data_freshness` — database-scoped local synchronization status, staleness, and the available
  user catalog; it does not claim per-user sync provenance or expose database-wide processed-record
  counts.
- `render_finance_chart` — re-executes a report and applies a safe declarative ChartSpec; its
  nested report can select users, and its optional daily cashflow comparison derives and executes
  the immediately preceding equal-length period instead of calculating a cosmetic series.

Each data result contains typed `structuredContent`, calculation metadata, and a compact text/table
fallback. `render_finance_chart` additionally references the embedded
`ui://zenmoney/finance-chart` resource. See [MCP architecture](mcp-architecture.md) for exact
contracts and calculation policies.

## Container deployment

### Docker Compose and local Codex

The MCP Compose file contains only `mcp`. It joins the existing exporter Compose network and uses
the PostgreSQL service, migrations, synchronized data, and persistent volume managed by
`docker/docker-compose.postgres.yml`; it never duplicates or owns them. From the repository root,
create an ignored local environment file, replace every credential placeholder, and generate a
random bearer secret:

```bash
cp -n docker/.env.example docker/.env
openssl rand -hex 32
# Edit docker/.env and paste the generated value into ZENMCP_BEARER_TOKEN.
```

Start the exporter stack first so it creates PostgreSQL and its default `docker_default` network.
After migrations and synchronization are available, start the independent `zenmcp` project:

```bash
docker compose --env-file docker/.env -f docker/docker-compose.postgres.yml up -d
docker compose --env-file docker/.env -f docker/docker-compose.mcp.yml up -d --build
docker compose --env-file docker/.env -f docker/docker-compose.mcp.yml ps
curl --fail --silent --show-error http://127.0.0.1:8080/healthz
curl --fail --silent --show-error http://127.0.0.1:8080/readyz
```

The exporter Compose project defaults to the name `docker`, so its default network is
`docker_default`. If it was started with a different project name, set
`ZENEXPORT_DOCKER_NETWORK=<project>_default` before starting MCP. The external network must already
exist; a missing network is an intentional startup error. `DB_URL` must remain an internal Docker
URL whose host is the exporter service name `postgres`.

If `ZENMCP_PORT` was changed, use that host port in the URLs. The MCP Compose file publishes only
the MCP port and binds it to `127.0.0.1`; PostgreSQL port and volume policy remain in the exporter
Compose file. On macOS, Codex runs on the host, so use `http://127.0.0.1:8080/mcp`. Container DNS
names such as `postgres` work only on the shared Docker network.

The server secret and the Codex client variable are related but live in different processes:

- `ZENMCP_BEARER_TOKEN` in `docker/.env` is injected into the server container. It is the secret
  against which requests are authenticated.
- `ZENMCP_CLIENT_BEARER_TOKEN` below is an example client-side variable. Its value must exactly
  match the server secret. The name is arbitrary and does not need to match the server variable.
- `--bearer-token-env-var` stores the client variable's **name**, not its secret value, in the
  shared Codex MCP configuration. It does not read the Compose environment automatically.

Export the matching value in the shell that launches Codex CLI, then register the Streamable HTTP
server:

```bash
export ZENMCP_CLIENT_BEARER_TOKEN='paste-the-same-secret-used-by-the-server'
codex mcp add zenmoney-local \
  --url http://127.0.0.1:8080/mcp \
  --bearer-token-env-var ZENMCP_CLIENT_BEARER_TOKEN
codex mcp list
codex mcp get zenmoney-local --json
```

Do not commit either token, put a literal token in `~/.codex/config.toml`, or reuse it for another
service. A shell `export` is visible only to processes descended from that shell. In particular, a
Codex desktop app opened from Finder on macOS may not inherit it. Make the variable available to
the desktop launch environment using your normal secret-management mechanism, then fully quit and
reopen Codex. For a development-only launch environment, macOS also provides
`launchctl setenv ZENMCP_CLIENT_BEARER_TOKEN '<matching-secret>'`; remove it later with
`launchctl unsetenv ZENMCP_CLIENT_BEARER_TOKEN`.

Codex CLI and desktop share MCP configuration, but an already running session may retain its
original tool list. Start a new task after registration. In desktop, Settings > MCP servers can
also restart the server connection; fully relaunch the app after changing its environment. In CLI
or the TUI, use `/mcp` to inspect the live connection, then ask Codex to call
`get_data_freshness` and a report such as `get_spending_summary`.

Only `render_finance_chart` advertises the MCP App resource. Embedded UI rendering depends on the
Codex host and build; if the interactive chart does not appear, the validated structured result
and compact text/table fallback are still usable. The five data tools intentionally have no
embedded UI. See the official [Codex MCP setup guide](https://learn.chatgpt.com/docs/extend/mcp)
for the shared configuration model and current Streamable HTTP support.

Stop only MCP:

```bash
docker compose --env-file docker/.env -f docker/docker-compose.mcp.yml down
```

Because `zenmcp` is a separate Compose project and `docker_default` is external, this command does
not stop PostgreSQL, remove the exporter network, or touch `docker_postgres_data`. Manage those only
through `docker/docker-compose.postgres.yml`. Do not use `--remove-orphans` across the two projects.

### Standalone image

Build the MCP image independently from the synchronizer:

```bash
docker build -f docker/Dockerfile.zenmcp -t zenmcp:local .
```

Run it with bearer authentication. In production, inject the database URL and bearer token through
your platform's secrets facility instead of writing them on the command line:

```bash
docker run --rm -p 8080:8080 \
  -e ZENMCP_DB_URL='postgres://mcp_reader:password@database:5432/zenmoney?sslmode=verify-full' \
  -e ZENMCP_LISTEN_ADDRESS='0.0.0.0:8080' \
  -e ZENMCP_AUTH_MODE=bearer \
  -e ZENMCP_BEARER_TOKEN='replace-with-a-long-random-secret' \
  zenmcp:local
```

The image contains only the `zenmcp` binary and CA certificates, runs as a non-root user, exposes
port 8080, and checks `/healthz`. It is independent of the existing `zenexport` image and entrypoint.
Database migrations and synchronization remain separate deployment responsibilities.

## Health, readiness, and shutdown

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

- `/healthz` returns success when the process and HTTP mux are alive.
- `/readyz` returns `503` until PostgreSQL answers a ping.
- the MCP endpoint is request-size limited and propagates client cancellation to report queries.
- `ZENMCP_REQUEST_TIMEOUT` adds a server-side deadline to every MCP request and cancels in-flight
  PostgreSQL work when it expires.
- the HTTP server also enforces a fixed 15-second request-read timeout before handler execution.
- `SIGINT` and `SIGTERM` stop accepting traffic, allow in-flight HTTP work up to the ten-second
  shutdown deadline, and close the PostgreSQL pool.

During deployment, route traffic only to ready instances and set the orchestrator termination grace
period above ten seconds.

## Troubleshooting and limitations

- `401 unauthenticated`: bearer header is missing or not an exact match.
- `403 invalid_identity`: the resolver produced no subject or produced an inconsistent all-users /
  allowlist scope.
- `403` before authentication in a browser: add the exact host origin to
  `ZENMCP_ALLOWED_ORIGINS`; do not disable origin protection globally.
- `/readyz` returns `503`: verify database routing, TLS, credentials, migrations, and PostgreSQL
  version.
- Compose reports that `docker_default` is missing: start the exporter Compose project first, or
  set `ZENEXPORT_DOCKER_NETWORK` to the actual `<project>_default` network name.
- currency/rate error: all users selected for one report must share one primary currency and every
  required current instrument rate must be positive. Use `get_data_freshness` to discover the
  catalog and have Codex issue separate `users.userIds` calls for different currencies.
- stale data: run or repair `zenexport sync`; `zenmcp` never calls ZenMoney itself.
- a report changed between analysis and rendering: rendering intentionally re-runs the report, so a
  synchronization completed between the two calls can change authoritative values.
- a budget period is rejected: use complete calendar months (first day through last day,
  inclusive); the aggregate `category:all` budget is distinct from uncategorized and takes
  precedence for totals when present.

Reports use current synchronized currency rates rather than historical rates; calendar dates have
no original timezone; budget forecasts are limited to stored budget rows; multi-tag amounts are
split equally across canonical root categories; and bearer mode is a single-principal bootstrap.
Every individual report is read in one repeatable-read PostgreSQL snapshot. Separate report calls,
including the two reads used by a previous-period comparison, may observe different completed
synchronizations. These limitations are described in more detail in
[MCP architecture](mcp-architecture.md).
