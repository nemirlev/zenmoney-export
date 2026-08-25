# MCP financial analytics architecture

## Decision summary

The financial analytics MCP server lives in this repository because this project owns the
PostgreSQL schema, migrations, ZenMoney synchronization model, and compatibility tests that define
the meaning of the stored data. Keeping the read model next to those artifacts lets schema and
analytics changes be reviewed and tested together.

`zenmcp` is nevertheless a separate executable and runtime component from `zenexport`:

- `zenexport` owns write-side synchronization with the ZenMoney API and requires a ZenMoney token;
- `zenmcp` is read-only, requires only PostgreSQL, and serves authenticated analytics requests;
- either component can be deployed, restarted, scaled, or granted database permissions without
  affecting the other;
- a slow model/tool request cannot delay synchronization, and a synchronization failure cannot
  corrupt MCP transport state.

The recommended production database role for `zenmcp` has `SELECT` access only. The MCP server
does not run migrations and does not need `ZEN_API_TOKEN`.

## Component boundaries

```text
MCP client / MCP App
        |
        | MCP 2026-07-28, stateless Streamable HTTP
        v
internal/mcpserver
  transport, authentication boundary, typed tools, ui:// resource
        |
        | authenticated analytics.Principal + typed request
        v
internal/analytics
  validation, normalization, report semantics, ChartSpec, table fallback
        |
        | small read-only AnalyticsStore interface
        v
internal/db/postgres
  tenant-scoped SQL, aggregation, currency conversion, keyset pagination
        |
        v
PostgreSQL populated by zenexport
```

The boundaries are intentional:

- PostgreSQL code knows the schema and performs set-based aggregation, but does not know MCP.
- `internal/analytics` owns public report semantics and rejects unsafe or oversized requests. It
  depends on the small `AnalyticsStore` interface rather than the existing write-side storage API.
- `internal/mcpserver` maps the typed service methods to MCP tools. Request `users.userIds` values
  are stable selection keys that can only narrow the authenticated catalog; they never establish
  identity or expand authorization.
- the embedded UI renders only the server-produced `structuredContent`. It has no database or
  network access and cannot change report values.

### Local container topology

`docker/docker-compose.mcp.yml` contains only the read-only `mcp` service and uses the separate
Compose project name `zenmcp`. It joins the exporter project's existing default network as an
external network (`docker_default` by default, configurable with `ZENEXPORT_DOCKER_NETWORK`). The
exporter Compose project remains the sole owner of PostgreSQL, migrations, synchronization, its
network lifecycle, and `docker_postgres_data`. Consequently, stopping or removing the MCP project
cannot delete the database volume or stop the writer stack.

## Public tool contracts

The official Go SDK derives `inputSchema` and `outputSchema` from the Go DTOs and validates tool
input and output. Every successful data tool returns structured JSON plus a short text summary and
an accessible table fallback. Financial decimals are JSON strings, not binary floating-point
numbers.

All report periods use ISO `YYYY-MM-DD` dates. Both public `from` and `to` are inclusive. The
domain service converts this to an equivalent internal `[from, day-after-to)` range. An optional
IANA timezone is validated; the configured default is used when it is absent.

The four financial data requests accept an optional `users.userIds` selection using stable
`user:<numeric-id>` keys. An empty selection means the complete authenticated catalog. Every report
returns its effective catalog in `metadata.users`; `get_data_freshness` returns the available
catalog so a client can discover keys before requesting a narrower report.

### `get_spending_summary`

Input: `period`, optional account/category/merchant filters, `includeHold`, and a category `limit`.
Output: authoritative total spending and transaction count, a bounded category list with stable IDs
and shares, `hasMore`/`truncated` disclosure, metadata, and table fallback. The totals cover the
whole filtered report even when the category list is truncated.

### `get_cashflow`

Input: `period`, optional filters, and optional `day`, `week`, or `month` granularity. If omitted,
the service chooses a granularity that fits the point limit. Output: ordered buckets containing
income, outcome, and net values plus totals.

### `get_budget_progress`

Input: a period spanning complete calendar months, optional category/hold filters, and row `limit`.
The inclusive `from` must be the first day of a month and `to` the last day of a month. Output:
authoritative budget, spent, remaining, and optional percentage totals plus a bounded canonical
category list with `hasMore`/`truncated` disclosure. A zero budget produces no percentage.

### `search_transactions`

Input: `period`, optional filters, bounded free-text search, opaque cursor, and `pageSize`. Output:
a bounded page with a stable keyset cursor. Results expose report-relevant identifiers and labels,
not database credentials or ZenMoney tokens.

### `get_data_freshness`

Input: an empty object. Output: latest completed sync, latest attempted sync, age, stale flag, and
the authenticated analytics user catalog. The explicit freshness scope is `database`: the current
schema has no reliable per-user synchronization provenance, so the server does not imply that a
run belongs to any catalog user and does not expose its database-wide processed-record count.
Freshness describes the local PostgreSQL snapshot, not the age of ZenMoney exchange rates.

### `render_finance_chart`

Input has two parts:

- `report`: a discriminated report request containing `kind` and exactly one matching request;
- `chart`: a validated declarative `ChartSpec`.

The renderer normalizes and executes the report again for the authenticated principal, then applies
presentation-only sorting, series selection, top-N, and Other-bucket aggregation. Caller-supplied
rows, amounts, JavaScript, HTML, SQL, executable expressions, and URLs are never accepted.

## ChartSpec

The supported chart types are `bar`, `horizontal_bar`, `line`, `area`, `donut`, `stacked_bar`, and
`grouped_bar`. A spec selects a report-compatible dimension and one to four known series, and can
set title, subtitle, sorting, top-N, Other bucket, stacking, legend, tooltip, negative-value
display, comparison periods, granularity, value format, and table caption.

Validation enforces report-compatible dimensions and series. Donut charts have exactly one series;
line and area charts require the period dimension; grouped and stacked bars require multiple
series. Stacked bars require normal stacking. All user-facing strings are length- and character-
checked and rejected when they resemble markup, URLs, scripts, SQL, or executable expressions.
`comparisonPeriods` is a deliberately narrow cashflow contract. It requires a chronological daily
line chart with exactly the net series; the service derives an immediately preceding period with
the same number of calendar days, re-executes both periods, and returns explicit current/previous
net series plus their period and alignment metadata. Top-N, Other grouping, and value recalculation
are forbidden, so presentation cannot rewrite either authoritative time series.

The LLM controls presentation, not financial semantics. Values returned to the UI always come from
the freshly executed authoritative report.

## Financial semantics

The rules below are returned in report metadata as well as enforced in SQL/domain code.

### Tenant and account scope

Every store call receives an authenticated `Principal`. With no `ZENMCP_USER_IDS`, the bootstrap
identity can discover every ZenMoney user in the database. The optional environment allowlist is an
authorization ceiling; a request-level `users.userIds` list can narrow it further but cannot add an
unknown or disallowed user. Every report query then scopes transaction, account, tag, merchant,
budget, and user joins to that explicit effective set. Aggregate reports include only accounts with
`in_balance = true`; the official ZenMoney API defines such accounts as participating in the total
balance and income/expense reports.

### Dates and timezones

Public date bounds are inclusive. They are normalized to an internal half-open range to avoid
off-by-one errors. ZenMoney transactions are stored as calendar dates, not original timestamps
with timezone, so timezone selection determines validation/default boundaries but cannot recover an
original transaction timezone. Month/week buckets use PostgreSQL calendar-date operations.

### Deleted and held transactions

Deleted transactions are always excluded. Held transactions are excluded by default because a
pending amount is not authoritative; `includeHold: true` opts in explicitly.

### Income, outcome, transfers, and debt

For a normal same-account transaction, positive income and outcome are separate legs. A mixed
same-account transaction can therefore contribute to both sides of cashflow. Different-account
flows are transfers between accounts and do not contribute to spending or cashflow aggregates;
they remain visible and labelled as transfers in transaction search. Debt-account movements remain
distinguishable in search and are not treated as ordinary aggregate income or expense.

### Currency conversion

The model never chooses a report currency. It is derived from `user.currency` for the selected
ZenMoney users, which must agree on one currency within a report. The official ZenMoney API defines
`User.currency` as the primary currency used for balances and reports, and `Instrument.rate` as the
value of one instrument unit in RUB. Therefore a source amount is converted as:

```text
amount_in_user_currency = amount * source_rate_in_RUB / user_currency_rate_in_RUB
```

This also prevents an LLM from silently changing the currency. A report selecting multiple users
is accepted only when every selected user has the same primary currency. When the catalog contains
different currencies, Codex must issue separate calls for each user or same-currency group rather
than asking the server to invent a cross-user reporting currency. A missing, zero, or negative
required rate fails the report rather than dropping a leg. The implementation uses the currently
synchronized rate snapshot; it does not claim historical exchange-rate accuracy. The ZenMoney
`opIncome`/`opOutcome` fields describe the immediate payment currency and are deliberately not used
for report valuation.

Primary ZenMoney API reference: [ZenMoney API entities](https://github.com/zenmoney/ZenPlugins/wiki/ZenMoney-API#instrument), including the definitions of
[`User.currency`](https://github.com/zenmoney/ZenPlugins/wiki/ZenMoney-API#user),
[`Account.inBalance`](https://github.com/zenmoney/ZenPlugins/wiki/ZenMoney-API#account), and
[transaction legs](https://github.com/zenmoney/ZenPlugins/wiki/ZenMoney-API#transaction).

### Categories and multiple tags

A transaction can contain several tags. Distinct tags are first canonicalized to their valid root
category; the transaction amount is then divided equally among the distinct roots. This preserves
the report total and prevents double counting. A child rolls up to its parent; a category without a
valid parent is its own root. Empty tags use the stable `category:uncategorized` bucket. The
official API permits one parent level.

### Budgets and merchants

Budget progress accepts only complete calendar months: inclusive `from` is the first day and
inclusive `to` is the last day of a month. Its internal half-open range therefore starts and ends
on month boundaries, and only monthly expense (`outcome`) budget rows inside that exact range are
read. Child categories roll up identically to spending and are compared with allocated spending.
The official ZenMoney zero tag UUID is the aggregate all-categories budget and is exposed as the
stable `category:all` row; a SQL `NULL` tag is the distinct uncategorized budget. When an aggregate
budget exists, report totals use that aggregate row instead of summing it together with category
budgets, which would double-count the same plan. Current code does not model forecast-generated or
historical budget revisions beyond the rows stored by synchronization.

Missing merchants use stable `merchant:none` / `No merchant` values in search. A blank title is
reported as `Unnamed merchant`; a blank category or account is handled similarly. Merchant IDs are
filters and result identifiers, never an authorization boundary.

### Decimal values

PostgreSQL `NUMERIC` values remain exact through aggregation and are serialized as non-exponential
decimal strings. No financial rounding occurs in the service. Locale, currency symbol, and visible
decimal-place rounding belong to the client/UI formatting layer.

## Authentication and isolation

`IdentityResolver` is the HTTP authentication boundary. It resolves credentials into a subject and
either the full database catalog or a configured numeric allowlist for each request; neither values
from a previous MCP request nor tool arguments can establish identity.

- Local mode uses an explicitly configured `StaticIdentityResolver`. It is suitable only for a
  loopback/private development endpoint. With no allowlist it grants the local principal access to
  every user in the connected database.
- Remote mode requires an exact `Authorization: Bearer <token>` header and rejects configured
  secrets shorter than 32 bytes. The resolver retains only a SHA-256 digest of the configured
  header value and uses constant-time comparison. TLS is still required, normally at a reverse
  proxy or load balancer.
- The boundary is intentionally replaceable by verified OAuth/JWT/mTLS identity later without
  changing analytics or SQL.

Bearer mode is a single-principal bootstrap, not per-user OAuth. Do not share one bearer token
between mutually untrusted users. Because an omitted `ZENMCP_USER_IDS` intentionally exposes the
full database catalog, production deployments should use a database containing only the intended
tenant or configure the restrictive allowlist.

## Stateless transport and rendering

The server uses the official Go SDK's stateless Streamable HTTP handler for MCP `2026-07-28`.
Protocol identity/capabilities arrive on every request and no `Mcp-Session-Id`, hidden report cache,
or instance affinity is required. Any request can reach any healthy replica. Each PostgreSQL report
method reads its currency, totals, bounded rows, and freshness marker inside one read-only
`REPEATABLE READ` transaction, so those pieces describe one database snapshot even if a
synchronization commits concurrently.

`render_finance_chart` uses deterministic re-execution rather than an in-memory report handle: its
input contains the normalized report intent, the server validates it again under the current
principal, reads current authoritative data, and returns the result. For a previous-period
comparison it derives the immediately preceding equal-length calendar-day range and performs a
second authoritative cashflow read; it never fabricates the previous series from current values.
This scales across instances and avoids transmitting large unsigned datasets through the model.
Each report read is internally snapshot-consistent, but two calls (including the current and
previous reads of one comparison) may observe different completed synchronizations. Metadata
exposes freshness so clients can explain that behavior.

## MCP Apps and fallback

Only `render_finance_chart` declares nested `_meta.ui.resourceUri`, pointing to the embedded
`ui://zenmoney/finance-chart` resource with MIME type `text/html;profile=mcp-app`. The self-contained
resource uses the portable MCP Apps `2026-01-26` JSON-RPC/postMessage lifecycle and no host-specific
global object, external CDN, cookie, or parent DOM access. Empty resource CSP origin lists prevent
direct network access.

Apps-capable hosts render responsive SVG charts with keyboard-focusable marks, tooltip, legend,
light/dark theming, and a table switch. Other clients still receive concise text content and the
structured table fallback. Data tools never open UI automatically.

## Operational limits and known limitations

Defaults are deliberately bounded: a 3,660-day maximum period, 50 default / 100 maximum row/page
size, 400 chart points, 100 filter values, 1 MiB request body, 30-second request/query timeout,
24-hour stale threshold, and UTC default timezone. The PostgreSQL adapter has an additional hard
ceiling of 500 rows or chart points. Deployments can lower these values through `zenmcp`
configuration.

Known limitations:

- conversions use current synchronized instrument rates, not rates on transaction dates;
- the source schema has calendar dates but no original transaction timezone;
- freshness is database-wide because the schema lacks per-user sync provenance, and measures the
  local sync snapshot rather than remote ZenMoney availability or rate age;
- budget reporting distinguishes the aggregate zero-UUID budget from uncategorized, but otherwise
  follows stored monthly expense rows and does not recreate every ZenMoney forecast behavior;
- multi-tag allocation is equal by canonical root because ZenMoney stores no per-tag weights;
- a single multi-user report requires one shared primary currency; clients must split different
  currencies into separate stable-key selections;
- the default principal can read every database user unless `ZENMCP_USER_IDS` restricts it, and
  neither bootstrap mode is a substitute for per-user OAuth authorization;
- bearer authentication is a bootstrap single-principal mode; production multi-tenant deployments
  should implement a stronger `IdentityResolver`.
