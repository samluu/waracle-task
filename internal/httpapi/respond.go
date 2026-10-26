package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

func badRequest(msg string) BadRequestJSONResponse {
	return BadRequestJSONResponse{Error: msg}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Error{Error: msg}); err != nil {
		slog.Error("write response", "err", err)
	}
}

// paramError handles query and path parameters the generated code couldn't parse, e.g. a bad date.
func paramError(w http.ResponseWriter, _ *http.Request, err error) {
	writeError(w, http.StatusBadRequest, err.Error())
}

// requestError handles request bodies the generated strict handler couldn't decode.
func requestError(w http.ResponseWriter, _ *http.Request, err error) {
	if tooLarge := new(http.MaxBytesError); errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body must be at most %d bytes", tooLarge.Limit))
		return
	}
	// The generated code wraps the decoder's error as "can't decode JSON body: ..."; report just the cause.
	if cause := errors.Unwrap(err); cause != nil {
		err = cause
	}
	writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
}

// responseError handles errors handlers return rather than turning into a response: they're unexpected,
// so they're logged and the client gets a 500 without their details.
func responseError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
