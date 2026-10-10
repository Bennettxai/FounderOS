// Package finances is the bridge port of FounderOS v1's /finances page logic:
// card lanes (lib/cards.ts), statement parsing (lib/statements.ts,
// lib/bank-statements.ts), the ledger's derived reads (lib/ledger.ts,
// lib/spend-report.ts), the Money Volume view-model (lib/finances-volume.ts),
// the Postgres stores behind them, and the page payload.
//
// Pure where FounderOS v1 was pure: no network, no LLM. The only side effects
// live in store.go (Postgres bridge data) and pdf.go (the local pdftotext
// binary, behind an injectable runner).
package finances

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---- card lanes (lib/cards.ts) --------------------------------------------

type CardLane struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Blurb string `json:"blurb"`
}

// CardLanes are the three cards the operator spends on, in FounderOS v1's order.
var CardLanes = []CardLane{
	{"gold", "Gold · Personal", "Personal-life spend"},
	{"platinum", "Platinum · Business", "Business general + cohort programmes"},
	{"blue", "Business Blue · Vantage", "Second-entity (Vantage) spend"},
}

const DefaultCard = "platinum"

// legacyCards are the first pass's lane ids; rows under them keep their money.
var legacyCards = map[string]string{"business": "platinum", "vantage": "blue"}

func IsCardID(v string) bool {
	for _, c := range CardLanes {
		if c.ID == v {
			return true
		}
	}
	return false
}

// NormalizeCardID coerces anything (form field, query param, legacy row) into a lane.
func NormalizeCardID(raw string) string {
	t := strings.ToLower(strings.TrimSpace(raw))
	if IsCardID(t) {
		return t
	}
	if c, ok := legacyCards[t]; ok {
		return c
	}
	return DefaultCard
}

func CardLabel(id string) string {
	for _, c := range CardLanes {
		if c.ID == id {
			return c.Label
		}
	}
	return id
}

var (
	reBlue     = regexp.MustCompile(`(?i)vantage|business\s*blue|blue\s*business`)
	rePlatinum = regexp.MustCompile(`(?i)platinum|launchpad[\s_-]*cohort`)
	reGold     = regexp.MustCompile(`(?i)\bgold\b`)
)

// DetectCard is a best-effort lane from statement text or a filename; "" when unsure.
func DetectCard(text string) string {
	switch {
	case reBlue.MatchString(text):
		return "blue"
	case rePlatinum.MatchString(text):
		return "platinum"
	case reGold.MatchString(text):
		return "gold"
	}
	return ""
}

// CardWorkspace is the table map's per-row rule for founderos_ledger_rows.
func CardWorkspace(card string) string {
	switch NormalizeCardID(card) {
	case "gold":
		return "personal"
	case "blue":
		return "vantage"
	}
	return "launchpad-cohort"
}

// BankWorkspace is the table map's per-row rule for founderos_bank_summaries:
// Vantage → vantage; General Operations and anything else → launchpad-cohort.
func BankWorkspace(business string) string {
	if strings.Contains(strings.ToLower(business), "vantage") {
		return "vantage"
	}
	return "launchpad-cohort"
}

// ---- card / bank CSV and PDF text (lib/statements.ts) ---------------------

// ParsedRow is one statement line. Direction is "in" or "out".
type ParsedRow struct {
	Date           string `json:"date"`
	Description    string `json:"description"`
	AmountCents    int64  `json:"amountCents"`
	Direction      string `json:"direction"`
	SourceCategory string `json:"sourceCategory,omitempty"`
}

// tokenizeCSV splits CSV text into records, honouring quoted fields that hold
// commas and newlines (Amex "Extended Details"); "" inside quotes is a quote.
func tokenizeCSV(text string) [][]string {
	var records [][]string
	var field strings.Builder
	var record []string
	inQ, started := false, false
	rs := []rune(text)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case inQ:
			if c == '"' {
				if i+1 < len(rs) && rs[i+1] == '"' {
					field.WriteRune('"')
					i++
				} else {
					inQ = false
				}
			} else {
				field.WriteRune(c)
			}
		case c == '"':
			inQ, started = true, true
		case c == ',':
			record = append(record, field.String())
			field.Reset()
			started = true
		case c == '\n' || c == '\r':
			if c == '\r' && i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			if started || field.Len() > 0 {
				records = append(records, append(record, field.String()))
			}
			field.Reset()
			record, started = nil, false
		default:
			field.WriteRune(c)
			started = true
		}
	}
	if started || field.Len() > 0 {
		records = append(records, append(record, field.String()))
	}
	return records
}

func cell(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[i])
}

var (
	reISODate   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	reSlashDate = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{2,4})`)
)

func pad2(s string) string {
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

func year4(y string) string {
	if len(y) == 2 {
		return "20" + y
	}
	return y
}

func normDate(s string) string {
	t := strings.TrimSpace(s)
	d := ""
	if reISODate.MatchString(t) {
		d = t[:10]
	} else if m := reSlashDate.FindStringSubmatch(t); m != nil {
		d = year4(m[3]) + "-" + pad2(m[1]) + "-" + pad2(m[2])
	}
	// Only a real calendar day: Postgres refuses 2026-13-45 at ::date and
	// would roll back the whole upload for one bad row.
	if _, err := time.Parse("2006-01-02", d); err != nil {
		return ""
	}
	return d
}

var reAmountJunk = regexp.MustCompile(`[$,()\s-]`)

// parseAmount: "$1,234.56" → 1234.56; "($50.00)" → -50; "" → not ok.
func parseAmount(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return 0, false
	}
	neg := (strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")")) || strings.HasPrefix(t, "-")
	cleaned := reAmountJunk.ReplaceAllString(t, "")
	if cleaned == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
		return 0, false
	}
	if neg {
		return -math.Abs(n), true
	}
	return math.Abs(n), true
}

func findCol(headers []string, names ...string) int {
	for i, h := range headers {
		for _, n := range names {
			if strings.Contains(h, n) {
				return i
			}
		}
	}
	return -1
}

func cents(dollars float64) int64 { return int64(math.Floor(math.Abs(dollars)*100 + 0.5)) }

// ParseStatementCSV parses a bank or card CSV export into rows, skipping rows
// it cannot read rather than guessing. Card exports (Amex) list charges as
// positive, the opposite of the bank convention, and are flipped.
func ParseStatementCSV(text string) []ParsedRow {
	records := tokenizeCSV(text)
	rows := []ParsedRow{}
	if len(records) < 2 {
		return rows
	}
	headers := make([]string, len(records[0]))
	for i, h := range records[0] {
		headers[i] = strings.ToLower(strings.TrimSpace(h))
	}
	dateCol := findCol(headers, "date")
	descCol := findCol(headers, "description", "details", "memo", "name", "payee")
	amountCol := findCol(headers, "amount")
	debitCol := findCol(headers, "debit", "withdrawal")
	creditCol := findCol(headers, "credit", "deposit")
	categoryCol := findCol(headers, "category")
	cardConvention := false
	for _, h := range headers {
		if strings.Contains(h, "card member") || strings.Contains(h, "appears on your statement") {
			cardConvention = true
		}
	}
	hasAmounts := amountCol >= 0 || (debitCol >= 0 && creditCol >= 0)
	if dateCol < 0 || descCol < 0 || !hasAmounts {
		return rows
	}
	for _, cells := range records[1:] {
		date := normDate(cell(cells, dateCol))
		desc := cell(cells, descCol)
		if date == "" || desc == "" {
			continue
		}
		var amount float64
		ok := false
		if amountCol >= 0 {
			amount, ok = parseAmount(cell(cells, amountCol))
		} else if d, dok := parseAmount(cell(cells, debitCol)); dok {
			amount, ok = -math.Abs(d), true
		} else if c, cok := parseAmount(cell(cells, creditCol)); cok {
			amount, ok = math.Abs(c), true
		}
		if !ok {
			continue
		}
		out := amount < 0
		if cardConvention {
			out = amount > 0
		}
		r := ParsedRow{Date: date, Description: desc, AmountCents: cents(amount), Direction: "in"}
		if out {
			r.Direction = "out"
		}
		if raw := cell(cells, categoryCol); raw != "" {
			r.SourceCategory = strings.TrimSpace(strings.SplitN(raw, "-", 2)[0])
		}
		rows = append(rows, r)
	}
	return rows
}

var categoryRules = []struct {
	re  *regexp.Regexp
	cat string
}{
	{regexp.MustCompile(`(?i)facebook|meta ads|google ads|tiktok ads|advertis|\bads\b`), "Advertising"},
	{regexp.MustCompile(`(?i)aws|amazon web|vercel|supabase|cloudflare|digitalocean|render\.com|namecheap|domain|godaddy|hosting`), "Infrastructure"},
	{regexp.MustCompile(`(?i)upwork|fiverr|contractor|payroll|gusto|deel|wise|freelanc`), "Contractors"},
	{regexp.MustCompile(`(?i)attio|hubspot|salesforce|\bcrm\b`), "CRM & Revenue"},
	{regexp.MustCompile(`(?i)openai|anthropic|claude|cursor|figma|notion|github|slack|zoom|adobe|elevenlabs|higgsfield|software|saas|subscription`), "Software"},
}

// Categorize buckets a row: keyword rules, then the export's own category,
// then Uncategorized. Inbound rows are Income.
func Categorize(r ParsedRow) string {
	if r.Direction == "in" {
		return "Income"
	}
	for _, rule := range categoryRules {
		if rule.re.MatchString(r.Description) {
			return rule.cat
		}
	}
	if r.SourceCategory != "" {
		return r.SourceCategory
	}
	return "Uncategorized"
}

var (
	reStatementDate = regexp.MustCompile(`(?i)(?:closing date|statement date|statement closing date|billing period)[:\s]*(?:\d{1,2}/\d{1,2}/\d{2,4}\s*(?:-|to|through)\s*)?(\d{1,2})/(\d{1,2})/(\d{2,4})`)
	reTxnLine       = regexp.MustCompile(`^\s*(\d{1,2})/(\d{1,2})(?:/(\d{2,4}))?\*?\s+(.+?)\s+(-?)\$?(-?)([\d,]+\.\d{2})\s*(CR)?\s*$`)
	reLeadingDate   = regexp.MustCompile(`^\d{1,2}/\d{1,2}(?:/\d{2,4})?\*?\s+`)
	reMultiSpace    = regexp.MustCompile(`\s{2,}`)
	reTotal         = regexp.MustCompile(`(?i)^total\b`)
	reNewline       = regexp.MustCompile(`\r?\n`)
)

// ParseCardStatementText reads a card statement that arrived as PDF text:
// only lines that start with a transaction date count, and an MM/DD line with
// no statement year is refused rather than guessed.
func ParseCardStatementText(text string) []ParsedRow {
	stmtYear := ""
	if m := reStatementDate.FindStringSubmatch(text); m != nil {
		stmtYear = year4(m[3])
	}
	rows := []ParsedRow{}
	for _, line := range reNewline.Split(text, -1) {
		m := reTxnLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		mm, dd, yy, rawDesc, dash, innerDash, amount, cr := m[1], m[2], m[3], m[4], m[5], m[6], m[7], m[8]
		year := stmtYear
		if yy != "" {
			year = year4(yy)
		}
		if year == "" {
			continue
		}
		desc := strings.TrimSpace(reMultiSpace.ReplaceAllString(reLeadingDate.ReplaceAllString(rawDesc, ""), " "))
		if desc == "" || reTotal.MatchString(desc) {
			continue
		}
		n, err := strconv.ParseFloat(strings.ReplaceAll(amount, ",", ""), 64)
		if err != nil {
			continue
		}
		c := int64(math.Floor(n*100 + 0.5))
		if c == 0 {
			continue
		}
		dir := "out"
		if dash == "-" || innerDash == "-" || cr == "CR" {
			dir = "in"
		}
		rows = append(rows, ParsedRow{Date: year + "-" + pad2(mm) + "-" + pad2(dd), Description: desc, AmountCents: c, Direction: dir})
	}
	return rows
}

// ParseAnyStatement is the upload route's order: CSV first, then statement
// text. Whichever finds rows wins; neither guesses.
func ParseAnyStatement(text string) []ParsedRow {
	if rows := ParseStatementCSV(text); len(rows) > 0 {
		return rows
	}
	return ParseCardStatementText(text)
}

// ---- bank statement summaries (lib/bank-statements.ts) --------------------

// BankSummary is one statement's per-period totals, not its transactions.
type BankSummary struct {
	Account      string `json:"account"`
	Business     string `json:"business"`
	Month        string `json:"month"`
	CreditsCents int64  `json:"creditsCents"`
	DebitsCents  int64  `json:"debitsCents"`
	NetCents     int64  `json:"netCents"`
}

type MonthPoint struct {
	Month        string `json:"month"`
	CreditsCents int64  `json:"creditsCents"`
	DebitsCents  int64  `json:"debitsCents"`
	NetCents     int64  `json:"netCents"`
}

type BusinessMonths struct {
	Business string       `json:"business"`
	Months   []MonthPoint `json:"months"`
}

var (
	reAccountEnding = regexp.MustCompile(`(?i)Account Ending:\s*\*?(\d{3,4})`)
	reAccountName   = regexp.MustCompile(`(?i)Account Name:\s*([^\n]+?)(?:\s{2,}|\n|$)`)
	reBankDate      = regexp.MustCompile(`(?i)Statement Date:\s*(\d{1,2})/(\d{1,2})/(\d{4})`)
	reCredits       = regexp.MustCompile(`(?i)Total Credits This Period`)
	reDebits        = regexp.MustCompile(`(?i)Total Debits This Period`)
	reDollar        = regexp.MustCompile(`(-?)\$?([\d,]+\.\d{2})`)
)

// amountAfter is the first $-amount after a label (pdftotext often puts the
// value on the next line).
func amountAfter(text string, label *regexp.Regexp) (float64, bool) {
	loc := label.FindStringIndex(text)
	if loc == nil {
		return 0, false
	}
	m := reDollar.FindStringSubmatch(text[loc[0]:])
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", ""), 64)
	if err != nil {
		return 0, false
	}
	if m[1] == "-" {
		n = -n
	}
	return n, true
}

// ParseBankStatementSummary reads one statement's text, or nil if it isn't one.
func ParseBankStatementSummary(text string) *BankSummary {
	acct := reAccountEnding.FindStringSubmatch(text)
	name := reAccountName.FindStringSubmatch(text)
	dm := reBankDate.FindStringSubmatch(text)
	credits, cok := amountAfter(text, reCredits)
	debits, dok := amountAfter(text, reDebits)
	if acct == nil || name == nil || strings.TrimSpace(name[1]) == "" || dm == nil || !cok || !dok {
		return nil
	}
	c := int64(math.Floor(credits*100 + 0.5))
	d := int64(math.Abs(math.Floor(debits*100 + 0.5)))
	return &BankSummary{Account: acct[1], Business: strings.TrimSpace(name[1]), Month: dm[3] + "-" + pad2(dm[1]), CreditsCents: c, DebitsCents: d, NetCents: c - d}
}

// BusinessSeries groups summaries per business, months ascending, a repeated
// month keeping the latest.
func BusinessSeries(summaries []BankSummary) []BusinessMonths {
	byBiz := map[string]map[string]BankSummary{}
	for _, s := range summaries {
		if byBiz[s.Business] == nil {
			byBiz[s.Business] = map[string]BankSummary{}
		}
		byBiz[s.Business][s.Month] = s
	}
	out := []BusinessMonths{}
	for biz, months := range byBiz {
		b := BusinessMonths{Business: biz, Months: []MonthPoint{}}
		for _, s := range months {
			b.Months = append(b.Months, MonthPoint{s.Month, s.CreditsCents, s.DebitsCents, s.NetCents})
		}
		sort.Slice(b.Months, func(i, j int) bool { return b.Months[i].Month < b.Months[j].Month })
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Business < out[j].Business })
	return out
}
