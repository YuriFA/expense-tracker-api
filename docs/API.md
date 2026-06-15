# API Endpoints

Справочник по endpoints приложения. Заменяет хранение в localStorage серверным API.

## Соглашения

### Базовый путь

Все endpoints начинаются с `/api`.

### Поля, генерируемые сервером

- `id` — строковый идентификатор (UUID или аналог)
- `createdAt` — ISO 8601, момент создания
- `updatedAt` — ISO 8601, момент последнего обновления

> Поле `updatedAt` сейчас есть только у транзакций. При переходе на API добавить и к Account, и к Category для консистентности.

### Коды ответов

| Код | Значение |
|---|---|
| `200` | Успех (GET, PATCH) |
| `201` | Создано (POST) |
| `204` | Успешное удаление (DELETE) |
| `400` | Неверный payload (синтаксис/типы) |
| `403` | Действие запрещено (например, редактирование дефолтной категории) |
| `404` | Ресурс не найден |
| `409` | Конфликт ссылок (удаление с привязанными транзакциями) |
| `422` | Нарушение бизнес-правил (невалидные ссылки в транзакции) |


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

### DELETE → 204 No Content

Успешное удаление возвращает пустое тело.

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
  balance: number // вычисляется сервером: openingBalance + manualAdjustment + Σ транзакций
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
| `GET` | `/api/accounts/balances` | Сводка `{ [accountId]: balance }` + общий net worth |

### Пример тела запроса (POST)

```json
{
  "name": "Debit card",
  "openingBalance": 1000
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
| `GET` | `/api/categories` | Список категорий. Query: `?type=income|expense`, `?source=default|user` |
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
  description?: string
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
| `PATCH` | `/api/transactions/:id` | Обновление. Сервер обновляет `updatedAt` |
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

### Правила валидации ссылок (соответствует `hasValidTransactionReferences`)

- **Cashflow:** `accountId` существует; `categoryId` существует и `category.type === transaction.type`
- **Transfer:** `fromAccountId` и `toAccountId` существуют и различаются

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

## Auth (задел, не реализуется сейчас)

Приложение сейчас single-user. Зарезервировать префикс `/api/auth/*` и поле `userId` в таблицах, чтобы при добавлении multi-user не менять контракты.

| Метод | Endpoint | Описание |
|---|---|---|
| `POST` | `/api/auth/register` | Регистрация |
| `POST` | `/api/auth/login` | Вход |
| `POST` | `/api/auth/logout` | Выход |
| `GET` | `/api/auth/me` | Текущий пользователь |

---

## Миграция с localStorage

После реализации API на клиенте:

1. Заменить `useStorage` в Pinia stores на API-вызовы с состояниями `loading`/`error`.
2. Удалить `parse*Storage`/`serialize*Storage` (оставить только как one-time migration seed).
3. `getTransactions` упрощается — фильтры уходят в query-параметры.
4. `useAccountBalances` упрощается — берёт готовый `balance` из ответа.
5. `src/entities/category/defaults.ts` → использовать как seed-скрипт для бэкенда (без локализации имён, они остаются в i18n).
