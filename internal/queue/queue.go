package queue

import (
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

type QueueConfig struct {
	// SqsQueue is the URL to the SQS queue used to write and read pending executions.
	// If not defined local mode is assumed and NSQ will be attempted instead.
	SqsQueue string

	// The configuration for the NSQ which is used in local mode. This will only
	// be used if SqsQueue is not defined or local mode is enforced.
	Nsq *NsqConfig

	// If local mode should be forced or not regardless if the SqsQueue is configured.
	ForceLocalMode bool
}

type Queue interface {
	HandleIncomingRequest(data []byte) error
	SubmitMessageToQueue(data []byte) error
	Stop()
}

func NewQueue(config *QueueConfig) (Queue, error) {
	return newNsqQueue(config.Nsq)
}
