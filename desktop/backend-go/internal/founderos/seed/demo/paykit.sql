PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE paykit_customer_snapshots (
    account TEXT NOT NULL,
    captured_on TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    total_spent_cents INTEGER NOT NULL,
    total_transactions INTEGER NOT NULL,
    last_transaction_date TEXT,
    source TEXT NOT NULL,
    PRIMARY KEY (account, captured_on, customer_id)
  );
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900001',250000,2,NULL,'reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900002',500000,1,'2025-07-01T12:00:00Z','reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900003',250000,1,'2025-08-15T12:00:00Z','reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900004',750000,3,'2025-11-01T12:00:00Z','reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900005',100000,1,'2026-01-15T12:00:00Z','reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900006',500000,1,'2026-03-10T12:00:00Z','reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900007',400000,2,'2026-06-05T12:00:00Z','reconstructed');
INSERT INTO paykit_customer_snapshots VALUES('paykit-lc','2026-08-20','900008',150000,1,'2026-08-01T12:00:00Z','reconstructed');
COMMIT;
