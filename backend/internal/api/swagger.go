package api

import (
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"

	swaggerFiles "github.com/swaggo/files"
)

var swaggerIndexTemplate = template.Must(template.New("swagger_index.html").Parse(swaggerIndexHTML))

var swaggerPathRegexp = regexp.MustCompile(`^(.*/)([^?].*)?[?|.]*$`)

// swaggerHandler serves the Swagger UI (assets bundled by swaggo/files) plus
// the generated OpenAPI document at doc.json, without depending on the
// heavier swaggo/swag package at runtime.
func swaggerHandler(specJSON []byte) http.HandlerFunc {
	var once sync.Once
	handler := swaggerFiles.Handler

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		matches := swaggerPathRegexp.FindStringSubmatch(r.RequestURI)
		prefix, path := matches[1], matches[2]
		once.Do(func() { handler.Prefix = prefix })

		switch filepath.Ext(path) {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}

		switch path {
		case "index.html":
			_ = swaggerIndexTemplate.Execute(w, struct{ URL string }{URL: "doc.json"})
		case "doc.json":
			_, _ = w.Write(specJSON)
		case "":
			http.Redirect(w, r, prefix+"index.html", http.StatusMovedPermanently)
		default:
			handler.ServeHTTP(w, r)
		}
	}
}

const swaggerIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="./swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="./swagger-ui-bundle.js"></script>
<script src="./swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "{{.URL}}",
    dom_id: "#swagger-ui",
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout"
  })
}
</script>
</body>
</html>
`
