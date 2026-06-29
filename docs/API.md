# API Endpoints

Справочник по endpoints приложения. Заменяет хранение в localStorage серверным API.

## Соглашения

### Базовый путь

Все endpoints начинаются с `/api`.

### Поля, генерируемые сервером

- `id` — строковый идентификатор (UUID или аналог)
- `createdAt` — ISO 8601, момент создания
- `updatedAt` — ISO 8601, момент последнего обновления

### Коды ответов

| Код | Значение |
|---|---|
| `200` | Успех (GET, PATCH) |
| `201` | Создано (POST) |
| `204` | Успешное удаление (DELETE) |
| `400` | Неверный payload (синтаксис/типы/валидация) |
| `401` | Не авторизован (после реализации auth — см. Roadmap) |
| `403` | Действие запрещено (например, редактирование дефолтной категории) |
| `404` | Ресурс не найден |
| `409` | Конфликт ссылок (удаление с привязанными транзакциями) |
| `422` | Нарушение бизнес-правил (невалидные ссылки в транзакции) |
| `500` | Внутренняя ошибка сервера (`INTERNAL_ERROR`) |


### Формат ошибок

Все ошибки возвращаются в единой форме:

\`\`\`json
{
  "code": "ACCOUNT_NOT_FOUND",
  "message": "account not found"
}
\`\`\`

`code` — машиночитаемая строка для switch'а на клиенте.
`message` — человекочитаемое описание.

### Ошибки валидации

Когда тело запроса не проходит валидацию, сервер возвращает детали по каждому полю:

```json
{
  "code": "VALIDATION_FAILED",
  "message": "validation failed",
  "errors": [
    {"field": "name", "message": "name is required"},
  ]
}
```
Поле errors присутствует только при validation-ошибках. В остальных случаях (404, 500) возвращаются только code и message.

### Различие `INVALID_REQUEST` и `VALIDATION_FAILED`

Оба возвращают HTTP 400 и одинаковый формат с `errors[]`, но разные `code`:

- **`INVALID_REQUEST`** — malformed JSON, некорректные типы полей (например, `amount: "abc"` вместо числа), либо body отсутствует. Структура запроса не прошла парсинг.
- **`VALIDATION_FAILED`** — парсинг успешен, но нарушены правила `binding`-тегов (например, `required`, `gt=0`, `oneof`, `uuid`). Структура валидна по типам, но не по правилам.

Клиент может различать их, например, для разных UX-сообщений («некорректный запрос» vs «проверьте поля»).

### Формат дат

ISO 8601 (как текущее поле `occurredAt`).

---

## Accounts

Баланс счёта считается на сервере и возвращается полем `balance` в ответе.

### Модель

```ts
type Account = {
  id: string
  name: string
  openingBalance: number
  manualAdjustment: number
  balance: number // вычисляется сервером: openingBalance + manualAdjustment +
                  //   income: +amount, expense: −amount (по accountId)
                  //   transfer: −amount с fromAccountId, +amount на toAccountId
  createdAt: string
  updatedAt: string
}
```

### Endpoints

| Метод | Endpoint | Описание |
|---|---|---|
| `GET` | `/api/accounts` | Список счетов (с `balance`) |
| `POST` | `/api/accounts` | Создание счёта. Сервер проставляет `manualAdjustment: 0` |
| `GET` | `/api/accounts/:id` | Один счёт (с `balance`) |
| `PATCH` | `/api/accounts/:id` | Обновление полей счёта |
| `DELETE` | `/api/accounts/:id` | Удаление. **409**, если есть привязанные транзакции |
| `GET` | `/api/accounts/balances` | Сводка балансов + общий `netWorth` (см. модель ниже) |

### Пример тела запроса (POST)

```json
{
  "name": "Debit card",
  "openingBalance": 1000
}
```

### Ответ `GET /api/accounts/balances`

```ts
type AccountBalancesResponse = {
  balances: AccountBalance[]
  netWorth: number // Σ всех balances
}

type AccountBalance = {
  id: string
  name: string
  balance: number
}
```

```json
{
  "balances": [
    { "id": "acc_1", "name": "Debit card", "balance": 850 },
    { "id": "acc_2", "name": "Savings",    "balance": 5150 }
  ],
  "netWorth": 6000
}
```

---

## Categories

Дефолтные категории (24 шт.) сидируются на бэкенде при инициализации. Пользовательские создаются через API. Дефолтные доступны только на чтение.

### Модель

```ts
type Category = {
  id: string
  name: string
  slug?: string // только для дефолтных категорий (isDefault: true); используется клиентом как i18n-ключ
  type: 'income' | 'expense'
  icon: string
  color: string
  isDefault: boolean // true для сидированных, false для пользовательских
  createdAt: string
  updatedAt: string
}
```

### Endpoints

| Метод | Endpoint | Описание |
|---|---|---|
| `GET` | `/api/categories` | Список категорий. Query: `?type=income|expense` |
| `POST` | `/api/categories` | Создание пользовательской категории |
| `GET` | `/api/categories/:id` | Одна категория |
| `PATCH` | `/api/categories/:id` | Обновление. Для `isDefault: true` → **403** |
| `DELETE` | `/api/categories/:id` | Удаление. Для `isDefault: true` → **403**; есть транзакции → **409** |

### Пример тела запроса (POST)

```json
{
  "name": "Pet supplies",
  "type": "expense",
  "icon": "🐾",
  "color": "#FFA500"
}
```

### Локализация дефолтных категорий

Дефолтные категории сидируются без локализованных имён. На клиенте имя берётся из i18n по `id` категории (сохраняется текущая схема `defaults.ts` + `i18n.global.t`). Бэкенд хранит только `id`, `icon`, `color`, `type` для дефолтных.

---

## Transactions

Транзакции трёх типов: `income`, `expense`, `transfer`. Серверная валидация ссылочной целостности на account/category.

### Модель

```ts
type BaseTransaction = {
  id: string
  type: 'income' | 'expense' | 'transfer'
  amount: number
  description?: string // если не присласть → сервер выставит "" (NOT NULL DEFAULT '' в storage). Семантически empty string ≈ absent.
  occurredAt: string
  createdAt: string
  updatedAt: string
}

type CashflowTransaction = BaseTransaction & {
  type: 'income' | 'expense'
  accountId: string
  categoryId: string
}

type TransferTransaction = BaseTransaction & {
  type: 'transfer'
  fromAccountId: string
  toAccountId: string
}

type Transaction = CashflowTransaction | TransferTransaction
```

### Endpoints

| Метод | Endpoint | Описание |
|---|---|---|
| `GET` | `/api/transactions` | Список с фильтрами (см. ниже) |
| `POST` | `/api/transactions` | Создание. Сервер валидирует ссылки; **422** при нарушении |
| `GET` | `/api/transactions/:id` | Одна транзакция |
| `PATCH` | `/api/transactions/:id` | Обновление. `type` иммутабелен; сервер обновляет `updatedAt` |
| `DELETE` | `/api/transactions/:id` | Удаление |

### Query-параметры `GET /api/transactions`

| Параметр | Тип | Описание |
|---|---|---|
| `type` | `income|expense|transfer` | Фильтр по типу |
| `accountId` | `string` | Для transfer проверяет и `fromAccountId`, и `toAccountId` |
| `categoryId` | `string` | Только для cashflow |
| `fromDate` | `CalendarDay` (ISO date) | С начала дня (включительно) |
| `toDate` | `CalendarDay` (ISO date) | До конца дня (включительно) |
| `limit` | `number` | Ограничение количества записей |
| `sort` | `string` | По умолчанию `-occurredAt` (как в сторе) |

### Правила валидации

**По типу (shape) — 400 `VALIDATION_FAILED`:**

- **Cashflow (`income`/`expense`):** `accountId`, `categoryId` обязательны; `fromAccountId`, `toAccountId` запрещены
- **Transfer:** `fromAccountId`, `toAccountId` обязательны; `accountId`, `categoryId` запрещены

**Ссылочная целостность — 422:**

- **Cashflow:** `accountId` существует; `categoryId` существует и `category.type === transaction.type`
- **Transfer:** `fromAccountId` и `toAccountId` существуют и различаются (`SAME_ACCOUNT_TRANSFER`)

**PATCH-семантика:**

- Поле `type` иммутабельно — его нельзя изменить после создания. Попытка передать `type` в теле PATCH игнорируется (поле отсутствует в `UpdateTransactionRequest`).
- Тип транзакции определяет, какие ссылочные поля валидны (см. shape-правила выше): нельзя PATCH'ем добавить `fromAccountId`/`toAccountId` в cashflow-транзакцию или `accountId`/`categoryId` в transfer.
- Ссылочные поля, не соответствующие `type`, остаются `null` в storage.

### Примеры тела запроса

Cashflow (POST):

```json
{
  "type": "expense",
  "amount": 12.50,
  "description": "Coffee",
  "occurredAt": "2026-06-14T08:30:00.000Z",
  "accountId": "acc_1",
  "categoryId": "cat_1"
}
```

Transfer (POST):

```json
{
  "type": "transfer",
  "amount": 500,
  "description": "Move to savings",
  "occurredAt": "2026-06-14T10:00:00.000Z",
  "fromAccountId": "acc_1",
  "toAccountId": "acc_2"
}
```

---

## Roadmap (следующие этапы)

Этапы развития. Каждый — с **trigger** (когда делать) и **trade-offs**.
Подход pain-driven: этап активируется, когда trigger срабатывает, не раньше.

### 1. Авторизация (multi-user)

**Цель:** каждый пользователь видит и редактирует только свои данные.

**Endpoints** (префикс `/api/auth/*`):

| Метод | Endpoint | Описание |
|---|---|---|
| `POST` | `/api/auth/register` | Регистрация (`{email, password}` → `201 {userId}`) |
| `POST` | `/api/auth/login` | Вход (`{email, password}` → `200 {token}` или cookie) |
| `POST` | `/api/auth/logout` | Инвалидация token/session |
| `GET` | `/api/auth/me` | Текущий пользователь |

**Архитектурные решения (trade-offs):**

- **Auth method:**
  - JWT (stateless) — горизонтально масштабируется, но token нельзя отозвать без blacklist.
  - Session cookies (stateful) — отзыв легко, но требует shared session store для multi-instance.
- **Password storage:** `bcrypt` или `argon2` (никогда plaintext или MD5/SHA без salt).
- **Token storage на клиенте:**
  - HTTP-only cookie — защита от XSS, но требует CSRF-защиты.
  - localStorage — удобно для SPA, но уязвим к XSS.

**Storage migration:**
- `accounts`, `categories` (пользовательские), `transactions` — добавить `user_id INTEGER REFERENCES users(id)`.
- **Default categories** остаются **shared** (без `user_id`) — видны всем пользователям.

**Triggers:**
- Появление 2+ реальных пользователей.
- Подключение фронтенда с login flow.
- Публичный deployment (даже для одного пользователя — безопасность данных).

---

### 2. Деньги (int64 минорных единиц)

**Цель:** точность до копейки/цента, без `0.1 + 0.2 = 0.30000000000000004` артефактов.

**Production-подходы (trade-offs):**

| Подход | Example | Pros | Cons |
|---|---|---|---|
| **int64 центов** | `1250` = $12.50 | Точность, performance, simple | Breaking change API contract |
| **Decimal type** (shopspring/decimal) | `decimal.New(1250, -2)` | Удобство, точность, multi-currency | Внешняя зависимость, медленнее |
| **String representation** | `"12.50"` | Точность | Serialisation hassle всем клиентам |
| **Custom JSON type** | Внутри int64, снаружи `"12.50"` | Backward-compatible, точность | Больше кода (MarshalJSON/UnmarshalJSON) |

**Рекомендация:** `int64` центов + custom JSON marshalling (внутри int64, в API как `12.50`) — backward-compatible путь.

**Breaking change aspects:**
- Если перешёл на `amount: 1250` (cents) — клиенты должны знать, что число в минорных единицах.
- Если на custom marshalling `12.50` — обратно совместим для клиентов, но внутри Go и storage меняется всё.
- **Migration:** конвертация существующих `float64` значений в БД (`UPDATE transactions SET amount_cents = ROUND(amount * 100)`).
- **Testing:** обновить все assertion'ы в handler/storage тестах.

**Triggers:**
- Реальный баг с rounding (видимое расхождение на 1 цент).
- Подключение реальных клиентов (после этого breaking change дороже).

---

### 3. Pagination

**Цель:** масштабируемость GET-эндпоинтов при большом объёме данных.

**Сейчас:** только `limit` без `offset`/cursor. На 10000+ транзакций станет медленным и неудобным.

**Подходы:**

| Подход | Example | Pros | Cons |
|---|---|---|---|
| **limit + offset** | `?limit=50&offset=100` | Простота, stateless | Медленно на большом offset (DB сканирует) |
| **Cursor-based** | `?cursor=abc123` | Stable, fast, не ломается при inserts | Сложнее, cursor надо хранить |
| **Page-based** | `?page=3&page_size=50` | Знакомый UX | То же что offset под капотом |

**Рекомендация:**
- **`/api/transactions`** — cursor-based (потенциально большой объём).
- **`/api/accounts`, `/api/categories`** — `limit + offset` (малый объём, редко > 50).

**Response shape change:**
```json
{
  "data": [...],
  "nextCursor": "abc123",
  "hasMore": true
}
```
Breaking change — клиенты должны адаптироваться.

**Triggers:**
- Количество транзакций > 1000.
- Видимая задержка GET-запросов.
- Фронтенд добавляет infinite scroll или "load more".

---

### 4. Регулярные платежи (recurring transactions)

**Цель:** автоматическое создание транзакций по расписанию (зарплата, rent, subscriptions).

**Схема:**
- Отдельная таблица `recurring_transactions` (`id, userId, type, amount, categoryId, accountId, frequency, nextDate, endDate`).
- `frequency` — `daily | weekly | monthly | yearly`.
- Background job (cron/ticker) раз в день создаёт transaction по расписанию.

**Endpoints:**
- `POST/GET/PATCH/DELETE /api/recurring`.

**Complexity:** высокая. Требует scheduler (`robfig/cron` или `time.Ticker` в goroutine) — это новая инфраструктура. Также требует graceful handling crash'ей (что если день пропущен?).

**Triggers:**
- Фронтенд просит автоматизировать регулярные платежи.
- Пользователь жалуется на ручной ввод одинаковых транзакций.

---

### 5. Бюджеты (budgets)

**Цель:** месячные лимиты по категориям (`Food: $500/month`) и отслеживание прогресса.

**Схема:**
- Таблица `budgets` (`id, userId, categoryId, amount, period`).
- Endpoint `GET /api/budgets?month=2024-06` возвращает `{categoryId, budgeted, spent, remaining}` per category.
- Spent считается через `SUM(amount)` по transactions за период.

**Endpoints:**
- `POST/GET/PATCH/DELETE /api/budgets`.
- `GET /api/budgets/summary?month=2024-06` — сводка.

**Complexity:** medium. Не требует background jobs, только query-логику.

**Triggers:**
- Фронтенд просит планирование расходов.
- Пользователь хочет видеть «сколько осталось на еду в этом месяце».

---

### 6. Operational (production deploy)

**Цель:** готовность к Docker/K8s deployment, observability, security.

**Что добавить:**

- **`GET /healthz`** — минимальный health check для K8s liveness probe (`200 OK` или `503` если DB down).
- **`GET /readyz`** — readiness probe (готов ли принимать трафик).
- **`GET /metrics`** — Prometheus metrics (request count, latency histogram, error rate).
- **Rate limiting** — `golang.org/x/time/rate` middleware, защита от abuse.
- **CORS** — для веб-клиента с другого origin (`Access-Control-Allow-Origin`).
- **Request ID middleware** — `X-Request-ID` header, пробрасывается в логи для tracing.
- **Graceful shutdown** — уже реализован (Волна 2), но при multi-instance нужен readiness probe drain.

**Triggers:**
- Первый production deploy (Docker/K8s).
- Подключение реального веб-клиента (CORS).
- Заметный traffic (rate limit).

---

### 7. Database migrations

**Цель:** эволюция schema без потери данных.

**Сейчас:** `CREATE TABLE IF NOT EXISTS`. Не позволяет изменять существующие таблицы (например, добавить `user_id`, конвертировать `amount` в int64).

**Options:**

| Инструмент | Pros | Cons |
|---|---|---|
| `golang-migrate/migrate` | Стандарт индустрии, SQL-файлы, CLI | Внешняя зависимость |
| `pressly/goose` | Go-friendly, можно embed migrations | Меньше ecosystem |
| In-code versioning | Простейший (`schema_version` таблица + if-branches) | Не масштабируется |

**Рекомендация:** `golang-migrate` — industry standard, хорошо документирован.

**Triggers:**
- Любой schema change на existing таблице (для auth, для int64 money).
- Появление production данных, которые нельзя потерять.

---

### 8. Архитектурная эволюция: service + repository layer

**Цель:** выделить бизнес-логику из handler'ов для testability и reuse.

**Текущее состояние** (зафиксировано в `AGENTS.md`): `handler → storage` напрямую, без service layer. Это работает для current scope, но имеет пределы.

**Когда переходить — triggers:**

Главный trigger — **появление `userId` (auth)**. При multi-user:
- `userId` нужно прокидывать во все storage methods → signature bloat (`GetAccounts(userId)`, `CreateTransaction(userId, ...)`, и т.д.).
- Permission checks («этот account принадлежит этому user?») — cross-cutting concern, нужен в каждом handler.
- Auth middleware извлекает `userId` из JWT/session, но handler'у надо знать, как его пробросить в storage.

Другие triggers:
- **Cross-entity operations:** «delete user» → cascade delete accounts, transactions, custom categories. Это service-layer operation (оркестрация нескольких storage calls в одной транзакции).
- **Сложная бизнес-логика:** budgets, recurring transactions — это доменные операции, не просто CRUD.
- **Testability:** handler-тесты с real in-memory SQLite становятся медленными или хрупкими на сложных сценариях → нужны mock'и storage interfaces.

**Что меняется:**

```
Currently:                Future:
                          
Client                    Client
  ↓                         ↓
Gin middleware            Gin middleware (auth, requestID, ...)
  ↓                         ↓
Handler                   Handler (HTTP concerns only: bind, write response)
  ↓                         ↓
Storage (sqlite)          Service (business logic, orchetration, tx)
                            ↓
                          Storage interface (Storage interface)
                            ↓
                          sqlite implementation
```

**Что появляется:**
- `internal/service/` — доменные операции (`TransactionService.Create`, `AccountService.DeleteWithChecks`).
- `internal/storage/storage.go` — interface (минимальный, только методы которые реально нужны service).
- `internal/storage/sqlite/` — реализация interface.
- Mocks для service-тестов (`mockgen`).

**Trade-offs:**

| | Pros | Cons |
|---|---|---|
| Service layer | Testability с mock'ами, reusability, чистые handler'ы | Больше слоёв, indirection, больше кода |
| Repository interface | Mock'и для service-тестов, swap storage backend | Interface duplication, update при изменении storage |
| Без service/repository (сейчас) | Simple, быстро, мало кода | Сложно mock'ать, бизнес-логика в handler'ах |

**Рекомендация:**
1. Сначала auth (пункт 1) → станет больно прокидывать `userId` везде.
2. Тогда же ввести `Storage` interface (минимальный) + service layer.
3. Mock'и появятся, когда handler-тесты с real DB перестанут покрывать edge cases.

---

### Priority matrix

| Этап | Urgency | Trigger |
|---|---|---|
| **Database migrations** | 🟡 Medium | Любой schema change (например, для auth или money) |
| **Деньги (int64)** | 🟡 Medium | Реальный rounding баг. Сам решил раньше времени — твоё право. |
| **Pagination** | 🟡 Medium | > 1000 transactions в БД |
| **Auth (multi-user)** | 🟡 Medium-High | 2+ реальных пользователей, public deployment |
| **Архитектура (service/repository)** | 🟡 Medium | Сразу после/вместе с auth — `userId` болево |
| **Operational (health, CORS)** | 🟢 Low | Production deploy, веб-клиент |
| **Recurring transactions** | ⚪ Когда фронтенд попросит | |
| **Budgets** | ⚪ Когда фронтенд попросит | |

---

## Миграция с localStorage

После реализации API на клиенте:

1. Заменить `useStorage` в Pinia stores на API-вызовы с состояниями `loading`/`error`.
2. Удалить `parse*Storage`/`serialize*Storage` (оставить только как one-time migration seed).
3. `getTransactions` упрощается — фильтры уходят в query-параметры.
4. `useAccountBalances` упрощается — берёт готовый `balance` из ответа.
5. `src/entities/category/defaults.ts` → использовать как seed-скрипт для бэкенда (без локализации имён, они остаются в i18n).

---

## Приложение: Error codes

Полный список машиночитаемых кодов ошибок, которые сервер возвращает в поле `code`. Актуально синхронизировать с `internal/http-server/handlers/errors.go`.

| Code | HTTP | Описание |
|---|---|---|
| `INVALID_REQUEST` | 400 | Malformed JSON / некорректные типы полей / body отсутствует |
| `VALIDATION_FAILED` | 400 | Парсинг успешен, нарушены правила `binding`-тегов (`required`, `gt`, `oneof`, `uuid`, и т.п.) |
| `FORBIDDEN` | 403 | Действие запрещено (например, изменение/удаление дефолтной категории) |
| `ACCOUNT_NOT_FOUND` | 404 | Account не найден по `id` (GET/PATCH/DELETE напрямую) |
| `CATEGORY_NOT_FOUND` | 404 | Category не найдена по `id` (GET/PATCH/DELETE напрямую) |
| `TRANSACTION_NOT_FOUND` | 404 | Transaction не найдена по `id` |
| `ACCOUNT_IN_USE` | 409 | На account есть привязанные transactions (DELETE) |
| `CATEGORY_IN_USE` | 409 | На category есть привязанные transactions (DELETE) |
| `SAME_ACCOUNT_TRANSFER` | 422 | `fromAccountId === toAccountId` в transfer-транзакции |
| `INVALID_REFS` | 422 | Null refs в type-specific полях (внутренняя ошибка, обычно отсекается на shape validation) |
| `CATEGORY_TYPE_MISMATCH` | 422 | `transaction.type` не совпадает с `category.type` (например, income-transaction ссылается на expense-category) |
| `INTERNAL_ERROR` | 500 | Непредвиденная ошибка сервера (логируется на уровне Error) |

### Note про `ACCOUNT_NOT_FOUND` / `CATEGORY_NOT_FOUND`

Эти коды возвращаются с разным HTTP status в зависимости от контекста:
- **404** — когда обращаешься напрямую к ресурсу по `id` (`GET /api/accounts/:id`).
- **422** — когда account/category указан как FK в transaction, и не существует (референс невалиден).

Клиент может различать по HTTP status, не только по `code`.
