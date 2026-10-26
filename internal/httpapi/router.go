// Package httpapi wires the API handlers into an HTTP router, with the docs routes, the request body
// limit and the JSON error responses for requests the generated code rejects.
package httpapi

import (
	"net/http"
	"time"

	"github.com/samluu/waracle-task/internal/httpapi/handlers"
)

func NewRouter(s handlers.Store) http.Handler {
	return newRouter(handlers.NewServer(s, time.Now))
}

func newRouter(srv *handlers.Server) http.Handler {
	mux := http.NewServeMux()
	// Docs routes sit outside the spec: they describe the API rather than being part of it.
	mux.HandleFunc("GET /openapi.yaml", serveSpec)
	mux.HandleFunc("GET /docs", serveDocs)

	// The generated strict wrapper decodes requests and encodes responses around srv.
	strict := handlers.NewStrictHandlerWithOptions(srv, nil, handlers.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  requestError,
		ResponseErrorHandlerFunc: responseError,
	})
	return limitBody(handlers.HandlerWithOptions(strict, handlers.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: paramError,
	}))
}
