package httpapi

import "net/http"

// maxBodyBytes caps request bodies; a booking request is a few hundred bytes.
const maxBodyBytes = 1 << 16

// limitBody caps every request body at maxBodyBytes, so no handler can be made to read an unbounded
// body into memory. Reading past the cap fails with *http.MaxBytesError.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
