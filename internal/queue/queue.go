package queue

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/repository"
	"compile-and-run-sandbox/internal/sandbox"
)

type CompileMessage struct {
	ID                 string   `json:"id"`
	Language           string   `json:"language"`
	StdinData          []string `json:"stdin_data"`
	ExpectedStdoutData []string `json:"expected_stdout_data"`
}

type NsqConfig struct {
	Topic            string
	Channel          string
	NsqLookupAddress string
	NsqLookupPort    int
	MaxInFlight      int

	Consumer bool
	Producer bool

	Manager *sandbox.ContainerManager

	Repo repository.Repository

	FilesHandler files.Files
}

type SqsConfig struct {
	// QueueURL is the URL to the SQS queue used to write and read pending executions.
	// If not defined local mode is assumed and NSQ will be attempted instead.
	QueueURL        string
	WaitTimeSeconds int
	MaxInFlight     int

	Consumer bool

	Manager *sandbox.ContainerManager

	Repo repository.Repository

	FilesHandler files.Files
}

type Config struct {
	// The configuration for the NSQ queue which is used in local mode. This will only
	// be used if SqsQueue is not defined or local mode is enforced.
	Nsq *NsqConfig

	// The configuration for the SQS queue. This will only be used if SqsQueue
	// is not defined or local mode is enforced.
	Sqs *SqsConfig

	// If local mode should be forced or not regardless if the SqsQueue is configured.
	ForceLocalMode bool
}

type Queue interface {
	HandleIncomingRequest(data []byte) error
	SubmitMessageToQueue(data []byte) error
	Stop()
}

func NewQueue(config *Config) (Queue, error) {
	if config.ForceLocalMode || config.Sqs == nil || config.Sqs.QueueURL == "" {
		return newNsqQueue(config.Nsq)
	}

	return newSqsQueue(config.Sqs)
}

func handleNewCompileRequest(data []byte, manager *sandbox.ContainerManager, repo repository.Repository, fileHandler files.Files) error {
	var compileMsg CompileMessage

	if err := json.Unmarshal(data, &compileMsg); err != nil {
		return errors.Wrap(err, "failed to parse compile request")
	}

	log.Info().
		Str("id", compileMsg.ID).
		Str("language", compileMsg.Language).
		Msg("handling new compile request")

	compiler := sandbox.Compilers[compileMsg.Language]

	sourceCode, _ := fileHandler.GetFile(compileMsg.ID, compiler.SourceFile)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	sandboxRequest := sandbox.Request{
		ID:               compileMsg.ID,
		ExecutionProfile: sandbox.GetProfileForMachine(),
		Path:             filepath.Join(os.TempDir(), "executions", "raw", compileMsg.ID),
		SourceCode:       string(sourceCode),
		Compiler:         compiler,
		Test:             nil,
	}

	if len(compileMsg.StdinData) > 0 || len(compileMsg.ExpectedStdoutData) > 0 {
		sandboxRequest.Test = &sandbox.Test{
			ID:                 compileMsg.ID,
			StdinData:          compileMsg.StdinData,
			ExpectedStdoutData: compileMsg.ExpectedStdoutData,
		}
	}

	_ = repo.UpdateExecutionStatus(compileMsg.ID, sandbox.Created.String())
	containerID, complete, err := manager.AddContainer(ctx, &sandboxRequest)

	if err != nil {
		_ = repo.UpdateExecutionStatus(compileMsg.ID, sandbox.NonDeterministicError.String())
		return errors.Wrap(err, "failed to add container to Manager")
	}

	_ = repo.UpdateExecutionStatus(compileMsg.ID, sandbox.Running.String())

	// this needs a timeout otherwise It's going to continue until the container
	// has run too long. It's probably something that should happen inside the
	// manager to close this after failure but for now lets just timeout after
	// a selective amount of time.
	maxTimeout := sandboxRequest.ExecutionProfile.CodeTimeout +
		sandboxRequest.ExecutionProfile.CompileTimeout

	select {
	case <-complete:
	case <-time.After(maxTimeout):
		log.Error().
			Str("id", containerID).
			Dur("duration", maxTimeout).
			Msg("entire container execution timeout")

		return nil
	}

	resp := manager.GetResponse(ctx, containerID)

	uploadFiles := []*files.File{{
		ID:   sandboxRequest.ID,
		Name: compiler.OutputFile,
		Data: []byte(strings.Join(resp.Output, "\n")),
	}, {
		ID:   sandboxRequest.ID,
		Name: compiler.OutputErrFile,
		Data: []byte(strings.Join(resp.OutputError, "\n")),
	}}

	if !sandboxRequest.Compiler.Interpreter {
		uploadFiles = append(uploadFiles, &files.File{
			ID:   sandboxRequest.ID,
			Name: compiler.CompilerOutputFile,
			Data: []byte(strings.Join(resp.CompilerOutput, "\n")),
		})
	}

	_ = fileHandler.WriteFiles(uploadFiles...)

	_ = manager.RemoveContainer(context.Background(), containerID, false)

	_, _ = repo.UpdateExecution(compileMsg.ID, &repository.Execution{
		Status:          resp.Status.String(),
		TestStatus:      resp.TestStatus.String(),
		CompileMs:       resp.CompileTime.Milliseconds(),
		RuntimeMs:       resp.Runtime.Milliseconds(),
		RuntimeMemoryMb: resp.RuntimeMemory.Megabytes(),
	})

	return nil
}
