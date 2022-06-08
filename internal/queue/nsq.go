package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"compile-and-run-sandbox/internal/repository"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/sandbox"
)

type NsqQueue struct {
	config *NsqConfig

	consumer *nsq.Consumer
	producer *nsq.Producer
}

func newNsqQueue(config *NsqConfig) (NsqQueue, error) {
	queue := NsqQueue{config: config}

	if config.Consumer {
		newConsumer, err := NewNsqConsumer(config, queue)

		if err != nil {
			return queue, err
		}

		queue.consumer = newConsumer
	}

	if config.Producer {
		newProducer, err := NewNsqProducer(config)

		if err != nil {
			return queue, err
		}

		queue.producer = newProducer
	}

	return queue, nil
}

func (n NsqQueue) HandleIncomingRequest(data []byte) error {

	var compileMsg CompileMessage

	if err := json.Unmarshal(data, &compileMsg); err != nil {
		return errors.Wrap(err, "failed to parse compile request")
	}

	log.Info().
		Str("id", compileMsg.ID).
		Str("language", compileMsg.Language).
		Msg("handling new compile request")

	sourceCode, _ := n.config.FilesHandler.GetFile(compileMsg.ID, "source")
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

	_ = n.config.Repo.UpdateExecutionStatus(compileMsg.ID, sandbox.Created.String())
	containerID, complete, err := n.config.Manager.AddContainer(ctx, &sandboxRequest)

	if err != nil {
		_ = n.config.Repo.UpdateExecutionStatus(compileMsg.ID, sandbox.NonDeterministicError.String())
		return errors.Wrap(err, "failed to add container to Manager")
	}

	_ = n.config.Repo.UpdateExecutionStatus(compileMsg.ID, sandbox.Running.String())

	<-complete

	resp := n.config.Manager.GetResponse(ctx, containerID)

	_ = n.config.FilesHandler.WriteFile(sandboxRequest.ID, "output",
		[]byte(strings.Join(resp.Output, "\r\n")))

	_ = n.config.Manager.RemoveContainer(context.Background(), containerID, false)

	_, _ = n.config.Repo.UpdateExecution(compileMsg.ID, &repository.Execution{
		Status:     resp.Status.String(),
		TestStatus: resp.TestStatus.String(),
		CompileMs:  resp.CompileTime.Milliseconds(),
		RuntimeMs:  resp.Runtime.Milliseconds(),
	})

	return nil
}

func (n NsqQueue) SubmitMessageToQueue(data []byte) error {
	err := n.producer.Publish(n.config.Topic, data)

	if err != nil {
		return errors.Wrap(err, "failed to publish message onto the queue")
	}

	return nil
}

func (n NsqQueue) Stop() {
	log.Info().Msg("stopping NSQ consumer")
	n.consumer.Stop()
}

func (n NsqQueue) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		return nil
	}

	if err := n.HandleIncomingRequest(m.Body); err != nil {
		log.Err(err).Msg("failed to handle incoming compile request")
		return err
	}

	return nil
}

func NewNsqProducer(params *NsqConfig) (*nsq.Producer, error) {
	address := fmt.Sprintf("%s:%d", params.NsqLookupAddress, params.NsqLookupPort)
	return nsq.NewProducer(address, nsq.NewConfig())
}

func NewNsqConsumer(params *NsqConfig, handler nsq.Handler) (*nsq.Consumer, error) {
	config := nsq.NewConfig()
	config.MaxInFlight = params.MaxInFlight

	consumer, err := nsq.NewConsumer(params.Topic, params.Channel, config)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create NSQ consumer")
	}

	consumer.AddConcurrentHandlers(handler, params.MaxInFlight)

	address := fmt.Sprintf("%s:%d", params.NsqLookupAddress, params.NsqLookupPort)
	err = consumer.ConnectToNSQD(address)

	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to NSQ lookup")
	}

	return consumer, nil
}
