# CLI reference

ZenMoney Export currently provides one command: `sync`.

## Configuration priority

Configuration is resolved in this order, from highest to lowest priority:

1. Command-line flags.
2. Environment variables.
3. The selected YAML configuration file.
4. Built-in defaults.

The default file is `~/.zenexport.yaml`. Select another file with `--config`.

```yaml
token: your-zenmoney-token
db_type: postgres
db_url: postgres://user:password@localhost:5432/zenmoney
log_level: info
```

For backward compatibility, the YAML key `db_config` is accepted in place of `db_url`.

Environment variables:

- `ZEN_API_TOKEN` — required ZenMoney API token; legacy alias: `TOKEN`.
- `DB_URL` — required PostgreSQL connection URL; legacy alias: `DB_CONFIG`.
- `DB_TYPE` — database type; defaults to `postgres` and currently supports only PostgreSQL.
- `LOG_LEVEL` — `debug`, `info`, `warn`, or `error`; defaults to `info`.

The canonical environment name takes precedence when both it and its legacy alias are present.

## Root command

```text
zenexport [global flags] sync [sync flags]
```

Global flags:

```text
--config string      config file path (default ~/.zenexport.yaml)
--token string       ZenMoney API token
--db-type string     database type (postgres) (default "postgres")
--db-url string      database connection URL
--log-level string   log level: debug, info, warn, error (default "info")
```

## `sync`

Synchronizes ZenMoney data to PostgreSQL. The first run performs a full sync; later runs continue from the last successful synchronization. Use `--force` to ignore the saved cursor.

```text
zenexport sync [flags]
```

Sync flags:

```text
-d, --daemon          run continuously
    --interval int    interval in minutes for daemon mode (default 30)
    --entities string entities to sync (default "all")
    --force            force a full sync
    --dry-run          fetch data without writing it to the database
```

Examples:

```bash
# Use flags
zenexport sync \
  --token="your-token" \
  --db-url="postgres://user:password@localhost:5432/zenmoney"

# Use a selected configuration file
zenexport sync --config=/etc/zenexport.yaml

# Run every five minutes
zenexport sync --daemon --interval=5

# Force a full sync for selected entities
zenexport sync --entities=transactions,accounts --force
```
