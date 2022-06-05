package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/queue"
	"compile-and-run-sandbox/internal/sandbox"

	"github.com/namsral/flag"
)

type flags struct {
	sqsQueue                string
	maxConcurrentContainers int

	nsqTopic   string
	nsqChannel string
	nsqAddress string
	nsqPort    int
}

func configureArgs() flags {
	args := flags{}

	flag.IntVar(&args.maxConcurrentContainers, "max-concurrent-containers", 10, "")

	flag.StringVar(&args.sqsQueue, "sqs-queue", "", "")

	flag.StringVar(&args.nsqTopic, "nsq-topic", "containers", "")
	flag.StringVar(&args.nsqChannel, "nsq-channel", "main", "")
	flag.StringVar(&args.nsqAddress, "nsq-address", "nsqd", "")
	flag.IntVar(&args.nsqPort, "nsq-port", 4150, "")

	flag.Parse()

	log.Info().Msgf("%+v parsed arguments", args)

	return args
}

func main() {
	log.Info().Msg("starting cars-loader")

	args := configureArgs()

	log.Info().Msg("starting docker client")
	dockerClient, dockerErr := client.NewClientWithOpts(client.FromEnv)

	if dockerErr != nil {
		log.Fatal().Err(dockerErr)
	}

	manager := sandbox.NewSandboxContainerManager(dockerClient, args.maxConcurrentContainers)

	log.Info().Msg("starting NSQ consumer")
	nsqService, err := queue.NewNsqConsumer(&queue.NsqParams{
		Channel:          args.nsqChannel,
		MaxInFlight:      args.maxConcurrentContainers,
		NsqLookupAddress: args.nsqAddress,
		NsqLookupPort:    args.nsqPort,
		Topic:            args.nsqTopic,
	}, manager)

	if err != nil {
		log.Fatal().Err(err)
	}

	log.Info().Msg("starting sandbox manager")
	go manager.Start(context.Background())

	// wait for signal to exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	manager.Stop()
	nsqService.Stop()
}
