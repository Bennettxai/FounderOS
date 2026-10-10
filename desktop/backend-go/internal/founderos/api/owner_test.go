package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

func ownerRouter(t *testing.T, d *Deps, user *middleware.BetterAuthUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(middleware.UserContextKey, user)
		}
		c.Next()
	})
	r.GET("/x", OwnerOnly(d), func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func TestOwnerOnlyAdmitsTheFounderOSOwnerAndNobodyElse(t *testing.T) {
	t.Setenv("FOUNDEROS_OWNER_EMAIL", "")
	pool := throwawayPool(t) // founderos owned by 'u1'
	d := &Deps{Pool: pool}
	cases := []struct {
		user *middleware.BetterAuthUser
		want int
	}{
		{nil, http.StatusUnauthorized},
		{&middleware.BetterAuthUser{ID: "u1", Email: "owner@example.com"}, 200},
		{&middleware.BetterAuthUser{ID: "stranger", Email: "someone@example.com"}, http.StatusForbidden},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		ownerRouter(t, d, tc.user).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != tc.want {
			t.Errorf("user %+v: %d, want %d", tc.user, w.Code, tc.want)
		}
	}
}

func TestOwnerOnlyEmailAllowlistAndFailClosed(t *testing.T) {
	t.Setenv("FOUNDEROS_OWNER_EMAIL", "Alex@Example.com")
	d := &Deps{} // no database
	w := httptest.NewRecorder()
	ownerRouter(t, d, &middleware.BetterAuthUser{ID: "x", Email: "alex@example.com"}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != 200 {
		t.Fatalf("allowlisted email: %d", w.Code)
	}
	w = httptest.NewRecorder()
	ownerRouter(t, d, &middleware.BetterAuthUser{ID: "y", Email: "other@example.com"}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("no database and not allowlisted must fail closed: %d", w.Code)
	}
	_ = context.Background()
}

// A signed-in account that does not own Founder OS gets a body the /os layout
// can turn into a setup panel: setup=true and a human next step.
func TestOwnerOnlyTellsANewAccountHowToSetUp(t *testing.T) {
	t.Setenv("FOUNDEROS_OWNER_EMAIL", "")
	w := httptest.NewRecorder()
	ownerRouter(t, &Deps{}, &middleware.BetterAuthUser{ID: "new", Email: "new@example.com"}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d", w.Code)
	}
	var body struct {
		Setup bool   `json:"setup"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Setup || !strings.Contains(body.Error, "make demo") || strings.Contains(body.Error, "v1") {
		t.Fatalf("body = %s", w.Body.String())
	}
}
