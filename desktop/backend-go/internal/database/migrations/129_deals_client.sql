-- 129_deals_client.sql
-- Link deals to their client. The deals table predates the client-nested deal
-- endpoints; CreateClientDeal now stamps client_id (clients_deals.go) so a
-- client's deals show on its list and board.
ALTER TABLE deals ADD COLUMN IF NOT EXISTS client_id UUID REFERENCES clients(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_deals_client ON deals(client_id);
