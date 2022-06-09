package queue

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/repository"
	"compile-and-run-sandbox/internal/sandbox"
)

type CompileMessage struct {
	ID                 string   `json:"id"`
	Language           string   `json:"language" validate:"required,oneof=python node"`
	StdinData          []string `json:"stdin_data" validate:"required"`
	ExpectedStdoutData []string `json:"expected_stdout_data" validate:"required"`
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

	sourceCode, _ := fileHandler.GetFile(compileMsg.ID, "source")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	sandboxRequest := sandbox.Request{
		ID:               compileMsg.ID,
		Timeout:          1,
		MemoryConstraint: 1024,
		Path:             filepath.Join(os.TempDir(), "executions", uuid.NewString()),
		SourceCode:       string(sourceCode),
		Compiler:         sandbox.Compilers[compileMsg.Language],
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

	<-complete

	resp := manager.GetResponse(ctx, containerID)

	_ = fileHandler.WriteFile(sandboxRequest.ID, "output",
		[]byte(strings.Join(resp.Output, "\r\n")))

	_ = manager.RemoveContainer(context.Background(), containerID, false)

	_, _ = repo.UpdateExecution(compileMsg.ID, &repository.Execution{
		Status:     resp.Status.String(),
		TestStatus: resp.TestStatus.String(),
		CompileMs:  resp.CompileTime.Milliseconds(),
		RuntimeMs:  resp.Runtime.Milliseconds(),
	})

	return nil
}
