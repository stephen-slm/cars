package routing

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
)

type CompileRequest struct {
	ID         string   `json:"id"`
	Language   string   `json:"language"`
	SourceCode []string `json:"source_code"`

	StdinData          []string `json:"stdin_data"`
	ExpectedStdoutData []string `json:"expected_stdout_data"`
}

type QueueCompileResponse struct {
	ID string `json:"id"`
}

type CompileErrorResponse struct {
	Message string `json:"id"`
	Code    int    `json:"code"`
}

func handleJsonResponse(w http.ResponseWriter, body any, code int) {
	response, _ := json.Marshal(body)

	fmt.Println(string(response))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
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

type CompilerHandler struct {
	Publisher *nsq.Producer
}

func (h CompilerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use http.MaxBytesReader to enforce a maximum read of 2MB.
	r.Body = http.MaxBytesReader(w, r.Body, 1_048576*2)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var direct CompileRequest
	direct.ID = uuid.New().String()

	if err := dec.Decode(&direct); err != nil {
		handleDecodeError(w, err)
		return
	}

	bytes, _ := json.Marshal(direct)
	err := h.Publisher.Publish("containers", bytes)

	if err != nil {
		log.Print(err.Error())

		handleJsonResponse(w, CompileErrorResponse{
			Message: "failed to execute compile request",
			Code:    0,
		}, http.StatusInternalServerError)

		return
	}

	handleJsonResponse(w, QueueCompileResponse{ID: direct.ID}, http.StatusOK)
}
