package routing

import (
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
)

type CompilerHandlers struct {
	FileHandler files.Files
	Repo        repository.Repository
	Translator  ut.Translator
	Validator   *validator.Validate
	Queue       queue.Queue
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
