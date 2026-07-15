# API Reference

REST API для expense tracker. Все endpoints начинаются с `/api`. Auth — через session cookie (stateful).

> **Formal spec:** [`api/openapi.yaml`](./api/openapi.yaml) — OpenAPI 3.0.3.
> Просмотр в браузере: поднять сервер и открыть `/docs` (Redoc). Локальный
> preview без сервера: `npx @redocly/cli preview-docs docs/api/openapi.yaml`.
>
> Spec сейчас покрывает `auth + transactions`. Остальные endpoints
> добавляются по мере надобности — prose-документация ниже актуальна для всех
> ресурсов.

## Соглашения

- **Базовый путь:** `/api`
- **Формат дат:** ISO 8601 (`2026-07-13T10:30:00Z`).
- **Денежные суммы:** `integer (int64)` в минорных единицах. $12.50 → `1250`. Divisor = 100 для USD/EUR/RUB.
- **ID:** UUID v4 строки.
- **Content-Type:** `application/json` для всех request bodies.

### Коды ответов

| HTTP | Значение |
|------|----------|
| 200 | Успех (GET, PATCH) |
| 201 | Создано (POST) |
| 204 | Нет контента (DELETE, logout) |
| 400 | Невалидный запрос |
| 401 | Не авторизован |
| 403 | Запрещено |
| 404 | Не найдено |
| 409 | Конфликт (дубликат, ссылка используется) |
| 422 | Бизнес-правило нарушено (невалидные ссылки в transaction) |
| 429 | Rate limit превышен |
| 500 | Внутренняя ошибка |

### Формат ошибок

```json
{
  "code": "ACCOUNT_NOT_FOUND",
  "message": "account not found"
}
```

Для validation ошибок добавляется `errors[]`:
```json
{
  "code": "VALIDATION_FAILED",
  "message": "validation failed",
  "errors": [
    {"field": "name", "message": "name is required"}
  ]
}
```

`code` — машиночитаемый (см. [Error codes](#error-codes)).
`message` — человекочитаемое описание.

---

## Auth

Stateful sessions. Server хранит сессии в таблице `sessions`, клиент получает `session_id` через httpOnly cookie.

### Cookie

- **Name:** `session_id` (настраивается в config).
- **Attributes:** `HttpOnly`, `Secure` (config-driven), `SameSite=Lax` (config-driven).
- **TTL:** 24h по умолчанию (config-driven).
- **Sliding expiration:** при запросе, если до истечения < 25% TTL, `expires_at` продлевается.

### Endpoints

| Метод | Endpoint | Auth | Описание |
|------|----------|------|----------|
| `POST` | `/api/auth/register` | ❌ | Регистрация. Создаёт user + сессию + 24 дефолтные категории. Auto-login. |
| `POST` | `/api/auth/login` | ❌ | Логин. Создаёт сессию, ставит cookie. |
| `POST` | `/api/auth/logout` | ❌ | Удаление сессии, сброс cookie. Idempotent. |
| `GET` | `/api/me` | ✅ | Текущий пользователь. |

### POST /api/auth/register

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

Валидация: `email` — валидный email, `password` — 8-72 символа (bcrypt обрезает > 72 байт).

**Response:** `201 Created` + cookie + body:
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "createdAt": "...",
  "updatedAt": "..."
}
```

Ошибки:
- `409 USER_ALREADY_EXISTS` — email занят.
- `400 VALIDATION_FAILED` — невалидный input.
- `429 TOO_MANY_REQUESTS` — превышен лимит регистраций с IP (10/час).

### POST /api/auth/login

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

**Response:** `200 OK` + cookie + body (как register).

Ошибки:
- `401 INVALID_CREDENTIALS` — неверный email или пароль (единый ответ, не раскрывает что именно).
- `429 TOO_MANY_REQUESTS` — 5 неудачных попыток в 5 минут.

### POST /api/auth/logout

Тело не нужно. Cookie читается из request, сессия удаляется из БД, cookie сбрасывается.

**Response:** `204 No Content`. Idempotent — повторный вызов с уже сброшенной cookie тоже 204.

### GET /api/me

**Response:** `200 OK`:
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "createdAt": "...",
  "updatedAt": "..."
}
```

`passwordHash` не возвращается (`json:"-"`).

**Ошибки:** `401 UNAUTHORIZED` — нет cookie или сессия невалидна.

---

## Accounts

Все endpoints требуют auth. User видит только свои accounts.

### Модель

```ts
type Account = {
  id: string
  userId: string         // владелец
  name: string
  currency: string       // USD | EUR | RUB
  openingBalance: number // int64, minor units
  manualAdjustment: number
  balance: number        // computed: openingBalance + manualAdjustment + Σ transactions
  createdAt: string
  updatedAt: string
}
```

`balance` вычисляется сервером через SQL view `account_contributions`:
- income: `+amount` на `accountId`
- expense: `-amount` на `accountId`
- transfer: `-amount` с `fromAccountId`, `+amount` на `toAccountId`

### Endpoints

| Метод | Endpoint | Описание |
|------|----------|----------|
| `GET` | `/api/accounts` | Список (с balance) |
| `POST` | `/api/accounts` | Создание |
| `GET` | `/api/accounts/:id` | Один |
| `PATCH` | `/api/accounts/:id` | Update (name, manualAdjustment) |
| `DELETE` | `/api/accounts/:id` | Удаление. **409** если есть transactions |
| `GET` | `/api/accounts/balances` | Сводка + `netWorth` |

**POST/PATCH body:**
```json
{
  "name": "Debit card",
  "currency": "USD",
  "openingBalance": 100000
}
```

**GET /api/accounts/balances response:**
```json
{
  "balances": [
    {"id": "...", "name": "...", "currency": "USD", "balance": 85000}
  ],
  "netWorth": 85000
}
```

---

## Categories

Все endpoints требуют auth. **Categories per-user** — каждый юзер видит только свои.

При регистрации пользователю копируется **24 дефолтные категории** (seed). Дальше он может их редактировать, удалять, добавлять свои.

### Модель

```ts
type Category = {
  id: string
  userId: string
  name: string
  type: "income" | "expense"
  icon: string
  color: string
  createdAt: string
  updatedAt: string
}
```

`name` уникально в рамках `(userId, name)`. Slug отсутствует — name уже на нужном языке при сидировании.

### Endpoints

| Метод | Endpoint | Описание |
|------|----------|----------|
| `GET` | `/api/categories` | Список. Query: `?type=income\|expense` |
| `POST` | `/api/categories` | Создание |
| `GET` | `/api/categories/:id` | Одна |
| `PATCH` | `/api/categories/:id` | Update |
| `DELETE` | `/api/categories/:id` | Удаление. **409** если есть transactions |

**POST body:**
```json
{
  "name": "Pet supplies",
  "type": "expense",
  "icon": "🐾",
  "color": "#FFA500"
}
```

---

## Transactions

Три типа: `income`, `expense`, `transfer`. Все endpoints требуют auth.

### Модель

```ts
type Transaction = {
  id: string
  userId: string
  type: "income" | "expense" | "transfer"
  amount: number          // int64, > 0
  description: string     // "" если не передано
  occurredAt: string
  createdAt: string
  updatedAt: string
  // Cashflow (income/expense):
  accountId?: string
  categoryId?: string
  // Transfer:
  fromAccountId?: string
  toAccountId?: string
}
```

### Endpoints

| Метод | Endpoint | Описание |
|------|----------|----------|
| `GET` | `/api/transactions` | Список с фильтрами |
| `POST` | `/api/transactions` | Создание. Сервер валидирует ссылки; **422** при нарушении |
| `GET` | `/api/transactions/:id` | Одна |
| `PATCH` | `/api/transactions/:id` | Update. `type` иммутабелен |
| `DELETE` | `/api/transactions/:id` | Удаление |

### GET /api/transactions — query параметры

| Параметр | Тип | Описание |
|----------|------|----------|
| `type` | `income\|expense\|transfer` | Фильтр по типу |
| `accountId` | `string` | Для transfer проверяет и from, и to |
| `categoryId` | `string` | Только для cashflow |
| `fromDate` | ISO date | С начала дня (включительно) |
| `toDate` | ISO date | До конца дня (включительно) |
| `limit` | `number` | Ограничение количества |
| `sort` | `string` | `occurredAt`, `-occurredAt`, `amount`, `-amount` |

### Validation rules

**Shape validation (400 VALIDATION_FAILED):**

- **Cashflow (income/expense):** `accountId`, `categoryId` обязательны. `fromAccountId`, `toAccountId` запрещены.
- **Transfer:** `fromAccountId`, `toAccountId` обязательны. `accountId`, `categoryId` запрещены.

**Referential integrity (422):**

- **Cashflow:** `accountId` существует и принадлежит user. `categoryId` существует, принадлежит user, `category.type === transaction.type`.
- **Transfer:** `fromAccountId`, `toAccountId` существуют, принадлежат user, различаются (`SAME_ACCOUNT_TRANSFER`).

**IDOR protection:** все ссылки проверяются на ownership. Чужой `accountId` → `ACCOUNT_NOT_FOUND` (не раскрываем существование).

### POST body examples

Cashflow:
```json
{
  "type": "expense",
  "amount": 1250,
  "description": "Coffee",
  "occurredAt": "2026-07-13T08:30:00Z",
  "accountId": "acc-uuid",
  "categoryId": "cat-uuid"
}
```

Transfer:
```json
{
  "type": "transfer",
  "amount": 50000,
  "description": "Move to savings",
  "occurredAt": "2026-07-13T10:00:00Z",
  "fromAccountId": "acc-1",
  "toAccountId": "acc-2"
}
```

---

## Error codes

| Code | HTTP | Когда |
|------|------|-------|
| `INVALID_REQUEST` | 400 | Malformed JSON, неверные типы |
| `VALIDATION_FAILED` | 400 | Нарушены binding правила (`required`, `gt`, `oneof`, `uuid`, и т.д.) |
| `UNAUTHORIZED` | 401 | Нет cookie или сессия невалидна |
| `INVALID_CREDENTIALS` | 401 | Неверный email или пароль при login |
| `FORBIDDEN` | 403 | Запрещено (зарезервировано) |
| `USER_ALREADY_EXISTS` | 409 | Email уже зарегистрирован |
| `ACCOUNT_NOT_FOUND` | 404 | Account не найден или чужой |
| `CATEGORY_NOT_FOUND` | 404 | Category не найдена или чужая |
| `TRANSACTION_NOT_FOUND` | 404 | Transaction не найдена или чужая |
| `ACCOUNT_IN_USE` | 409 | На account есть transactions (DELETE) |
| `CATEGORY_IN_USE` | 409 | На category есть transactions (DELETE) |
| `SAME_ACCOUNT_TRANSFER` | 422 | `fromAccountId === toAccountId` |
| `INVALID_REFS` | 422 | Null/несоответствующие refs в transaction |
| `CATEGORY_TYPE_MISMATCH` | 422 | `transaction.type` ≠ `category.type` |
| `TOO_MANY_REQUESTS` | 429 | Превышен rate limit (login/register) |
| `INTERNAL_ERROR` | 500 | Внутренняя ошибка сервера |

### Замечания по кодам

- **404 vs 422 для `ACCOUNT_NOT_FOUND`:** 404 — прямой доступ по id; 422 — account указан как FK в transaction.
- **401 единый:** не различаем «нет cookie» и «сессия истекла» — не раскрываем существование.
- **`INVALID_CREDENTIALS` единый:** не различаем «неверный email» и «неверный пароль» — защита от enumeration.

---

## Rate limits

- **Login:** 5 неудачных попыток в 5 минут на email → 429.
- **Register:** 10 регистраций в час на IP → 429.
- **Остальные endpoints:** без rate limit (пока).

---

## Cookie & CSRF

- Cookie `session_id` с `HttpOnly`, `Secure` (config), `SameSite=Lax`.
- `SameSite=Lax` закрывает CSRF для большинства случаев JSON API.
- Фронтенд **не должен** читать session_id (httpOnly) или класть его в localStorage.

---

## Schema migrations

Используется `golang-migrate`. Файлы в `internal/storage/sqlite/migrations/`:

```
000001_init.{up,down}.sql
000002_create_users.{up,down}.sql
000003_categories_add_user_id.{up,down}.sql
000004_create_sessions.{up,down}.sql
000005_accounts_transactions_add_user_id.{up,down}.sql
```

Команды:
```bash
make migrate-up
make migrate-down -all    # полный откат
make migrate-create name=add_xxx
```

Миграции встраиваются в бинарь через `//go:embed`.

---

## Roadmap (не реализовано)

- **Pagination** для `/api/transactions` — cursor-based, когда > 1000 транзакций.
- **Recurring transactions** — фоновые job'ы.
- **Budgets** — месячные лимиты по категориям.
- **Health/metrics** — `/healthz`, `/readyz`, `/metrics` для K8s.
- **Email verification** — если понадобится.
