package main

import (
	"net"
	"os"
	"path/filepath"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"

	"compile-and-run-sandbox/internal/api/consumer"
	v1 "compile-and-run-sandbox/internal/gen/pb/content/consumer/v1"

	"google.golang.org/grpc"

	"compile-and-run-sandbox/internal/files"
	"compile-and-run-sandbox/internal/parser"
	"compile-and-run-sandbox/internal/sandbox"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	enTranslations "github.com/go-playground/validator/v10/translations/en"

	"compile-and-run-sandbox/internal/queue"
	"compile-and-run-sandbox/internal/repository"
)

func getTranslator() ut.Translator {
	english := en.New()
	uni := ut.New(english, english)
	translator, _ := uni.GetTranslator("en")

	return translator
}

func main() {
	log.Info().Msg("starting cars-api")
	args := parser.ParseDefaultConfigurationArguments()

	sandbox.LoadEmbeddedTemplateFiles()

	queueRunner, err := queue.NewQueue(&queue.Config{
		ForceLocalMode: true,

		Nsq: &queue.NsqConfig{
			Topic:            args.NsqTopic,
			Channel:          args.NsqChannel,
			NsqLookupAddress: args.NsqAddress,
			NsqLookupPort:    args.NsqPort,
			MaxInFlight:      args.MaxConcurrentContainers,
			Consumer:         false,
			Producer:         true,
		},
		Sqs: &queue.SqsConfig{
			QueueURL:        args.SqsQueue,
			WaitTimeSeconds: args.WaitTimeSeconds,
			MaxInFlight:     args.MaxConcurrentContainers,
			Consumer:        false,
		},
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to create queue")
	}

	repo, respErr := repository.NewRepository(args.DatabaseConn)

	if respErr != nil {
		log.Fatal().Err(respErr).Msg("failed to create database connection")
	}

	fileHandler, err := files.NewFilesHandler(&files.Config{
		Local:          &files.LocalConfig{LocalRootPath: filepath.Join(os.TempDir(), "executions")},
		S3:             &files.S3Config{BucketName: args.S3BucketName},
		ForceLocalMode: true,
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to create file handler")
	}

	if err != nil {
		log.Fatal().Err(err).Msg("failed to create file handler")
	}

	validate := validator.New()
	translator := getTranslator()

	// register the validator with the translator to get clean readable generated
	// error messages from validation actions. This massively simplifies returning
	// error messages in the future.
	_ = enTranslations.RegisterDefaultTranslations(validate, translator)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_validator.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	v1.RegisterConsumerServiceServer(server, &consumer.Server{
		FileHandler: fileHandler,
		Repo:        repo,
		Translator:  translator,
		Validator:   validate,
		Queue:       queueRunner,
	})

	log.Info().Msg("listening on :8080")
	if listenErr := server.Serve(lis); listenErr != nil {
		log.Fatal().Err(listenErr).Msg("failed to listen")
	}
}
