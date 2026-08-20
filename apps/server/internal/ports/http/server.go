// Package http exposes the API. Handler signatures are generated from the
// OpenAPI document (backend ADR 0002); nothing here is hand-rolled.
package http

import "net/http"

// NewMux returns the server's routes. It is empty until the OpenAPI document
// has something to generate from.
func NewMux() *http.ServeMux {
	return http.NewServeMux()
}
