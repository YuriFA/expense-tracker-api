CREATE TABLE categories_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE,
    type TEXT NOT NULL CHECK (type IN (
        'income',
        'expense'
    )),
    icon TEXT NOT NULL,
    color TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO categories_old
SELECT
    id,
    name,
    NULL AS slug,
    type,
    icon,
    color,
    0 AS is_default,
    created_at,
    updated_at
FROM categories;
DROP TABLE categories;
ALTER TABLE categories_old RENAME TO categories;
