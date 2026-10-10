package paykit

// Seed20260820 is the 2026-08-20 snapshot, RECONSTRUCTED (FounderOS v1
// lib/paykit-history.ts SEED_2026_08_20): the earliest datapoint that exists,
// and the one that makes September 2026 exact rather than a $0 – $2,000 band.
//
// FOS-234 recorded only the aggregate that day: 31 customers · 42 transactions ·
// $96,489.00, spanning 2025-07-10 → 2026-08-09. The per-customer split is
// recovered by differencing backwards from the 2026-09-17 live pull (31 · 43 ·
// $97,489.00): same customer count, lifetime totals only rise, and exactly one
// customer (452730) has a transaction after 2026-08-09, so that customer gained
// one $1,000 transaction and every other row is unchanged. 452730's prior
// transaction date is genuinely unknown and is nil rather than invented.
//
// Marked SourceReconstructed so it is never read as a measurement. Ids and
// numbers only; no PII.
var Seed20260820 = Snapshot{
	CapturedOn: "2026-08-20",
	Source:     SourceReconstructed,
	Customers: seedCustomers([]seedRow{
		{"159025", 250000, 5, "2025-12-31T10:41:08-06:00"},
		{"402026", 500000, 1, "2025-07-10T18:37:08-05:00"},
		{"449908", 250000, 1, "2025-07-31T14:24:23-05:00"},
		{"449911", 250000, 1, "2025-07-31T14:24:59-05:00"},
		{"452730", 100000, 2, ""},
		{"460693", 500000, 1, "2025-08-04T17:28:33-05:00"},
		{"465263", 233400, 1, "2025-08-14T07:22:10-05:00"},
		{"469582", 400000, 1, "2025-08-07T16:42:22-05:00"},
		{"472960", 250000, 1, "2025-08-09T14:50:31-05:00"},
		{"545383", 750000, 3, "2025-11-11T08:10:04-06:00"},
		{"554052", 100000, 1, "2025-09-07T19:45:03-05:00"},
		{"599664", 500000, 1, "2025-09-24T19:58:22-05:00"},
		{"624264", 850000, 1, "2025-10-31T09:19:54-05:00"},
		{"630096", 62500, 1, "2025-10-09T14:39:29-05:00"},
		{"633866", 125000, 2, "2025-10-07T13:51:53-05:00"},
		{"741017", 250000, 1, "2025-11-11T06:55:50-06:00"},
		{"771691", 680000, 2, "2026-03-18T11:49:02-05:00"},
		{"786228", 8000, 1, "2025-11-27T11:50:36-06:00"},
		{"1187092", 100000, 1, "2026-03-07T19:36:11-06:00"},
		{"1187179", 150000, 3, "2026-05-06T20:21:09-05:00"},
		{"1197108", 500000, 1, "2026-03-10T14:26:25-05:00"},
		{"1209411", 500000, 1, "2026-03-12T18:25:46-05:00"},
		{"1223996", 40000, 1, "2026-03-16T12:43:46-05:00"},
		{"1264592", 460000, 1, "2026-03-25T19:01:25-05:00"},
		{"1376644", 50000, 1, "2026-04-20T13:42:51-05:00"},
		{"1487163", 500000, 1, "2026-04-29T12:23:30-05:00"},
		{"1562298", 500000, 1, "2026-05-13T14:03:58-05:00"},
		{"1688800", 340000, 1, "2026-06-10T12:12:13-05:00"},
		{"1922197", 150000, 1, "2026-08-09T15:15:50-05:00"},
		{"1922438", 150000, 1, "2026-08-09T15:29:13-05:00"},
		{"1922443", 150000, 1, "2026-08-09T15:30:30-05:00"},
	}),
}

type seedRow struct {
	id    string
	cents int64
	txns  int
	at    string
}

func seedCustomers(rows []seedRow) []Customer {
	out := make([]Customer, 0, len(rows))
	for _, r := range rows {
		c := Customer{ID: r.id, TotalSpentCents: r.cents, Transactions: r.txns}
		if r.at != "" {
			at, m := r.at, r.at[:7]
			c.LastTransactionDate, c.Month = &at, &m
		}
		out = append(out, c)
	}
	return out
}
