package finances

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// PDFText extracts a PDF's text. The upload routes take it as a dependency so
// tests never exec a binary.
type PDFText interface {
	Text(ctx context.Context, pdf []byte) (string, error)
}

// ErrNoPdftotext: the host has no poppler. Callers point at the text/plain
// upload path, which is how statements land on the mini.
var ErrNoPdftotext = errors.New("pdftotext not installed (brew install poppler), or upload the extracted text as text/plain")

// ExecPDFText runs the local `pdftotext -layout - -` (lib/pdf-text.ts),
// trying PATH and then the Homebrew / usr-local locations.
type ExecPDFText struct {
	Candidates []string
	Timeout    time.Duration
}

var DefaultPDFCandidates = []string{"pdftotext", "/opt/homebrew/bin/pdftotext", "/usr/local/bin/pdftotext"}

func (e ExecPDFText) Text(ctx context.Context, pdf []byte) (string, error) {
	cands := e.Candidates
	if cands == nil {
		cands = DefaultPDFCandidates
	}
	bin := ""
	for _, c := range cands {
		if p, err := exec.LookPath(c); err == nil {
			bin = p
			break
		}
	}
	if bin == "" {
		return "", ErrNoPdftotext
	}
	timeout := e.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %v %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out.String(), nil
}
