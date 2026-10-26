// Package api holds the OpenAPI spec, embedded so the server can serve the exact spec it was built from.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
