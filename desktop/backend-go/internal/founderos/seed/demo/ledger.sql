PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE ledger_rows (
    hash TEXT PRIMARY KEY,
    date TEXT NOT NULL,
    description TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    direction TEXT NOT NULL,
    category TEXT NOT NULL
  , card TEXT NOT NULL DEFAULT 'platinum');
COMMIT;
