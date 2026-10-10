package finances

import (
	"reflect"
	"strings"
	"testing"
)

// Ports of FounderOS v1 tests/cards.test.ts, statements.test.ts and
// bank-statements.test.ts.

func TestCardLanesAreTheThreeCards(t *testing.T) {
	var ids []string
	for _, c := range CardLanes {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, []string{"gold", "platinum", "blue"}) {
		t.Fatalf("lanes = %v", ids)
	}
	if !IsCardID("gold") || !IsCardID("blue") || IsCardID("amex") || IsCardID("") {
		t.Fatal("IsCardID accepts only the three ids")
	}
	for _, c := range CardLanes {
		if CardLabel(c.ID) != c.Label {
			t.Fatalf("label %s", c.ID)
		}
	}
	// v1 lib/cards.ts blurbs: which physical card is a deployment detail, never recorded.
	var blurbs []string
	for _, c := range CardLanes {
		blurbs = append(blurbs, c.Blurb)
	}
	if !reflect.DeepEqual(blurbs, []string{"Personal-life spend", "Business general + cohort programmes", "Second-entity (Vantage) spend"}) {
		t.Fatalf("blurbs = %q", blurbs)
	}
}

func TestNormalizeCardID(t *testing.T) {
	cases := map[string]string{"blue": "blue", "  GOLD ": "gold", "nonsense": DefaultCard, "": DefaultCard,
		// the first pass's lane names carry forward instead of dropping rows
		"business": "platinum", "vantage": "blue"}
	for in, want := range cases {
		if got := NormalizeCardID(in); got != want {
			t.Errorf("NormalizeCardID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectCard(t *testing.T) {
	cases := map[string]string{
		"Business Blue Business Card  Prepared for ALEX EXAMPLE": "blue",
		"VANTAGE LLC  Prepared for ALEX EXAMPLE":                 "blue",
		"American Express® Gold Card":                            "gold",
		"The Platinum Card® Prepared for ALEX EXAMPLE":           "platinum",
		"launchpad-cohort-july-2026.pdf":                         "platinum",
		"Statement of account":                                   "",
	}
	for in, want := range cases {
		if got := DetectCard(in); got != want {
			t.Errorf("DetectCard(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseStatementCSVSignedAmounts(t *testing.T) {
	csv := strings.Join([]string{
		"Date,Description,Amount",
		`06/15/2026,"AMAZON WEB SERVICES",-57.00`,
		`06/14/2026,"STRIPE PAYOUT",1234.56`,
		"2026-06-13,Coffee,-4.50",
	}, "\n")
	want := []ParsedRow{
		{Date: "2026-06-15", Description: "AMAZON WEB SERVICES", AmountCents: 5700, Direction: "out"},
		{Date: "2026-06-14", Description: "STRIPE PAYOUT", AmountCents: 123456, Direction: "in"},
		{Date: "2026-06-13", Description: "Coffee", AmountCents: 450, Direction: "out"},
	}
	if got := ParseStatementCSV(csv); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseStatementCSVDebitCreditColumns(t *testing.T) {
	csv := "Transaction Date,Details,Debit,Credit\n06/01/2026,RENT,2000.00,\n06/02/2026,CLIENT PAYMENT,,5000.00"
	want := []ParsedRow{
		{Date: "2026-06-01", Description: "RENT", AmountCents: 200000, Direction: "out"},
		{Date: "2026-06-02", Description: "CLIENT PAYMENT", AmountCents: 500000, Direction: "in"},
	}
	if got := ParseStatementCSV(csv); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseStatementCSVParenthesesAreOut(t *testing.T) {
	got := ParseStatementCSV("Date,Description,Amount\n" + `06/10/2026,Adobe,"($52.99)"`)
	want := ParsedRow{Date: "2026-06-10", Description: "Adobe", AmountCents: 5299, Direction: "out"}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseStatementCSVSkipsJunkAndUnknownHeaders(t *testing.T) {
	rows := ParseStatementCSV(strings.Join([]string{"Date,Description,Amount", "", "06/15/2026,OK,-1.00", "garbage,row,notanumber", ",,"}, "\n"))
	if len(rows) != 1 || rows[0].Description != "OK" {
		t.Fatalf("got %+v", rows)
	}
	if len(ParseStatementCSV("")) != 0 || len(ParseStatementCSV("foo,bar\n1,2")) != 0 {
		t.Fatal("no understood header → no rows")
	}
}

func TestParseStatementCSVMultilineQuotedFields(t *testing.T) {
	csv := strings.Join([]string{"Date,Description,Amount", `06/15/2026,"AWS`, `extended details line two",57.00`, "06/16/2026,Coffee,4.50"}, "\n")
	rows := ParseStatementCSV(csv)
	if len(rows) != 2 || !strings.Contains(rows[0].Description, "AWS") || rows[0].Date != "2026-06-15" || rows[1].Description != "Coffee" {
		t.Fatalf("got %+v", rows)
	}
}

func TestParseStatementCSVCardConventionFlipsSign(t *testing.T) {
	csv := strings.Join([]string{
		"Date,Description,Card Member,Amount,Appears On Your Statement As",
		"06/15/2026,OPENAI,ALEX EXAMPLE,52.99,OPENAI",
		"06/10/2026,AUTOPAY PAYMENT - THANK YOU,ALEX EXAMPLE,-2000.00,AUTOPAY",
	}, "\n")
	rows := ParseStatementCSV(csv)
	if len(rows) != 2 || rows[0].Direction != "out" || rows[0].AmountCents != 5299 || rows[1].Direction != "in" {
		t.Fatalf("got %+v", rows)
	}
}

func TestCategorize(t *testing.T) {
	out := func(d string) string {
		return Categorize(ParsedRow{Date: "2026-06-01", Description: d, AmountCents: 100, Direction: "out"})
	}
	for d, want := range map[string]string{"AMAZON WEB SERVICES": "Infrastructure", "Facebook Ads": "Advertising", "OpenAI subscription": "Software", "UPWORK contractor": "Contractors", "Some Random Merchant LLC": "Uncategorized"} {
		if got := out(d); got != want {
			t.Errorf("%s → %s, want %s", d, got, want)
		}
	}
	if Categorize(ParsedRow{Description: "STRIPE PAYOUT", Direction: "in"}) != "Income" {
		t.Fatal("inbound rows are Income")
	}
	row := ParseStatementCSV("Date,Description,Card Member,Amount,Category\n06/15/2026,SOMECO STORE,FOUNDEROS,40.00,Merchandise & Supplies-Internet Purchase")[0]
	if row.SourceCategory != "Merchandise & Supplies" || Categorize(row) != "Merchandise & Supplies" {
		t.Fatalf("export category: %+v", row)
	}
	if Categorize(ParsedRow{Description: "FACEBK ADS", Direction: "out", SourceCategory: "Business Services"}) != "Advertising" {
		t.Fatal("keyword rules win over the export category")
	}
}

const amexText = `
The Platinum Card®
Prepared for
ALEX EXAMPLE
Closing Date 07/26/26        Account Ending 3-71005

Payments and Credits
07/03/26*  ONLINE PAYMENT - THANK YOU                              -$4,210.55

New Charges
07/01/26   ANTHROPIC*CLAUDE       SAN FRANCISCO CA           $200.00
07/04/26   AMAZON.COM*RT4G2       SEATTLE WA                  $23.45
07/14/26   DELTA AIR LINES        ATLANTA GA                 $612.40
Total New Charges                                          $2,041.85
`

func TestParseCardStatementText(t *testing.T) {
	rows := ParseCardStatementText(amexText)
	var outs int
	var anthropic, payment *ParsedRow
	for i, r := range rows {
		if r.Direction == "out" {
			outs++
		}
		if strings.HasPrefix(r.Description, "ANTHROPIC") {
			anthropic = &rows[i]
		}
		if strings.Contains(r.Description, "ONLINE PAYMENT") {
			payment = &rows[i]
		}
		if strings.Contains(strings.ToLower(r.Description), "total new charges") {
			t.Fatal("summary lines are ignored")
		}
	}
	if anthropic == nil || anthropic.Date != "2026-07-01" || anthropic.AmountCents != 20000 || anthropic.Direction != "out" {
		t.Fatalf("anthropic = %+v", anthropic)
	}
	if outs != 3 {
		t.Fatalf("outs = %d", outs)
	}
	if payment == nil || payment.Direction != "in" || payment.AmountCents != 421055 {
		t.Fatalf("payment = %+v", payment)
	}
	r := ParseCardStatementText("Statement Date: 01/15/2027\n01/02  SPOTIFY USA  $11.99\n")
	if len(r) != 1 || r[0].Date != "2027-01-02" || r[0].AmountCents != 1199 || r[0].Direction != "out" {
		t.Fatalf("MM/DD against the statement year: %+v", r)
	}
	if len(ParseCardStatementText("hello world\nno money here")) != 0 {
		t.Fatal("non-statement text yields nothing")
	}
}

const bankSample = `Business Checking Account Statement
IntelliReach LLC
Statement Date: 04/30/2026
Account Ending: *4219 Account Name: General Operations
Statement Summary
Beginning Balance as of 04/01/2026
$14,591.71 Earned Period
Total Credits This Period
$25,899.28 Days in Statement Period
Total Debits This Period
-$21,695.58 Interest Rate1
Ending Balance as of 04/30/2026
$18,795.41`

func TestParseBankStatementSummary(t *testing.T) {
	got := ParseBankStatementSummary(bankSample)
	want := &BankSummary{Account: "4219", Business: "General Operations", Month: "2026-04", CreditsCents: 2589928, DebitsCents: 2169558, NetCents: 2589928 - 2169558}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	m := ParseBankStatementSummary(strings.Replace(bankSample, "*4219 Account Name: General Operations", "*5630 Account Name: Vantage", 1))
	if m == nil || m.Account != "5630" || m.Business != "Vantage" {
		t.Fatalf("vantage = %+v", m)
	}
	if ParseBankStatementSummary("just some random text") != nil || ParseBankStatementSummary("") != nil {
		t.Fatal("non-statements parse to nil")
	}
}

func TestParseBankStatementSummaryFromTheRouteFixture(t *testing.T) {
	// tests/bank-statement-route.test.ts: extracted text, label and value on one line.
	s := ParseBankStatementSummary(`
  Account Name: Vantage LLC
  Account Ending: *5630
  Statement Date: 04/30/2026
  Total Credits This Period      $12,500.00
  Total Debits This Period       $4,250.00
`)
	if s == nil || s.Business != "Vantage LLC" || s.Month != "2026-04" || s.CreditsCents != 1250000 || s.DebitsCents != 425000 {
		t.Fatalf("got %+v", s)
	}
}

func bsum(account, business, month string, c, d int64) BankSummary {
	return BankSummary{Account: account, Business: business, Month: month, CreditsCents: c, DebitsCents: d, NetCents: c - d}
}

func TestBusinessSeries(t *testing.T) {
	series := BusinessSeries([]BankSummary{
		bsum("4219", "General Operations", "2026-04", 2589928, 2169558),
		bsum("5630", "Vantage", "2026-04", 4000000, 1200000),
		bsum("4219", "General Operations", "2026-03", 1800000, 1500000),
		bsum("4219", "General Operations", "2026-04", 2589928, 2169558),
	})
	if len(series) != 2 || series[0].Business != "General Operations" || series[1].Business != "Vantage" {
		t.Fatalf("series = %+v", series)
	}
	var months []string
	for _, m := range series[0].Months {
		months = append(months, m.Month)
	}
	if !reflect.DeepEqual(months, []string{"2026-03", "2026-04"}) {
		t.Fatalf("months = %v", months)
	}
	if BusinessSeries(nil) == nil {
		t.Fatal("empty is [], not null")
	}
}

func TestBankWorkspace(t *testing.T) {
	for biz, want := range map[string]string{"Vantage": "vantage", "Vantage LLC": "vantage", "General Operations": "launchpad-cohort", "": "launchpad-cohort"} {
		if got := BankWorkspace(biz); got != want {
			t.Errorf("BankWorkspace(%q) = %s, want %s", biz, got, want)
		}
	}
	for card, want := range map[string]string{"gold": "personal", "blue": "vantage", "platinum": "launchpad-cohort"} {
		if got := CardWorkspace(card); got != want {
			t.Errorf("CardWorkspace(%q) = %s, want %s", card, got, want)
		}
	}
}

// A date that is not on the calendar is skipped like any junk row. Postgres
// would refuse it at $3::date and roll back the whole statement upload, where
// FounderOS v1's SQLite stored the text; skipping the one row keeps the rest.
func TestParseStatementCSVSkipsImpossibleDates(t *testing.T) {
	rows := ParseStatementCSV(strings.Join([]string{"Date,Description,Amount",
		"06/15/2026,OK,-1.00", "13/45/2026,BAD MONTH,-2.00", "2026-02-30,BAD DAY,-3.00", "02/29/2028,LEAP,-4.00"}, "\n"))
	if len(rows) != 2 || rows[0].Description != "OK" || rows[1].Description != "LEAP" {
		t.Fatalf("got %+v", rows)
	}
}
