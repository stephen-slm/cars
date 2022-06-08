package queue

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go/aws/session"
)

type SqsQueue struct {
	config *SqsConfig

	sqsQueue *sqs.SQS

	stopFlag bool
}

func newSqsQueue(config *SqsConfig) (SqsQueue, error) {
	queue := SqsQueue{config: config}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	queue.sqsQueue = sqs.New(sess)

	// if we are a consumer lets go and start polling for messages
	// this will be in its own go routine which will have a
	// stop flag for each iteration to check.
	if queue.config.Consumer {
		go queue.startPollingMessages()
	}

	return queue, nil
}

func (s SqsQueue) startPollingMessages() {
	for {
		// if we are being told to stop, lets go and break out and
		// exist the entire go routine.
		if s.stopFlag {
			break
		}

		output, err := s.sqsQueue.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.config.QueueURL),
			MaxNumberOfMessages: aws.Int64(int64(s.config.MaxInFlight)),
			WaitTimeSeconds:     aws.Int64(int64(s.config.WaitTimeSeconds)),
		})

		if err != nil {
			log.Error().Err(err).Msg("failed to gather SQS messages")
		}

		wg := sync.WaitGroup{}

		for _, message := range output.Messages {
			if len(*message.Body) == 0 {
				continue
			}

			wg.Add(1)

			go func(m *sqs.Message) {
				defer func() {
					if _, deleteErr := s.sqsQueue.DeleteMessage(&sqs.DeleteMessageInput{
						QueueUrl:      aws.String(s.config.QueueURL),
						ReceiptHandle: m.ReceiptHandle,
					}); deleteErr != nil {
						log.Err(deleteErr).Str("id", *m.MessageId).
							Msg("failed to handle incoming compile request")
					}

					wg.Done()
				}()

				// I don't want to use channels in this situation as this will become
				// my blocker. I only want this amount of messages in flight, if I pick
				// up channels then they will not block until completion.

				if handleErr := s.HandleIncomingRequest([]byte(*m.Body)); handleErr != nil {
					log.Err(handleErr).Str("id", *m.MessageId).
						Msg("failed to handle incoming compile request")
				}
			}(message)
		}

		wg.Wait()
	}

}

func (s SqsQueue) HandleIncomingRequest(data []byte) error {
	return handleNewCompileRequest(data, s.config.Manager, s.config.Repo, s.config.FilesHandler)
}

func (s SqsQueue) SubmitMessageToQueue(data []byte) error {
	_, err := s.sqsQueue.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(data)),
		QueueUrl:    aws.String(s.config.QueueURL),
	})

	return err
}

func (s SqsQueue) Stop() {
	s.stopFlag = true
}
