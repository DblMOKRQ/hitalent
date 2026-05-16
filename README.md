# HiTalent — API организационной структуры

REST API для работы со структурой компании:
- подразделения (`Department`) с древовидной иерархией
- сотрудники (`Employee`), привязанные к подразделениям

Стек:
- Go + `net/http`
- PostgreSQL
- GORM
- goose (миграции)
- Docker + docker-compose

## Возможности

- Создание подразделений (включая вложенные)
- Создание сотрудников в подразделении
- Получение подразделения с поддеревом (`depth`) и сотрудниками (`include_employees`)
- Перемещение подразделения (смена `parent_id`) с защитой от циклов
- Удаление подразделения:
  - `cascade` — каскадное удаление подразделения/поддерева/сотрудников
  - `reassign` — перевод сотрудников в другое подразделение перед удалением

## Структура проекта

- `cmd/main.go` — точка входа
- `internal/config` — загрузка конфигурации из env
- `internal/domain` — модели и доменные ошибки
- `internal/repository/postgres` — доступ к БД + запуск миграций
- `internal/service` — бизнес-логика
- `internal/transport/http` — DTO, хендлеры, роутер, middleware
- `migrations` — SQL-миграции goose
- `e2e` — интеграционные e2e-тесты

## Быстрый старт

### 1) Подготовка env

Создайте файл `.env` (можно на основе `env.example`):

```env
HTTP_ADDR=:8080
LOG_LEVEL=debug

DB_USER=postgres
DB_PASSWORD=123
DB_HOST=localhost
DB_PORT=5432
DB_NAME=hitalent
DB_SSLMODE=disable
DB_MIGRATION_PATH=./migrations
```

> В `docker-compose` для приложения `DB_HOST` автоматически подставляется как `db`.

### 2) Запуск

```bash
docker-compose up --build
```

API будет доступно по адресу: `http://localhost:8080`

### 3) Остановка

```bash
docker-compose down
```

## API

### 1. Создать подразделение
`POST /departments/`

Body:
```json
{
  "name": "Backend",
  "parent_id": 1
}
```

### 2. Создать сотрудника в подразделении
`POST /departments/{id}/employees/`

Body:
```json
{
  "full_name": "Ivan Ivanov",
  "position": "Go Developer",
  "hired_at": "2026-05-01"
}
```

### 3. Получить подразделение (детали + сотрудники + поддерево)
`GET /departments/{id}?depth=2&include_employees=true`

Параметры:
- `depth` — по умолчанию `1`, максимум `5`
- `include_employees` — по умолчанию `true`

### 4. Обновить подразделение (имя/родитель)
`PATCH /departments/{id}`

Body:
```json
{
  "name": "Platform",
  "parent_id": 3
}
```

### 5. Удалить подразделение
`DELETE /departments/{id}?mode=cascade`

или

`DELETE /departments/{id}?mode=reassign&reassign_to_department_id=2`

## Бизнес-ограничения

- Нельзя создать сотрудника в несуществующем подразделении (`404`)
- `department.name`: обязательное, длина `1..200`, с `TrimSpace`
- `employee.full_name`, `employee.position`: обязательные, длина `1..200`, с `TrimSpace`
- В пределах одного `parent_id` названия подразделений должны быть уникальны
- Нельзя назначить подразделение родителем самому себе
- Нельзя создать цикл в дереве (перемещение внутрь собственного поддерева)
- `cascade`-удаление реализовано на уровне FK (`ON DELETE CASCADE`)

## Тестирование

### Unit-тесты сервисов

```bash
go test ./internal/service/...
```

### E2E-тесты

Требуют запущенную PostgreSQL на `localhost:5432` с параметрами из `.env`.

```bash
go test -v ./e2e/...
```

## Полезные команды

```bash
make up      # docker-compose up -d
make down    # docker-compose down
make rebuild # пересборка и запуск
make test    # e2e тесты
```
