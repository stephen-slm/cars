package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/routing"
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
	manager *sandbox.ContainerManager
}

func NewNsqConsumer(params *NsqParams, manager *sandbox.ContainerManager) (*NsqConsumer, error) {
	config := nsq.NewConfig()
	config.MaxInFlight = params.MaxInFlight

	consumer, err := nsq.NewConsumer(params.Topic, params.Channel, config)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create NSQ consumer")
	}

	consumer.AddConcurrentHandlers(&nsqConsumerMessageHandler{
		manager: manager,
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

	var direct routing.CompileRequest

	if err := json.Unmarshal(m.Body, &direct); err != nil {
		return errors.Wrap(err, "failed to parse compile request.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	sandboxRequest := sandbox.Request{
		ID:               uuid.New().String(),
		Timeout:          1,
		MemoryConstraint: 1024,
		Path:             fmt.Sprintf("./temp/%s/", uuid.New().String()),
		SourceCode:       direct.SourceCode,
		Compiler:         sandbox.Compilers[direct.Language],
		Test:             nil,
	}

	if len(direct.StdinData) > 0 || len(direct.ExpectedStdoutData) > 0 {
		sandboxRequest.Test = &sandbox.Test{
			ID:                 uuid.New().String(),
			StdinData:          direct.StdinData,
			ExpectedStdoutData: direct.ExpectedStdoutData,
		}
	}

	ID, complete, err := h.manager.AddContainer(ctx, &sandboxRequest)

	if err != nil {
		return errors.Wrap(err, "failed to add container to manager.")
	}

	<-complete

	resp := h.manager.GetResponse(ctx, ID)

	log.Info().
		Dur("compileMs", resp.CompileTime).
		Dur("runtimeMs", resp.Runtime).
		Str("testStatus", resp.TestStatus.String()).
		Str("status", resp.Status.String()).
		Strs("output", resp.Output).
		Msg("response")

	_ = h.manager.RemoveContainer(context.Background(), ID, false)

	return nil
}

func (n NsqConsumer) Stop() {
	log.Info().Msg("stopping NSQ consumer")

	n.consumer.Stop()
}
