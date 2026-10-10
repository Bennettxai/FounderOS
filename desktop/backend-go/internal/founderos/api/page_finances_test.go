package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/pages/finances"
)

// Ports of tests/statements-route.test.ts and tests/bank-statement-route.test.ts
// against the bridge paths, plus GET /pages/finances.

type financesFakeStore struct {
	ledger map[string]finances.LedgerRow
	bank   []finances.BankSummary
}

func (s *financesFakeStore) InsertLedgerRows(_ context.Context, rows []finances.LedgerRow) (int, error) {
	n := 0
	for _, r := range rows {
		r.Card = finances.NormalizeCardID(r.Card)
		if _, ok := s.ledger[finances.LedgerHash(r)]; !ok {
			s.ledger[finances.LedgerHash(r)] = r
			n++
		}
	}
	return n, nil
}

func (s *financesFakeStore) LedgerRows(context.Context) ([]finances.SpendRow, error) {
	out := []finances.SpendRow{}
	for _, r := range s.ledger {
		out = append(out, finances.SpendRow{Date: r.Date, Description: r.Description, AmountCents: r.AmountCents, Direction: r.Direction, Category: r.Category, Card: r.Card})
	}
	finances.SortSpendRows(out)
	return out, nil
}

func (s *financesFakeStore) UpsertBankSummary(_ context.Context, b finances.BankSummary) error {
	s.bank = append(s.bank, b)
	return nil
}

func (s *financesFakeStore) BankSummaries(context.Context) ([]finances.BankSummary, error) {
	return s.bank, nil
}

type financesFakePDF struct {
	text  string
	err   error
	calls int
}

func (p *financesFakePDF) Text(context.Context, []byte) (string, error) {
	p.calls++
	return p.text, p.err
}

type financesHarness struct {
	store *financesFakeStore
	pdf   *financesFakePDF
	srv   http.Handler
}

func financesNewHarness(t *testing.T) *financesHarness {
	t.Helper()
	h := &financesHarness{store: &financesFakeStore{ledger: map[string]finances.LedgerRow{}}, pdf: &financesFakePDF{}}
	prev := financesDepsFor
	financesDepsFor = func(*Deps) financesDeps {
		return financesDeps{
			store: h.store,
			pdf:   h.pdf,
			now:   func() time.Time { return time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC) },
			income: func(context.Context, paykit.HistoryPort) payments.FinancesIncome {
				return payments.FinancesIncome{Accounts: payments.IncomeAccounts(false, nil, nil, nil)}
			},
		}
	}
	t.Cleanup(func() { financesDepsFor = prev })
	h.srv = router(t, &Deps{})
	return h
}

func (h *financesHarness) do(t *testing.T, method, path string, body *bytes.Buffer, ctype string) (int, map[string]any) {
	t.Helper()
	if body == nil {
		body = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Cookie", "session=ok")
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	w := httptest.NewRecorder()
	h.srv.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func financesForm(t *testing.T, fields map[string]string, fileName, fileType, content string) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if fileName != "" {
		hdr := map[string][]string{"Content-Disposition": {`form-data; name="file"; filename="` + fileName + `"`}, "Content-Type": {fileType}}
		fw, err := mw.CreatePart(hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write([]byte(content))
	}
	_ = mw.Close()
	return &b, mw.FormDataContentType()
}

const financesCSV = "Date,Description,Amount\n07/15/2026,AMAZON WEB SERVICES,-57.00"

const financesCardText = `
The Platinum Card
Closing Date 07/26/26
07/01/26   ANTHROPIC*CLAUDE      SAN FRANCISCO CA     $200.00
07/04/26   AWS                   SEATTLE WA            $57.00
`

func TestFinancesRoutesNeedASession(t *testing.T) {
	h := financesNewHarness(t)
	for p, method := range map[string]string{"/api/founderos/pages/finances": http.MethodGet, "/api/founderos/pages/finances/statements": http.MethodPost, "/api/founderos/pages/finances/bank-statement": http.MethodPost} {
		w := httptest.NewRecorder()
		h.srv.ServeHTTP(w, httptest.NewRequest(method, p, strings.NewReader(financesCSV)))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s without a session: %d", p, w.Code)
		}
	}
}

func TestFinancesStatementsFilesACSVUnderItsCard(t *testing.T) {
	h := financesNewHarness(t)
	body, ctype := financesForm(t, map[string]string{"card": "blue"}, "vantage-july.csv", "text/csv", financesCSV)
	code, out := h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements", body, ctype)
	if code != 200 || out["inserted"].(float64) != 1 || out["card"] != "blue" {
		t.Fatalf("%d %v", code, out)
	}
	for _, r := range h.store.ledger {
		if r.Card != "blue" {
			t.Fatalf("row filed under %s", r.Card)
		}
	}
	if h.pdf.calls != 0 {
		t.Fatal("a CSV never goes through pdftotext")
	}
}

func TestFinancesStatementsDefaultsToPlatinumAndTakesTheQueryCard(t *testing.T) {
	h := financesNewHarness(t)
	code, out := h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements", bytes.NewBufferString(financesCSV), "text/csv")
	if code != 200 || out["card"] != "platinum" {
		t.Fatalf("%d %v", code, out)
	}
	code, out = h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements?card=gold", bytes.NewBufferString(financesCardText), "text/plain")
	if code != 200 || out["inserted"].(float64) != 2 || out["parsed"].(float64) != 2 || out["card"] != "gold" {
		t.Fatalf("%d %v", code, out)
	}
	code, out = h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements?card=gold", bytes.NewBufferString(financesCardText), "text/plain")
	if code != 200 || out["inserted"].(float64) != 0 {
		t.Fatalf("re-upload: %d %v", code, out)
	}
	if code, _ = h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements", bytes.NewBufferString("just some notes"), "text/plain"); code != 400 {
		t.Fatalf("no rows: %d", code)
	}
}

func TestFinancesStatementsReadsAPDFThroughTheRunner(t *testing.T) {
	h := financesNewHarness(t)
	h.pdf.text = financesCardText
	body, ctype := financesForm(t, map[string]string{"card": "platinum"}, "july.pdf", "application/pdf", "%PDF-1.4")
	code, out := h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements", body, ctype)
	if code != 200 || out["inserted"].(float64) != 2 || h.pdf.calls != 1 {
		t.Fatalf("%d %v calls=%d", code, out, h.pdf.calls)
	}
	months, _ := out["uploadedMonths"].([]any)
	if len(months) != 1 || months[0] != "2026-07" {
		t.Fatalf("uploadedMonths %v", out["uploadedMonths"])
	}
	h.pdf.err = finances.ErrNoPdftotext
	code, out = h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements", bytes.NewBufferString("%PDF"), "application/pdf")
	if code != 400 || !strings.Contains(out["error"].(string), "pdftotext") {
		t.Fatalf("no poppler: %d %v", code, out)
	}
}

const financesBankText = `
  Account Name: Vantage LLC
  Account Ending: *5630
  Statement Date: 04/30/2026
  Total Credits This Period      $12,500.00
  Total Debits This Period       $4,250.00
`

func TestFinancesBankStatementTakesExtractedText(t *testing.T) {
	h := financesNewHarness(t)
	code, out := h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", bytes.NewBufferString(financesBankText), "text/plain")
	s, _ := out["summary"].(map[string]any)
	if code != 200 || s["business"] != "Vantage LLC" || s["month"] != "2026-04" || s["creditsCents"].(float64) != 1250000 {
		t.Fatalf("%d %v", code, out)
	}
	if len(h.store.bank) != 1 || h.store.bank[0].Account != "5630" {
		t.Fatalf("stored %+v", h.store.bank)
	}
	if code, _ = h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", bytes.NewBufferString("just some notes, not a statement"), "text/plain"); code != 400 {
		t.Fatalf("not a statement: %d", code)
	}
	if code, _ = h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", bytes.NewBufferString("   "), "text/plain"); code != 400 {
		t.Fatalf("empty: %d", code)
	}
}

func TestFinancesBankStatementPDFUpload(t *testing.T) {
	h := financesNewHarness(t)
	h.pdf.text = financesBankText
	body, ctype := financesForm(t, nil, "april.pdf", "application/pdf", "%PDF-1.4")
	if code, out := h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", body, ctype); code != 200 || h.pdf.calls != 1 {
		t.Fatalf("%d %v", code, out)
	}
	if code, _ := h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", nil, "application/pdf"); code != 400 {
		t.Fatalf("empty upload: %d", code)
	}
	h.pdf.err = errors.New("pdftotext: exit 1")
	body, ctype = financesForm(t, nil, "april.pdf", "application/pdf", "%PDF-1.4")
	if code, _ := h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", body, ctype); code != 500 {
		t.Fatalf("extraction failure: %d", code)
	}
}

func TestFinancesPagePayload(t *testing.T) {
	h := financesNewHarness(t)
	h.do(t, http.MethodPost, "/api/founderos/pages/finances/statements", bytes.NewBufferString(financesCardText), "text/plain")
	code, out := h.do(t, http.MethodGet, "/api/founderos/pages/finances", nil, "")
	if code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	if out["thisMonth"] != "2026-07" || out["expensesLive"] != true || out["statementIsThisMonth"] != true || out["expenses"].(float64) != 257 {
		t.Fatalf("payload %v", out)
	}
	vol, _ := out["volume"].(map[string]any)
	if vol == nil || vol["caption"] != "income this month · 0/3 processors live" {
		t.Fatalf("volume %v", out["volume"])
	}
	led, _ := out["ledger"].(map[string]any)
	if rows, _ := led["rows"].([]any); len(rows) != 2 {
		t.Fatalf("ledger %v", led)
	}
}

// A statement over the upload cap is refused whole (413), never ingested from
// the part that fit.
func TestFinancesBankStatementOverTheCapIsRefusedNotTruncated(t *testing.T) {
	h := financesNewHarness(t)
	big := financesBankText + strings.Repeat("padding line\n", (financesMaxUpload/13)+10)
	for _, mt := range []string{"text/plain", "application/pdf"} {
		code, out := h.do(t, http.MethodPost, "/api/founderos/pages/finances/bank-statement", bytes.NewBufferString(big), mt)
		if code != http.StatusRequestEntityTooLarge {
			t.Fatalf("%s: %d %v", mt, code, out)
		}
	}
	if len(h.store.bank) != 0 {
		t.Fatalf("a truncated statement was stored: %+v", h.store.bank)
	}
}
