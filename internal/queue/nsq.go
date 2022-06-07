package queue

import (
	"compile-and-run-sandbox/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/sandbox"
)

type NsqParams struct {
	Topic            string
	Channel          string
	NsqLookupAddress string
	NsqLookupPort    int
	MaxInFlight      int
}

type NsqConsumer struct {
	consumer *nsq.Consumer
}

type nsqConsumerMessageHandler struct {
	repo         repository.Repository
	manager      *sandbox.ContainerManager
	filesHandler files.Files
}

func NewNsqProducer(params *NsqParams) (*nsq.Producer, error) {
	address := fmt.Sprintf("%s:%d", params.NsqLookupAddress, params.NsqLookupPort)
	return nsq.NewProducer(address, nsq.NewConfig())
}

func NewNsqConsumer(params *NsqParams, manager *sandbox.ContainerManager, repo repository.Repository, fileHandler files.Files) (*NsqConsumer, error) {
	config := nsq.NewConfig()
	config.MaxInFlight = params.MaxInFlight

	consumer, err := nsq.NewConsumer(params.Topic, params.Channel, config)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create NSQ consumer")
	}

	consumer.AddConcurrentHandlers(&nsqConsumerMessageHandler{
		repo:         repo,
		filesHandler: fileHandler,
		manager:      manager,
	}, params.MaxInFlight)

	address := fmt.Sprintf("%s:%d", params.NsqLookupAddress, params.NsqLookupPort)
	err = consumer.ConnectToNSQD(address)

	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to NSQ lookup")
	}

	return &NsqConsumer{}, nil
}

func (h *nsqConsumerMessageHandler) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		return nil
	}

	var compileMsg CompileMessage

	if err := json.Unmarshal(m.Body, &compileMsg); err != nil {
		return errors.Wrap(err, "failed to parse compile request")
	}

	sourceCode, _ := h.filesHandler.GetFile(compileMsg.ID, "source")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	sandboxRequest := sandbox.Request{
		ID:               compileMsg.ID,
		Timeout:          1,
		MemoryConstraint: 1024,
		Path:             fmt.Sprintf(filepath.Join(os.TempDir(), "executions", uuid.NewString())),
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

	containerID, complete, err := h.manager.AddContainer(ctx, &sandboxRequest)

	if err != nil {
		return errors.Wrap(err, "failed to add container to manager")
	}

	<-complete

	resp := h.manager.GetResponse(ctx, containerID)

	_ = h.filesHandler.WriteFile(sandboxRequest.ID, "output",
		[]byte(strings.Join(resp.Output, "\r\n")))

	_, _ = h.repo.UpdateExecution(compileMsg.ID, repository.Execution{
		Status:     resp.Status.String(),
		TestStatus: resp.TestStatus.String(),
		CompileMs:  resp.CompileTime.Milliseconds(),
		RuntimeMs:  resp.Runtime.Milliseconds(),
	})

	log.Info().
		Dur("compileMs", resp.CompileTime).
		Dur("runtimeMs", resp.Runtime).
		Str("testStatus", resp.TestStatus.String()).
		Str("status", resp.Status.String()).
		Strs("output", resp.Output).
		Msg("response")

	_ = h.manager.RemoveContainer(context.Background(), containerID, false)

	return nil
}

func (n NsqConsumer) Stop() {
	log.Info().Msg("stopping NSQ consumer")

	n.consumer.Stop()
}
