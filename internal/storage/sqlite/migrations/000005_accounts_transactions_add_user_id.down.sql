DROP VIEW IF EXISTS account_contributions;

CREATE TABLE accounts_old (
    id TEXT PRIMARY KEY,
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
INSERT INTO accounts_old
SELECT
    id,
    name,
    opening_balance,
    manual_adjustment,
    currency,
    created_at,
    updated_at
FROM accounts;
DROP TABLE accounts;
ALTER TABLE accounts_old RENAME TO accounts;

CREATE TABLE transactions_old (
    id TEXT PRIMARY KEY,
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
INSERT INTO transactions_old
SELECT
    id,
    type,
    amount,
    description,
    occurred_at,
    created_at,
    updated_at,
    account_id,
    category_id,
    from_account_id,
    to_account_id
FROM transactions;
DROP TABLE transactions;
ALTER TABLE transactions_old RENAME TO transactions;

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
