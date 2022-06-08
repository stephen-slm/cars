package queue

import (
	"fmt"

	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	return handleNewCompileRequest(data, n.config.Manager, n.config.Repo, n.config.FilesHandler)
}

func (n NsqQueue) SubmitMessageToQueue(data []byte) error {
	return n.producer.Publish(n.config.Topic, data)
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
