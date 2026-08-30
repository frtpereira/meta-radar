package api

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/frtpereira/meta-radar/docs"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouterRoutesAndMiddleware(t *testing.T) {
	router := NewRouter(nil, nil, "")
	routes, ok := router.(chi.Routes)
	require.True(t, ok)

	var got []string
	require.NoError(t, chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		got = append(got, method+" "+route)
		return nil
	}))
	sort.Strings(got)

	assert.Subset(t, got, []string{
		"GET /health",
		"GET /swagger/*",
		"GET /api/tournaments",
		"GET /api/tournaments/{id}",
		"GET /api/metas",
		"GET /api/archetypes/stats",
		"GET /api/matchups/stats",
		"GET /api/archetypes/{id}",
		"GET /api/archetypes/{id}/variants",
		"GET /api/archetypes/{id}/card-stats",
		"GET /api/players/{nickname}",
		"GET /api/decklists/{id}",
		"POST /api/webhooks/limitless",
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	preflight := httptest.NewRequest(http.MethodOptions, "/api/tournaments", nil)
	preflight.Header.Set("Origin", "http://localhost:3000")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "Content-Type")
	preflightRR := httptest.NewRecorder()
	router.ServeHTTP(preflightRR, preflight)

	assert.Equal(t, "http://localhost:3000", preflightRR.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, preflightRR.Header().Get("Access-Control-Allow-Methods"), http.MethodGet)

	swaggerReq := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	swaggerRR := httptest.NewRecorder()
	router.ServeHTTP(swaggerRR, swaggerReq)
	assert.Equal(t, http.StatusOK, swaggerRR.Code)
	assert.JSONEq(t, string(docs.SwaggerJSON), swaggerRR.Body.String())
}
