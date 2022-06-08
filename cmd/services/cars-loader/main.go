package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"compile-and-run-sandbox/internal/parser"
	"compile-and-run-sandbox/internal/repository"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/queue"
	"compile-and-run-sandbox/internal/sandbox"
)

func main() {
	log.Info().Msg("starting cars-loader")
	args := parser.ParseDefaultConfigurationArguments()

	log.Info().Msg("starting docker client")
	dockerClient, dockerErr := client.NewClientWithOpts(client.FromEnv)

	if dockerErr != nil {
		log.Fatal().Err(dockerErr)
	}

	repo, err := repository.NewRepository(args.DatabaseConn)

	if err != nil {
		log.Fatal().Err(err)
	}

	manager := sandbox.NewSandboxContainerManager(dockerClient, args.MaxConcurrentContainers)
	localFileHandler := files.NewLocalFileHandler(filepath.Join(os.TempDir(), "executions"))

	log.Info().Msg("starting Queue")
	queueRunner, err := queue.NewQueue(&queue.QueueConfig{
		Nsq: &queue.NsqConfig{
			Topic:            args.NsqTopic,
			Channel:          args.NsqChannel,
			NsqLookupAddress: args.NsqAddress,
			NsqLookupPort:    args.NsqPort,
			MaxInFlight:      args.MaxConcurrentContainers,
			Consumer:         true,
			Producer:         false,
			Manager:          manager,
			Repo:             repo,
			FilesHandler:     localFileHandler,
		},
	})

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
	queueRunner.Stop()
}
