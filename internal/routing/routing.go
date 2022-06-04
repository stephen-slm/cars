package routing

import (
	"compile-and-run-sandbox/internal/sandbox"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"strings"
)

type CompileRequest struct {
	Language   string   `json:"language"`
	SourceCode []string `json:"source_code"`

	StdinData          []string `json:"stdin_data"`
	ExpectedStdoutData []string `json:"expected_stdout_data"`
}

type DirectCompileResponse struct {
	Output []string `json:"output"`

	Status     sandbox.ContainerStatus     `json:"status"`
	TestStatus sandbox.ContainerTestStatus `json:"test_status"`

	RuntimeMs     int64 `json:"runtime_ms"`
	CompileTimeMs int64 `json:"compile_time_ms"`
}

type QueueCompileResponse struct {
	ID string `json:"id"`
}

func handleDecodeError(w http.ResponseWriter, err error) {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxError):
		msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
		http.Error(w, msg, http.StatusBadRequest)

	case errors.Is(err, io.ErrUnexpectedEOF):
		msg := fmt.Sprintf("Request body contains badly-formed JSON")
		http.Error(w, msg, http.StatusBadRequest)

	case errors.As(err, &unmarshalTypeError):
		msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
		http.Error(w, msg, http.StatusBadRequest)

	case strings.HasPrefix(err.Error(), "json: unknown field "):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
		http.Error(w, msg, http.StatusBadRequest)

	case errors.Is(err, io.EOF):
		msg := "Request body must not be empty"
		http.Error(w, msg, http.StatusBadRequest)

	// Catch the error caused by the request body being too large. Again
	// there is an open issue regarding turning this into a sentinel
	// error at https://github.com/golang/go/issues/30715.
	case err.Error() == "http: request body too large":
		msg := "Request body must not be larger than 2MB"
		http.Error(w, msg, http.StatusRequestEntityTooLarge)

	// Otherwise default to logging the error and sending a 500 Internal
	// Server Error response.
	default:
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func DirectCompileHandler(w http.ResponseWriter, r *http.Request) {
	// Use http.MaxBytesReader to enforce a maximum read of 2MB from the response
	// body. A request body larger than that will now result in Decode() returning
	// a "http: request body too large" error.
	r.Body = http.MaxBytesReader(w, r.Body, 1_048576*2)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var direct DirectCompileResponse
	if err := dec.Decode(&direct); err != nil {
		handleDecodeError(w, err)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "%+v", direct)
}
