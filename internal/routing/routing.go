package routing

import (
	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/queue"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strings"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/repository"
	"compile-and-run-sandbox/internal/sandbox"
	"compile-and-run-sandbox/internal/validation"
)

type CompileRequest struct {
	Language   string `json:"language" validate:"required,oneof=python node"`
	SourceCode string `json:"source_code" validate:"required"`

	StdinData          []string `json:"stdin_data" validate:"required"`
	ExpectedStdoutData []string `json:"expected_stdout_data" validate:"required"`
}

type CompileInfoResponse struct {
	Status     string `json:"status"`
	TestStatus string `json:"test_status"`

	CompileMs int64 `json:"compile_ms"`
	RuntimeMs int64 `json:"runtime_ms"`

	Output string `json:"output"`
}

type QueueCompileResponse struct {
	ID string `json:"id"`
}

type CompileErrorResponse struct {
	Errors []string `json:"errors"`
	Code   int      `json:"code"`
}

func handleJSONResponse(w http.ResponseWriter, body any, code int) {
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
		msg := "Request body contains badly-formed JSON"
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
		msg := "Request body must not be larger than 1MB"
		http.Error(w, msg, http.StatusRequestEntityTooLarge)

	// Otherwise default to logging the error and sending a 500 Internal
	// Server Error response.
	default:
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type CompilerHandler struct {
	FileHandler files.Files
	Db          repository.Repository
	Translator  ut.Translator
	Validator   *validator.Validate
	Publisher   *nsq.Producer
}

func (h CompilerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use http.MaxBytesReader to enforce a maximum read of 1MB.
	r.Body = http.MaxBytesReader(w, r.Body, 1_048576*1)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var direct CompileRequest

	if err := dec.Decode(&direct); err != nil {
		handleDecodeError(w, err)
		return
	}

	if err := h.Validator.Struct(&direct); err != nil {
		log.Error().Err(err)

		handleJSONResponse(w, CompileErrorResponse{
			Errors: validation.TranslateError(err, h.Translator),
			Code:   0,
		}, http.StatusBadRequest)
	}

	requestId := uuid.NewString()
	_ = h.FileHandler.WriteFile(requestId, "source", []byte(direct.SourceCode))

	bytes, _ := json.Marshal(queue.CompileMessage{
		ID:                 requestId,
		Language:           direct.Language,
		StdinData:          direct.StdinData,
		ExpectedStdoutData: direct.ExpectedStdoutData,
	})

	err := h.Publisher.Publish("containers", bytes)

	if err != nil {
		log.Error().Err(err)

		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"failed to execute compile request"},
			Code:   0,
		}, http.StatusInternalServerError)

		return
	}

	dbErr := h.Db.InsertExecution(&repository.Execution{
		ID:         requestId,
		Status:     sandbox.NotRan.String(),
		TestStatus: sandbox.TestNotRan.String(),
	})

	if dbErr != nil {
		log.Error().Err(dbErr)

		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"failed to create execution record"},
			Code:   0,
		}, http.StatusInternalServerError)

		return
	}

	handleJSONResponse(w, QueueCompileResponse{ID: requestId}, http.StatusOK)
}

type CompilerInfoHandler struct {
	FileHandler files.Files
	Repo        repository.Repository
}

func (h CompilerInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uuidValue, ok := mux.Vars(r)["id"]

	if !ok {
		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"no or invalid compile request id provided."},
			Code:   0,
		}, http.StatusBadRequest)

		return
	}

	parsedIdValue, err := uuid.Parse(uuidValue)

	if err != nil {
		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"failed to parse id value"},
			Code:   0,
		}, http.StatusBadRequest)

		return
	}

	execution, err := h.Repo.GetExecution(parsedIdValue.String())

	if errors.Is(err, gorm.ErrRecordNotFound) {
		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"the execution does not exist by the provided id."},
			Code:   0,
		}, http.StatusNotFound)

		return
	}

	resp := CompileInfoResponse{
		Status:     execution.Status,
		TestStatus: execution.TestStatus,
		CompileMs:  execution.CompileMs,
		RuntimeMs:  execution.RuntimeMs,
		Output:     "",
	}

	if data, outputErr := h.FileHandler.GetFile(parsedIdValue.String(), "output"); outputErr == nil {
		log.Info().Str("data", string(data)).Msg("data")
		resp.Output = string(data)
	}

	handleJSONResponse(w, resp, http.StatusOK)
}
