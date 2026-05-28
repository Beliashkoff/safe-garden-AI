// Package docs serves the OpenAPI specification and a Swagger UI page. The spec
// in this directory is the single source of truth for the HTTP contract; the
// mobile client can generate its API client from it. Served at /v1/docs.
package docs

import (
	_ "embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger_ui.html
var swaggerUIHTML []byte

// Mount registers the docs routes on r:
//   - GET /docs              → Swagger UI page
//   - GET /docs/openapi.yaml → the raw spec
func Mount(r chi.Router) {
	r.Get("/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(swaggerUIHTML)
	})
	r.Get("/docs/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(openapiSpec)
	})
}
