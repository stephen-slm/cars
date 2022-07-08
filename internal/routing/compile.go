package routing

import (
	"encoding/json"
	"net/http"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/queue"
	"compile-and-run-sandbox/internal/repository"
	"compile-and-run-sandbox/internal/sandbox"
	"compile-and-run-sandbox/internal/validation"
)

type CompilerHandlers struct {
	FileHandler files.Files
	Repo        repository.Repository
	Translator  ut.Translator
	Validator   *validator.Validate
	Queue       queue.Queue
}

func (h CompilerHandlers) HandleCompileRequest(w http.ResponseWriter, r *http.Request) {
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

	compiler := sandbox.Compilers[direct.Language]
	requestID := uuid.NewString()

	_ = h.FileHandler.WriteFile(&files.File{
		ID:   requestID,
		Name: compiler.SourceFile,
		Data: []byte(direct.SourceCode),
	})

	bytes, _ := json.Marshal(queue.CompileMessage{
		ID:                 requestID,
		Language:           direct.Language,
		StdinData:          direct.StdinData,
		ExpectedStdoutData: direct.ExpectedStdoutData,
	})

	err := h.Queue.SubmitMessageToQueue(bytes)

	if err != nil {
		log.Error().Err(err)

		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"failed to execute compile request"},
			Code:   0,
		}, http.StatusInternalServerError)

		return
	}

	dbErr := h.Repo.InsertExecution(&repository.Execution{
		ID:         requestID,
		Language:   direct.Language,
		Status:     sandbox.NotRan.String(),
		TestStatus: sandbox.TestNotRan.String(),
	})

	if dbErr != nil {
		log.Error().Err(dbErr).Msg("failed to create execution record")

		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"failed to create execution record"},
			Code:   0,
		}, http.StatusInternalServerError)

		return
	}

	handleJSONResponse(w, QueueCompileResponse{ID: requestID}, http.StatusOK)
}

func (h CompilerHandlers) HandleGetCompileResponse(w http.ResponseWriter, r *http.Request) {
	uuidValue, ok := mux.Vars(r)["id"]

	if !ok {
		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"no or invalid compile request id provided."},
			Code:   0,
		}, http.StatusBadRequest)

		return
	}

	parsedIDValue, err := uuid.Parse(uuidValue)

	if err != nil {
		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"failed to parse id value"},
			Code:   0,
		}, http.StatusBadRequest)

		return
	}

	execution, err := h.Repo.GetExecution(parsedIDValue.String())

	if errors.Is(err, gorm.ErrRecordNotFound) {
		handleJSONResponse(w, CompileErrorResponse{
			Errors: []string{"the execution does not exist by the provided id."},
			Code:   0,
		}, http.StatusNotFound)

		return
	}

	resp := CompileInfoResponse{
		Status:          execution.Status,
		TestStatus:      execution.TestStatus,
		CompileMs:       execution.CompileMs,
		Language:        execution.Language,
		RuntimeMs:       execution.RuntimeMs,
		RuntimeMemoryMb: execution.RuntimeMemoryMb,
		Output:          "",
		OutputErr:       "",
	}

	compiler := sandbox.Compilers[execution.Language]

	if data, outputErr := h.FileHandler.GetFile(parsedIDValue.String(), compiler.OutputFile); outputErr == nil {
		log.Debug().Str("data", string(data)).Msg("data")
		resp.Output = string(data)
	}

	if data, outputErr := h.FileHandler.GetFile(parsedIDValue.String(), compiler.OutputErrFile); outputErr == nil {
		log.Debug().Str("data", string(data)).Msg("data")
		resp.OutputErr = string(data)
	}

	if data, outputErr := h.FileHandler.GetFile(parsedIDValue.String(), compiler.CompilerOutputFile); outputErr == nil {
		log.Debug().Str("data", string(data)).Msg("data")
		resp.CompilerOutput = string(data)
	}

	handleJSONResponse(w, resp, http.StatusOK)
}
