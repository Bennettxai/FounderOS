package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
	"github.com/rhl/businessos-backend/internal/founderos/pages/personas"
)

// catalogGet is a session-authed GET against the FounderOS router.
func catalogGet(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Cookie", "session=ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPersonasPageServesTheFounderOSPersonas(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ws := catalogdb.Workspace(t, pool, "founderos")
	if _, err := pool.Exec(context.Background(), `INSERT INTO founderos_personas
		(id, workspace_id, ord, name, archetype, tagline, summary, accent, north_star, pillars, connectors, metrics, brain_use, signature_play)
		VALUES ('persona-a', $1, 1, 'Agency Owner', 'Agency Operator', 't', 's', '#3df08c', 'n',
		'[{"name":"Sales","focus":"f","agents":["Closer"]}]', '["stripe"]', '["mrr"]', 'b', 'p')`, ws); err != nil {
		t.Fatal(err)
	}
	r := router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/personas", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	w = catalogGet(t, r, "/api/founderos/pages/personas")
	var body struct {
		Personas []personas.Persona `json:"personas"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if len(body.Personas) != 1 || body.Personas[0].Name != "Agency Owner" || body.Personas[0].NorthStar != "n" {
		t.Fatalf("personas = %+v", body.Personas)
	}
}

func TestPersonasPageIsHonestWhenTheDataIsMissing(t *testing.T) {
	// No Postgres at all: 503, never an empty list.
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	if w := catalogGet(t, r, "/api/founderos/pages/personas"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no pool: %d %s", w.Code, w.Body)
	}
	// Postgres but no founderos workspace: 503 too.
	pool := catalogdb.TestDB(t)
	r = router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})
	if w := catalogGet(t, r, "/api/founderos/pages/personas"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no workspace: %d %s", w.Code, w.Body)
	}
}
