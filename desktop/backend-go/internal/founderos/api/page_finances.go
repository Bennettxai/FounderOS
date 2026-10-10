package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/pages/finances"
)

// /finances (spec 6.15): the page payload plus the two statement uploads
// (route-map /api/finances/statements and /api/finances/bank-statement).
// Uploads write bridge data only (founderos_ledger_rows, founderos_bank_summaries);
// processor reads go through the payments connector, read-only.
func init() { RegisterPage(financesRegister) }

// financesDeps are the page's collaborators; tests swap financesDepsFor.
type financesDeps struct {
	store   finances.Store
	income  finances.IncomeSource
	history paykit.HistoryPort
	pdf     finances.PDFText
	now     func() time.Time
}

var financesDepsFor = func(d *Deps) financesDeps {
	fd := financesDeps{pdf: finances.ExecPDFText{}, now: time.Now}
	if d.Pool != nil {
		st := &finances.PgStore{Pool: d.Pool}
		fd.store = st
		fd.history = &finances.PgPaykitHistory{Store: st, Account: paykit.LaunchpadCohort.ID}
	}
	pay := payments.New(d.Resolver)
	fd.income = pay.FinancesIncome
	return fd
}

// financesMaxUpload bounds a statement upload (PDFs of a year of charges fit).
const financesMaxUpload = 25 << 20

func financesRegister(s *gin.RouterGroup, d *Deps) {
	fd := financesDepsFor(d)
	s.GET("/pages/finances", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()
		c.JSON(http.StatusOK, finances.Build(ctx, finances.Sources{Store: fd.store, Income: fd.income, History: fd.history, Now: fd.now}))
	})
	s.POST("/pages/finances/statements", func(c *gin.Context) { financesStatements(c, fd) })
	s.POST("/pages/finances/bank-statement", func(c *gin.Context) { financesBankStatement(c, fd) })
}

func financesError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// financesIngestStatus maps ingestion errors: a bad statement is the
// caller's (400); a store failure is ours (500), or 503 with no database.
func financesIngestStatus(err error) int {
	switch {
	case errors.Is(err, finances.ErrEmptyStatement), errors.Is(err, finances.ErrNoRows), errors.Is(err, finances.ErrNotABankStatement):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func financesMediaType(c *gin.Context) string {
	mt, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
	return strings.ToLower(mt)
}

// financesUploadFile reads the multipart `file` field: its bytes, whether it
// is a PDF (by name or type), and ok=false when absent.
func financesUploadFile(c *gin.Context) (data []byte, isPDF, ok bool, err error) {
	fh, ferr := c.FormFile("file")
	if ferr != nil {
		return nil, false, false, nil
	}
	f, err := fh.Open()
	if err != nil {
		return nil, false, false, err
	}
	defer f.Close()
	data, err = io.ReadAll(f)
	isPDF = strings.EqualFold(filepath.Ext(fh.Filename), ".pdf") || strings.HasPrefix(fh.Header.Get("Content-Type"), "application/pdf")
	return data, isPDF, true, err
}

// financesStatements is POST /api/finances/statements: a card statement as a
// CSV (multipart or text/csv), a PDF (multipart or application/pdf), or
// already-extracted text (text/plain, the path for a host without poppler),
// filed under the `card` field or ?card= (default Platinum).
func financesStatements(c *gin.Context, fd financesDeps) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, financesMaxUpload)
	ctx := c.Request.Context()
	card := finances.NormalizeCardID(c.Query("card"))
	var text string
	switch mt := financesMediaType(c); {
	case mt == "multipart/form-data":
		if v := c.PostForm("card"); v != "" {
			card = finances.NormalizeCardID(v)
		}
		data, isPDF, ok, err := financesUploadFile(c)
		if err != nil {
			financesError(c, http.StatusBadRequest, "could not read upload: "+err.Error())
			return
		}
		if ok && isPDF {
			if text, err = fd.pdf.Text(ctx, data); err != nil {
				financesError(c, http.StatusBadRequest, err.Error())
				return
			}
		} else if ok {
			text = string(data)
		}
	case mt == "application/pdf":
		data, err := io.ReadAll(c.Request.Body)
		if err == nil {
			text, err = fd.pdf.Text(ctx, data)
		}
		if err != nil {
			financesError(c, http.StatusBadRequest, err.Error())
			return
		}
	default:
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			financesError(c, http.StatusBadRequest, "could not read upload: "+err.Error())
			return
		}
		text = string(data)
	}
	if fd.store == nil {
		if strings.TrimSpace(text) == "" {
			financesError(c, http.StatusBadRequest, finances.ErrEmptyStatement.Error())
			return
		}
		financesError(c, http.StatusServiceUnavailable, "finances store: no database")
		return
	}
	res, err := finances.IngestCardStatement(ctx, fd.store, text, card)
	if err != nil {
		financesError(c, financesIngestStatus(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, res)
}

// financesReadError answers a body that could not be read whole: over the
// cap is 413, anything else 400. A partial body is never ingested.
func financesReadError(c *gin.Context, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		financesError(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("upload is over the %d MB limit", financesMaxUpload>>20))
		return
	}
	financesError(c, http.StatusBadRequest, "could not read upload: "+err.Error())
}

// financesBankStatement is POST /api/finances/bank-statement: a bank
// statement's summary as a PDF (multipart `file` or a raw body) or as
// extracted text (text/plain), upserted by account + month.
func financesBankStatement(c *gin.Context, fd financesDeps) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, financesMaxUpload)
	ctx := c.Request.Context()
	var text string
	if financesMediaType(c) == "text/plain" {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			financesReadError(c, err)
			return
		}
		if strings.TrimSpace(string(data)) == "" {
			financesError(c, http.StatusBadRequest, "expected statement text (text/plain body)")
			return
		}
		text = string(data)
	} else {
		var data []byte
		if financesMediaType(c) == "multipart/form-data" {
			data, _, _, _ = financesUploadFile(c)
		} else {
			var err error
			if data, err = io.ReadAll(c.Request.Body); err != nil {
				financesReadError(c, err)
				return
			}
		}
		if len(data) == 0 {
			financesError(c, http.StatusBadRequest, "expected a PDF upload (file field or PDF body), or extracted text as text/plain")
			return
		}
		var err error
		if text, err = fd.pdf.Text(ctx, data); err != nil {
			financesError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if fd.store == nil {
		financesError(c, http.StatusServiceUnavailable, "finances store: no database")
		return
	}
	s, err := finances.IngestBankStatement(ctx, fd.store, text)
	if err != nil {
		financesError(c, financesIngestStatus(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": s})
}
