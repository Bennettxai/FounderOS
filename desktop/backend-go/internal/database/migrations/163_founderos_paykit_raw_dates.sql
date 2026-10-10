-- 163: PayKit last_transaction_date is kept as the raw text PayKit sent,
-- as FounderOS v1 stores it. As TIMESTAMPTZ, values PayKit sends without an
-- offset ("2026-10-01 09:00:00") became NULL and the rest read back rewritten.
-- Existing instants become RFC 3339 in PayKit's zone (what the bridge used
-- to return). Idempotent: a TEXT column is left alone.
DO $$
DECLARE
    tz TEXT := current_setting('TimeZone');
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'founderos_paykit_customer_snapshots'
                 AND column_name = 'last_transaction_date'
                 AND data_type <> 'text') THEN
        PERFORM set_config('TimeZone', 'America/Chicago', true);
        ALTER TABLE founderos_paykit_customer_snapshots
            ALTER COLUMN last_transaction_date TYPE TEXT
            USING to_char(last_transaction_date, 'YYYY-MM-DD"T"HH24:MI:SSTZH:TZM');
        PERFORM set_config('TimeZone', tz, true);
    END IF;
END $$;
