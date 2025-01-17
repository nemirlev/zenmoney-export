# ZenMoney Export to DB

[![GoDoc](https://godoc.org/github.com/zenexport/zenexport?status.svg)](https://godoc.org/github.com/nemirlev/zenexport)
[![Go Report Card](https://goreportcard.com/badge/github.com/nemirlev/zenexport)](https://goreportcard.com/report/github.com/nemirlev/zenexport)
![GitHub License](https://img.shields.io/github/license/nemirlev/zenexport)
![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/nemirlev/zenexport)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/nemirlev/zenexport)
![Docker Pulls](https://img.shields.io/docker/pulls/nemirlev/zenexport)
![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/nemirlev/zenexport)
![Docker Image Version (latest by date)](https://img.shields.io/docker/v/nemirlev/zenexport)
[![codecov](https://codecov.io/gh/nemirlev/zenexport/graph/badge.svg?token=WOGJKM2YV0)](https://codecov.io/gh/nemirlev/zenexport)

Данный проект создан для экспорта данных в свою БД из сервиса для учета личных
финансов [ZenMoney](https://zenmoney.ru/).

На данный момент поддерживается только ClickHouse, но в будущем планируется добавить поддержку PostgreSQL, MySQL.
Возможно еще каких-то баз, если будет необходимо.

* Если вы хотите строить свою аналитику, но не хотите заморачиваться с инструментами и кодингом, то можете
  использовать [Готовый проект](https://github.com/nemirlev/zenmoney-dashboard).
* Если вы хотите написать свое - можете воспользоваться [Go SDK](https://github.com/nemirlev/zenapi) для доступа к API
  ZenMoney.

## Быстрый запуск

* Получить токен через [Zerro.app](https://zerro.app/token).
* Установите Docker если у вас его нет.

Скопируйте файл .env.example в .env `cp .env.example .env` и заполните там токен, который вы получили выше в Zerro.app.
В docker-compose.yml меняете значения подключения к БД в секции `command` на свои.

> Как в .env так и в command в качестве адреса сервера БД, если он запущен локально - указывается localhost, так
> как используется параметр network_mode: host. Если же БД запущена на другом сервере, то указываете его адрес.

```bash
docker-compose up -d
```

По умолчанию обновление будет происходить каждые 30 минут. Если хотите изменить - добавьте строчку `comand: -interval 180` в
docker-compose.yml, где 180 - это интервал в минутах. После этого перезапустите контейнеры `docker-compose restart`.

> БД ClickHouse должна где-то быть запущена. Если нет, можете использовать полностью готовый проект с Grafana для построения
> диаграмм -  [ZenMoney Dashboard](https://github.com/nemirlev/zenmoney-dashboard)

## Project structure

```
// Project structure
zenmoney-export/
├── cmd/
│   ├── migrate.go             // CLI command for migrations
│   └── root.go                // Root command for Cobra
├── config/
│   └── config.go              // Configuration management
├── docker/
│   ├── clickhouse-compose.yml // Docker Compose for Clickhouse
│   ├── redis-compose.yml      // Docker Compose for Redis
│   ├── mysql-compose.yml      // Docker Compose for MySQL
│   ├── postgres-compose.yml   // Docker Compose for PostgreSQL
│   └── mongo-compose.yml      // Docker Compose for MongoDB
├── internal/
│   ├── export/
│   │   └── exporter.go        // Logic for exporting data
│   ├── sync/
│   │   └── syncer.go          // Logic for syncing data
│   ├── db/
│   │   ├── clickhouse.go      // Clickhouse integration
│   │   ├── redis.go           // Redis integration
│   │   ├── in_memory.go       // In-memory DB implementation
│   │   ├── mysql.go           // MySQL driver integration
│   │   ├── postgres.go        // PostgreSQL driver integration
│   │   └── mongo.go           // MongoDB driver integration
│   ├── utils/
│   │   ├── logger.go          // Logging utilities with structured logging (slog)
│   │   └── interfaces.go      // Interfaces for mocks and testing
│   └── mocks/
│       └── db_mock.go         // Mock implementations for testing
├── migrations/                // SQL migrations
│   └── postgres/
│       └── 000001_init.up.sql // Initial migration for PostgreSQL
│   └── clickhouse/
│       └── 000001_init.up.sql // Initial migration for PostgreSQL
│   └── mysql/
│       └── 000001_init.up.sql // Initial migration for PostgreSQL
│   └── mongo/
│       └── 000001_init.up.sql // Initial migration for PostgreSQL
├── mockery.yaml               // Configuration for mockery
├── pkg/
│   └── zenapi/
│       └── client.go          // Wrapper for ZenMoney API SDK
├── go.mod                     // Module file
├── go.sum                     // Dependency management
├── README.md                  // Documentation
└── main.go                    // Entry point
```

## Использование

Необходимо:

* Получить токен через [Zerro.app](https://zerro.app/token).
* Установить утилиту для миграций [migrate](https://github.com/golang-migrate/migrate)

После установки, запускаем миграции, значение переменных не забудьте поменять на свои:

```bash
migrate -path ./migration -database 'clickhouse://$SERVER_ADDRES:9000?database=$DATABASE_NAME&username=$USER&password=$PASSWORD' up
```

Запускаем экспорт в обычном режиме (программа завершается после выполнения). Не забудьте поменять значения переменных на
свои:

```bash
go run main.go -token $TOKEN -server $SERVER -user $USER -db $DB_NAME -password $PASSWORD 
```

Либо может запустить в режиме демона, который будет запускать экспорт каждые столько минут, сколько вы указали в
параметре -interval. Не забудьте поменять значения переменных на свои:

```bash
go run main.go -d -interval 60 -token $TOKEN -server $SERVER -user $USER -db $DB_NAME -password $PASSWORD -interval 360
```

## Параметры и переменные окружения

Парметры:

| Переменная | Описание                                              | Значение по умолчанию |
|------------|-------------------------------------------------------|-----------------------|
| token      | Токен для доступа к API ZenMoney                      | ""                    |
| server     | Адрес сервера БД                                      | ""                    |
| dbtype     | Тип БД                                                | clickhouse            |
| user       | Пользователь БД                                       | ""                    |
| db         | Имя БД                                                | ""                    |
| password   | Пароль пользователя БД                                | ""                    |
| interval   | Интервал запуска экспорта в режиме демона (в минутах) | 5                     |
| d          | Запуск в режиме демона                                | false                 |

Переменные окружения:

| Переменная          | Описание                                                      | Значение по умолчанию |
|---------------------|---------------------------------------------------------------|-----------------------|
| ZENMONEY_TOKEN      | Токен для доступа к API ZenMoney                              | ""                    |
| CLICKHOUSE_SERVER   | Адрес сервера БД                                              | ""                    |
| CLICKHOUSE_USER     | Пользователь БД                                               | ""                    |
| CLICKHOUSE_DB       | Имя БД                                                        | ""                    |
| CLICKHOUSE_PASSWORD | Пароль пользователя БД                                        | ""                    |

## Вклад в проект

Мы приветствуем вклад от сообщества! Если вы хотите внести изменения в код, пожалуйста, следуйте этим шагам:

1. Форкните репозиторий.
2. Создайте новую ветку для ваших изменений.
3. Сделайте изменения в вашей ветке.
4. Отправьте Pull Request с описанием ваших изменений.

Пожалуйста, убедитесь, что ваш код соответствует стандартам Go и что все тесты проходят перед отправкой PR.

> Если вы хотите помочь, но не знаете с чего начать, то посмотрите Issues и создайте свой, если не нашли подходящего.

TODO:

- [ ] Добавить тесты
- [ ] Добавить частичное обновление на основе ServerTimestamp. Для этого можно использовать BadgerDB, что бы не было
  внешних
  зависимостей.
- [ ] Сделать реализацию сохранения в БД через интерфейс (заккоментировал набросок в виде интерфейса в bd.go и метода
  saveBatch в clickhouse.go)
- [ ] Добавить поддержку PostgreSQL
- [ ] CI/CD - сборка, линтинг, тесты, пуш в DockerHub
- [ ] Сделать через ENTRYPOINT в Dockerfile, что бы можно было использовать дополнительные аргументы в command
- [ ] Добавить прогресс бар
- [ ] Изменить логирование и вывод результатов в консоль через Zaper