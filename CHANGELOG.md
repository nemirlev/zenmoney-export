# Changelog

All notable changes to this project will be documented in this file.

## [unreleased]

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
- Add issue templates and CI workflows
- Add app service and Dockerfile for build
- Add MIT License to the project
- Add manual trigger for release workflow
- Specify Dockerfile in release workflow
- Add GoReleaser configuration for Docker builds
- Update CI/CD workflows and add Docker support
- Add environment variables for database connection
- Add changelog generation to release workflow

### 🐛 Bug Fixes

- Improve sync condition checks
- Update container names for clarity
- Move GetReminder function to a new file
- Update release workflow for Go and changelog
- Update GitHub token for release workflow
- Tidy up CI workflow formatting
- Update release args in workflow
- Update changelog commit config

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

### 📚 Documentation

- Tidy up badge formatting in README

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
- Update Go version in CI workflows
- Update changelog for version 2.0.0
- Full refactor ci and cd
- Обновление CHANGELOG.md [skip ci]
- Обновление CHANGELOG.md [skip ci]
- Обновление CHANGELOG.md [skip ci]
- Обновление CHANGELOG.md [skip ci]
- Обновление CHANGELOG.md
- Automate changelog commit on release
- Обновление CHANGELOG.md
- Обновление CHANGELOG.md for release
- Update CI workflows for changelog handling
- Update CHANGELOG.md
- Update Go versions and remove lint dependency
- Update release notes generation
- Add changelog generation to release workflow
- Update CHANGELOG.md
- Update release workflow for changelog handling
- Update CHANGELOG.md
- Update CHANGELOG.md
- Update CHANGELOG.md
- Update changelog action to v4
- Update lint and coverage actions
- *(ci)* Update changelog action version
- Remove goreleaser configuration file

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

<!-- generated by git-cliff -->
