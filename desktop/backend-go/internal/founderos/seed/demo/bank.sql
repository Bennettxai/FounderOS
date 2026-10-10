PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE bank_summaries (
    account TEXT NOT NULL,
    business TEXT NOT NULL,
    month TEXT NOT NULL,
    credits_cents INTEGER NOT NULL,
    debits_cents INTEGER NOT NULL,
    net_cents INTEGER NOT NULL,
    PRIMARY KEY (account, month)
  );
COMMIT;
