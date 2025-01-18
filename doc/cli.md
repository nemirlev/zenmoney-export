# CLI Commands Structure

## Root Command (zenexport)

Global flags available for all commands:
```
--config        Path to config file (default: $HOME/.zenexport.yaml)
--token         ZenMoney API token
--log-level    Log level (debug, info, warn, error)
--format       Output format (text, json) for applicable commands
```

## Command: sync
Synchronizes data from ZenMoney to the database.

```
zenexport sync [flags]
```

Flags:
```
--db-type          Database type (clickhouse, postgres, mysql)
--db-url           Database connection URL
--daemon, -d       Run in daemon mode
--interval         Sync interval in minutes (for daemon mode)
--from             Start date for sync (format: YYYY-MM-DD)
--to               End date for sync (format: YYYY-MM-DD)
--entities         Comma-separated list of entities to sync (transactions,accounts,tags,merchants)
--batch-size       Number of records to process in one batch
--force            Force full sync ignoring last sync timestamp
--dry-run          Show what would be synced without actually syncing
```

## Command: migrate
Manages database migrations.

```
zenexport migrate [command]
```

Subcommands:
```
up          Apply all or N up migrations
down        Apply all or N down migrations
create      Create a new migration file
status      Show migration status
version     Show current migration version
```

Flags:
```
--db-type          Database type (clickhouse, postgres, mysql)
--db-url           Database connection URL
--path             Path to migration files
--version          Target version for migrate up/down
```

## Command: check
Performs various checks and validations.

```
zenexport check [flags]
```

Flags:
```
--db-connection    Check database connection
--api-token        Validate API token
--migrations       Check if migrations are up to date
```

## Command: info
Shows various information about the system and sync status.

```
zenexport info [command]
```

Subcommands:
```
status      Show current sync status and statistics
config      Show current configuration
db          Show database information and statistics
```

## Command: export
Exports data from the database to various formats.

```
zenexport export [flags]
```

Flags:
```
--format           Export format (csv, json, excel)
--output, -o       Output file path
--entities         Comma-separated list of entities to export
--from             Start date for export (format: YYYY-MM-DD)
--to               End date for export (format: YYYY-MM-DD)
--compress         Compress output file
```

## Usage Examples

1. Basic sync with default settings:
```bash
zenexport sync --token="your-token" --db-url="clickhouse://..."
```

2. Run sync in daemon mode:
```bash
zenexport sync -d --interval=30 --token="your-token" --db-url="clickhouse://..."
```

3. Apply migrations:
```bash
zenexport migrate up --db-url="clickhouse://..."
```

4. Export data:
```bash
zenexport export --format=csv --output=./export.csv --entities=transactions
```

5. Check system status:
```bash
zenexport check --db-connection --api-token
```