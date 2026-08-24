## [2.0.3] - 2026-03-07

### 🚀 Features

- Update ZenMoney Go SDK to v3.0.1
- Persist tag archive status and merchant MCC values

### 🐛 Bug Fixes

- Upgrades Go version in Docker build image (#22)
## [2.1.4](https://github.com/nemirlev/zenmoney-export/compare/v2.1.3...v2.1.4) (2026-08-24)


### ⚙️ Miscellaneous Tasks

* **deps:** update testcontainers and transitive dependencies ([e83be20](https://github.com/nemirlev/zenmoney-export/commit/e83be204b05b8c01508889ed76fcbfbfbbb7b713))

## [2.1.3](https://github.com/nemirlev/zenmoney-export/compare/v2.1.2...v2.1.3) (2026-08-24)


### ⚙️ Miscellaneous Tasks

* **deps:** bump github.com/moby/go-archive from 0.2.0 to 0.3.0 ([#29](https://github.com/nemirlev/zenmoney-export/issues/29)) ([8c83321](https://github.com/nemirlev/zenmoney-export/commit/8c83321bd6e25520e947cf03c9afe734f33cc8b3))
* **deps:** bump golang.org/x/crypto from 0.48.0 to 0.52.0 ([#28](https://github.com/nemirlev/zenmoney-export/issues/28)) ([fb7d708](https://github.com/nemirlev/zenmoney-export/commit/fb7d708ab7471bcc8e49cc247e8d808704c53be6))

## [2.1.2](https://github.com/nemirlev/zenmoney-export/compare/v2.1.1...v2.1.2) (2026-08-24)


### 🧪 Testing

* **postgres:** run compatibility checks with testcontainers ([cfe92b9](https://github.com/nemirlev/zenmoney-export/commit/cfe92b9c4df9f4cbe491f0249252191d32922379))
* regenerate interface mocks with mockery v3 ([fd6afb8](https://github.com/nemirlev/zenmoney-export/commit/fd6afb82db5577ee9c7d55c0caf454cb41aa59a2))

## [2.1.1](https://github.com/nemirlev/zenmoney-export/compare/v2.1.0...v2.1.1) (2026-08-24)


### 🐛 Bug Fixes

* **ci:** remove codeql from configs fot use github deafult ([f3ad8dd](https://github.com/nemirlev/zenmoney-export/commit/f3ad8dd21fffc2c87fd8ec8972a1f0501049c1dd))
* **sync:** persist actual synchronization type ([322d45d](https://github.com/nemirlev/zenmoney-export/commit/322d45d912eea51dcaeee192e096aa6e4dd48389))


### 🚜 Refactor

* **postgres:** consolidate persistence helpers ([5261178](https://github.com/nemirlev/zenmoney-export/commit/5261178236a5ae064e2defd23a53839d1a9a27f9))
* **postgres:** rename sync lock connector ([f78439f](https://github.com/nemirlev/zenmoney-export/commit/f78439f83371d1f42a5bffb6673f0b8da6b7e9c9))
* **postgres:** simplify response persistence ([35bdce4](https://github.com/nemirlev/zenmoney-export/commit/35bdce42b759ed7f19f36f998be38c69f02ff35c))
* rename sync lock interface ([119fe9b](https://github.com/nemirlev/zenmoney-export/commit/119fe9b369cefc02a8940d3a5695e2cc0b210f00))


### 🧪 Testing

* **cmd:** simplify sync cleanup scenarios ([da0e5f1](https://github.com/nemirlev/zenmoney-export/commit/da0e5f139d39f461e5f7b1f9540101e52c76fe00))
* **postgres:** consolidate shared fixtures ([ae49110](https://github.com/nemirlev/zenmoney-export/commit/ae49110fef5313f0f4282beab6d37b24b1c6c5cf))


### 📦 Build System

* **docker:** upgrade compose to PostgreSQL 18 ([c5c15c7](https://github.com/nemirlev/zenmoney-export/commit/c5c15c70fa8ae83f9843081e1c42644d486843f0))


### 👷 Continuous Integration

* pin workflow actions to commit SHAs ([2167181](https://github.com/nemirlev/zenmoney-export/commit/216718130b2e8614998cde1776f0b63af7b8ab7f))

## [2.1.0](https://github.com/nemirlev/zenmoney-export/compare/v2.0.3...v2.1.0) (2026-08-24)


### 🚀 Features

* migrate to zenmoney sdk v3 ([e041e01](https://github.com/nemirlev/zenmoney-export/commit/e041e0161564c998922b3e0649cd8f43f0db6543))
* **postgres:** add transaction copy write mode ([6623ae3](https://github.com/nemirlev/zenmoney-export/commit/6623ae35350cc61f602c2112f0b32ffa9ef66264))
* **postgres:** chunk sync batches ([2433852](https://github.com/nemirlev/zenmoney-export/commit/2433852ffc1213901e25484ba10197e442817c3d))
* **postgres:** use analytical storage types ([f5688c9](https://github.com/nemirlev/zenmoney-export/commit/f5688c911e1b6e1653ea3c52bcf8f7fa090c1cf9))
* **sync:** serialize postgres synchronization ([521edb9](https://github.com/nemirlev/zenmoney-export/commit/521edb9c81ac058dcf9fffe956b122b18aa5cf67))


### 🐛 Bug Fixes

* **config:** align CLI and configuration contract ([5e252ad](https://github.com/nemirlev/zenmoney-export/commit/5e252ad9a715180523bdb6fa124b3707422e8f27))
* **docker:** build exporter with matching migrations ([edf6ab2](https://github.com/nemirlev/zenmoney-export/commit/edf6ab23d6b3d8db2f25b16fe930ec93a6f43a1a))
* **docker:** exclude secrets from build context ([fc24642](https://github.com/nemirlev/zenmoney-export/commit/fc24642b96e999bcee0b297b14368a995f6476cb))
* **logging:** install configured slog logger ([9a0d7cd](https://github.com/nemirlev/zenmoney-export/commit/9a0d7cd48eb4ede44bddfa09a490dab6e4dfb9d7))
* **migrations:** drop enum types on rollback ([e5409d4](https://github.com/nemirlev/zenmoney-export/commit/e5409d43333653800f7ef05ed4419ac41b8c9b8b))
* **postgres:** format dates independently of datestyle ([36b32f6](https://github.com/nemirlev/zenmoney-export/commit/36b32f655e2c0a914be7319fe3853825e36dd108))
* **postgres:** keep public batch saves atomic ([40b2e47](https://github.com/nemirlev/zenmoney-export/commit/40b2e47082d7b330f3a6c6fbb0008e15f15eb40d))
* **postgres:** normalize UUIDs before copy dedup ([d03bba1](https://github.com/nemirlev/zenmoney-export/commit/d03bba19462c12f42de98b3dfe36b3458ea9fe8b))
* **postgres:** reject budget deletion objects ([f9a4293](https://github.com/nemirlev/zenmoney-export/commit/f9a42932973d94ad100cd8368f32964acef1bbb4))
* **postgres:** save sync responses atomically ([b14e4eb](https://github.com/nemirlev/zenmoney-export/commit/b14e4eb4c21f0578a2e0a05eea4b46eb66340ab7))
* **postgres:** upsert budgets by natural key ([ff5e562](https://github.com/nemirlev/zenmoney-export/commit/ff5e562dbea1722a473e4b2489c8397d1f8416a8))
* **sync:** handle daemon lifecycle gracefully ([78e2bd8](https://github.com/nemirlev/zenmoney-export/commit/78e2bd889741c8e0098fba4097a2dba21b3c45e6))
* **sync:** honor entity selection ([c61dc7c](https://github.com/nemirlev/zenmoney-export/commit/c61dc7c3e0959b36e9299808a209b84cc9259059))
* **sync:** isolate advisory lock connection ([254cf5f](https://github.com/nemirlev/zenmoney-export/commit/254cf5fdd1b0e7ef8df1979fcbd1320868837755))
* **sync:** make API response limit configurable ([9b032ac](https://github.com/nemirlev/zenmoney-export/commit/9b032acf0dfbb4099886b0702d3b702862aff252))
* **sync:** report dry-run changes ([8e61a22](https://github.com/nemirlev/zenmoney-export/commit/8e61a223bd488901d0834bde0d01b638f0be025d))
* **sync:** resume only from completed runs ([7421e8f](https://github.com/nemirlev/zenmoney-export/commit/7421e8f5fe3243c06ea9aea2d38765d7079bdc84))


### 🚜 Refactor

* adopt modern Go helpers ([4a37ebe](https://github.com/nemirlev/zenmoney-export/commit/4a37ebede6417d79e3cef3d2bebb3bb59f5cf724))


### 📚 Documentation

* document sdk v3 migration ([5896534](https://github.com/nemirlev/zenmoney-export/commit/5896534b7456a0ec86cc0d16f9099db90e6d9852))
* replace go report card badge ([3ac2597](https://github.com/nemirlev/zenmoney-export/commit/3ac2597513b91d5604e7526a2c0a273ab8a88659))
* require PostgreSQL 16 or newer ([42800e0](https://github.com/nemirlev/zenmoney-export/commit/42800e0b89987b8c3266cd1ced3c293ca4dbeb09))


### 📦 Build System

* **docker:** harden local postgres stack ([66e9f07](https://github.com/nemirlev/zenmoney-export/commit/66e9f077ff65c96ee0504adeb89f0d56feb37a45))
* upgrade to Go 1.27 ([3ca560d](https://github.com/nemirlev/zenmoney-export/commit/3ca560dded5d3a2600c4c8df23864a3b018d9d21))


### 👷 Continuous Integration

* configure manual CodeQL build ([c76aea1](https://github.com/nemirlev/zenmoney-export/commit/c76aea19f3f070d6cbff8cc533fa955271506d51))
* improve release changelog formatting ([1dc95bd](https://github.com/nemirlev/zenmoney-export/commit/1dc95bda52e9bada1e5dc6833368074e6986d354))
* migrate releases to release please ([f438a4d](https://github.com/nemirlev/zenmoney-export/commit/f438a4d2372ae4a54dc8de0d8bd1f04c7340b846))
* pin release actions to commit SHAs ([a6d7d89](https://github.com/nemirlev/zenmoney-export/commit/a6d7d899fc28577e79d67e4c93ec147b10254162))
* restrict workflow token permissions ([4c63c49](https://github.com/nemirlev/zenmoney-export/commit/4c63c4908a6f16ce5430b428bd076c2cf1c67d25))
* support golangci-lint 2.13 ([48376f6](https://github.com/nemirlev/zenmoney-export/commit/48376f692c8b90731d72f31d0795a139a4f175ab))
* update GitHub Actions ([6b1a603](https://github.com/nemirlev/zenmoney-export/commit/6b1a60373554daf7489a574a5b76c90d208c4678))
* upgrade release please action to v5 ([4524127](https://github.com/nemirlev/zenmoney-export/commit/452412773ebd4cf0c519cb78c8da1481a7308a1e))


### ⚙️ Miscellaneous Tasks

* **deps:** bump github.com/jackc/pgx/v5 from 5.8.0 to 5.9.2 ([#24](https://github.com/nemirlev/zenmoney-export/issues/24)) ([1f81572](https://github.com/nemirlev/zenmoney-export/commit/1f815725e33f18531e275e9cd2703f41868dd700))
* **deps:** update Go modules ([6b7391a](https://github.com/nemirlev/zenmoney-export/commit/6b7391a0cc5eef054b70fb6bd0d3216cc8c15061))

## [2.0.2] - 2026-03-07

### 🐛 Bug Fixes

- Update Docker image badge link in README
- *(ci)* Exclude mocks from test coverage

### 📚 Documentation

- Update API token source in Quick Start section (#19)
## [2.0.1] - 2025-02-27

### 🚀 Features

- Update config and docker setup
## [2.0.0] - 2025-02-27

### 🚀 Features

- Добавлены тесты для логгера
- Добавлены моки
- Добавлены тесты для функций create, uodate и delete в БД
- Update ClickHouse connection handling
- Add new structure and base CLI
- Refactor command structure and add config support
- Add commands for check, config, export, sync
- Add PostgreSQL setup and initial migrations
- Add usage info to root command
- *(database)* Enhance schema with new fields
- *(migrations)* Add 'force' to sync_type enum
- Add storage interface and factory function
- Add PostgreSQL storage implementation
- Update init scripts for new tables
- Use postgres package for storage creation
- Add PostgreSQL storage and CRUD operations
- Update .gitignore to include build directory
- *(config)* Update DBConfig comment for clarity
- *(sync)* Enhance sync process and database handling
- Remove unused command files
- Update default sync entities option
- Update to version 2 structure
- Remove date flags from sync options
- Remove date options from sync command
- Add mocks
- Add interface for pgxpool.Pool

### 🐛 Bug Fixes

- Improve sync condition checks
- Update container names for clarity
- Move GetReminder function to a new file
- Update Docker platforms for release workflow

### 💼 Other

- Добавлена зависимость github.com/stretchr/objx v0.5.2

### 🚜 Refactor

- Delete all structure
- Rename config file constant
- Move factory comment up to function
- Rename package and update struct names
- Separate postgresql methods
- Sync status tracking in Save method
- Command structure and sync logic
- Rename Postgres compose file

### 🧪 Testing

- Add unit tests for storage creation
- Add account management tests
- Add batch tests for database save functions
- Add budget management tests
- Add comprehensive database tests

### ⚙️ Miscellaneous Tasks

- Update dependencies in go.mod and go.sum
- Update dependencies to latest versions
- Update README for ZenMoney Export
- Update Go version and dependencies
- Add changelog and code of conduct
- Full update ci/cd for working proposal
- Update release workflow and changelog generation
- Clean up extra files in release workflow
- Update release workflow cleanup step
## [1.4.2] - 2024-05-13

### 🐛 Bug Fixes

- После рефакторинга не проходило сохранение, исправил ошибку

### 💼 Other

- Добавил описание методов в пакете БД и clickhouse

### 🚜 Refactor

- Упростил код методов `saveX` и убрал дублирование
## [1.4.1] - 2024-05-13

### 🚀 Features

- Добавил логирование Debug
- *(log)* Добавил логирование в метод `executeBatch` clickhouse
- Добавил логирование для функции `runSyncAndSave`

### 🚜 Refactor

- Изменил `log.fatal` в сохранение на ошибки и добавил свой лог
- Убрал передачу переменных в методы БД - перенес в структуры
- Логи статуса импорта в БД поменял на сообщения в консоль
- В `Clickhouse.connect` заменил вывод в консоль на лог
- Перенес БД драйвер из параметров функции в `func receivers`
## [1.4.0] - 2024-05-12

### 🚀 Features

- Добавил подробный вывод в консоли статуса экспорта в БД

### 💼 Other

- Обновил версии пакетов

### 🚜 Refactor

- Переместил реализацию БД в `internal`, переписал интерфейс
- Вынес логику пакетной вставки в отдельную функцию
- Сделал разделение статуса в выводе консоли на zenmoney и БД

### ⚙️ Miscellaneous Tasks

- Обновил условия запуска тестов
## [1.3.0] - 2024-05-11

### 💼 Other

- Миграции для postgresql и docker-compose для тестирования (#9)
- Добавил пакет для удобного логирования (#12)

### 🚜 Refactor

- Переместил категорию с миграциями clickhouse в отдельную директорию (#10)
- Перенес конфигурирование переменных в отдельный пакет (#11)

### ⚙️ Miscellaneous Tasks

- Добавил возможность генерировать changelog
- Изменил тригер для создания артефактов релиза
## [1.2.1] - 2024-04-29

### 🚀 Features

- Add buildx for multistage build

### 🐛 Bug Fixes

- Docker build platform
- Build multiimage
- Return build without matrix
- Remove darwin
## [1.2.0] - 2024-04-29

### 🚀 Features

- Create ci for push feature (lint, test, build) (#4)

### 🐛 Bug Fixes

- Brache name on .github actions
- Get version name on github actions
- Удалил linux/riscv64 так как не поддерживается alpine
- Исправил название образа
- Исправил токен на персональный
- Repo name
- Add get version to workflow

### 💼 Other

- Github actions for test, lint code and build multiplatform images
- Remove pre-build release
## [1.1.0] - 2023-12-16

### 🚀 Features

- Добавлены Docker-метрики и ссылки на Docker Hub

### 🐛 Bug Fixes

- Изменить тип столбца color в таблице tag
- Обновлены настройки подключения к ClickHouse
- Обновлен go.sum
## [1.0.0] - 2023-12-09
