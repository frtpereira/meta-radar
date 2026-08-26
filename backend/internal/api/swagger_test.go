package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwaggerHandlerServesIndexAndSpec(t *testing.T) {
	h := swaggerHandler([]byte(`{"openapi":"3.0.0"}`))

	indexReq := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	indexReq.RequestURI = indexReq.URL.RequestURI()
	indexRR := httptest.NewRecorder()
	h(indexRR, indexReq)
	assert.Equal(t, http.StatusOK, indexRR.Code)
	assert.Contains(t, indexRR.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, indexRR.Body.String(), `url: "doc.json"`)

	docReq := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	docReq.RequestURI = docReq.URL.RequestURI()
	docRR := httptest.NewRecorder()
	h(docRR, docReq)
	assert.Equal(t, http.StatusOK, docRR.Code)
	assert.Contains(t, docRR.Header().Get("Content-Type"), "application/json")
	assert.JSONEq(t, `{"openapi":"3.0.0"}`, docRR.Body.String())
}

func TestSwaggerHandlerRedirectsUsingParsedPrefix(t *testing.T) {
	h := swaggerHandler([]byte(`{"openapi":"3.0.0"}`))

	matches := swaggerPathRegexp.FindStringSubmatch("/nested/swagger/doc.json")
	require.Len(t, matches, 3)
	assert.Equal(t, "/nested/swagger/", matches[1])
	assert.Equal(t, "doc.json", matches[2])

	req := httptest.NewRequest(http.MethodGet, "/nested/swagger/", nil)
	req.RequestURI = req.URL.RequestURI()
	rr := httptest.NewRecorder()
	h(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "/nested/swagger/index.html", rr.Header().Get("Location"))
}

func TestSwaggerHandlerMethodAndPassthroughBehavior(t *testing.T) {
	h := swaggerHandler([]byte(`{"openapi":"3.0.0"}`))

	postReq := httptest.NewRequest(http.MethodPost, "/swagger/doc.json", nil)
	postReq.RequestURI = postReq.URL.RequestURI()
	postRR := httptest.NewRecorder()
	h(postRR, postReq)
	assert.Equal(t, http.StatusMethodNotAllowed, postRR.Code)
	assert.Contains(t, postRR.Body.String(), "Method not allowed")

	assetReq := httptest.NewRequest(http.MethodGet, "/swagger/swagger-ui.css", nil)
	assetReq.RequestURI = assetReq.URL.RequestURI()
	assetRR := httptest.NewRecorder()
	h(assetRR, assetReq)
	assert.Equal(t, http.StatusOK, assetRR.Code)
	assert.Contains(t, assetRR.Header().Get("Content-Type"), "text/css")
	require.NotEmpty(t, assetRR.Body.String())

	missingReq := httptest.NewRequest(http.MethodGet, "/swagger/does-not-exist.js", nil)
	missingReq.RequestURI = missingReq.URL.RequestURI()
	missingRR := httptest.NewRecorder()
	h(missingRR, missingReq)
	assert.Equal(t, http.StatusNotFound, missingRR.Code)
	assert.Contains(t, missingRR.Header().Get("Content-Type"), "application/javascript")
}
