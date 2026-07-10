DROP VIEW IF EXISTS account_contributions;

CREATE TABLE accounts_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    opening_balance INTEGER NOT NULL,
    manual_adjustment INTEGER NOT NULL,
    currency TEXT NOT NULL CHECK (currency IN (
        'USD',
        'EUR',
        'RUB'
    )),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;

CREATE TABLE transactions_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN (
        'income',
        'expense',
        'transfer'
    )),
    amount INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    occurred_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    account_id TEXT,
    category_id TEXT,
    from_account_id TEXT,
    to_account_id TEXT,
    FOREIGN KEY (account_id) REFERENCES accounts (id),
    FOREIGN KEY (category_id) REFERENCES categories (id),
    FOREIGN KEY (from_account_id) REFERENCES accounts (id),
    FOREIGN KEY (to_account_id) REFERENCES accounts (id)
);
DROP TABLE transactions;
ALTER TABLE transactions_new RENAME TO transactions;

CREATE VIEW account_contributions AS SELECT
    account_id,
    CASE
        WHEN type = 'income' THEN amount
        WHEN type = 'expense' THEN -amount
    END AS signed
FROM
    transactions
WHERE type IN ('income', 'expense')
UNION ALL
SELECT
    from_account_id,
    -amount AS signed
FROM
    transactions
WHERE
    type = 'transfer'
UNION ALL
SELECT
    to_account_id,
    amount
FROM
    transactions
WHERE
    type = 'transfer';
