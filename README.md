# ZenMoney Export

[![GoDoc](https://godoc.org/github.com/zenexport/zenmoney-export?status.svg)](https://godoc.org/github.com/nemirlev/zenmoney-export) [![Go Report Card](https://goreportcard.com/badge/github.com/nemirlev/zenmoney-export)](https://goreportcard.com/report/github.com/nemirlev/zenmoney-export) ![GitHub License](https://img.shields.io/github/license/nemirlev/zenmoney-export) ![Go Version](https://img.shields.io/github/go-mod/go-version/nemirlev/zenmoney-export) ![Latest Release](https://img.shields.io/github/v/release/nemirlev/zenmoney-export) ![Docker Pulls](https://img.shields.io/docker/pulls/nemirlev/zenexport) ![Docker Image Size](https://img.shields.io/docker/image-size/nemirlev/zenexport) [![codecov](https://codecov.io/gh/nemirlev/zenmoney-export/graph/badge.svg?token=WOGJKM2YV0)](https://codecov.io/gh/nemirlev/zenmoney-export)

ZenMoney Export is a tool designed to export and sync data from the personal finance management
service [ZenMoney](https://zenmoney.ru/) to your own database.

## Features

- 🚀 Fast and reliable synchronization. Support Full and Incremental sync modes.
- 📊 Supports PostgreSQL (with plans for other databases).
- 🛠️ Easy-to-configure options for various use cases.
- 🐳 Docker-ready for seamless deployment.

## Quick Start

Obtain an API token from ZenMoney by visiting [Zerro.app](https://zerro.app/token) and following the instructions to
generate your token.

1. Choose a supported database type from `./docker` directory.
2. Change environment in the docker compose file.
3. Start the database and the exporter. Example for PostgreSQL:

```bash
docker compose -f ./docker/docker-compose.postgres.yml up -d
```

## Configuration

### Environment Variables

Global variables:

- `ZEN_API_TOKEN`: Your ZenMoney API token.
- `DB_URL`: Connection string for your database. Example: `postgres://user:password@localhost:5432/dbname`.
- `DB_TYPE`: Database type. Default: `postgres`.
- `LOG_LEVEL`: Log level for the exporter. Default: `info`.
- `FORMAT`: Export format. Default: `json`.

Command-specific variables:

- Refer to the command-specific help by running:

```bash
go run main.go --help
```

### File Configuration

Default configuration file is `~/.zenexport.yaml`. You can specify a custom file using the `--config` flag.

```yaml
db_type: postgres
db_config: "postgres://postgres:postgres@localhost:5432/postgres"
log_level: debug
format: json
token: not-a-real-token
```

### Comannnd-Line Arguments

Parameters can be set using environment variables or directly via command-line arguments.

```bash
go run main.go --token=your-token-here --db-url=postgres://user:password@localhost:5432/dbname
```

## Commands

Now app supports the following commands:

- `sync`: Synchronize data from ZenMoney to your database.

### Sync Command

The `sync` command is used to synchronize data from ZenMoney to your database. If command run first time, it will
perform a full sync (in demon mode full sync plus incremental sync every `interval` minutes). Otherwise, it will perform
an incremental sync. Also you can force a full sync using the `--force`

Flags:

- `-d`, `--daemon`: Run the sync in daemon mode, continuously syncing at intervals.
- `--dry-run`: Perform a trial run with no changes made to the database.
- `--entities string`: Specify which entities to sync (default "all").
- `--force`: Force a full sync, ignoring any previous sync state.
- `-h`, `--help`: Show help information for the sync command.
- `--interval int`: Set the sync interval in minutes (default 30).

Example for full sync entities `transactions` and `accounts` plus getting the latest data:

```bash
go run main.go sync --entities transactions,accounts --force
```

Example for incremental sync with a 5-minute interval:

```bash
go run main.go sync --interval 5 --daemon
```

Example for running full sync with a dry run:

```bash
go run main.go sync --dry-run
```

## Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository.
2. Create a feature branch.
3. Commit your changes.
4. Submit a pull request.

### Running Tests

To ensure the project remains robust, run tests using:

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

Thanks to the ZenMoney team for their API and documentation, and to all contributors who help make this tool better.
