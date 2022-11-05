package main

import (
	"net/http"
	"os"
	"path/filepath"

	"compile-and-run-sandbox/internal/config"
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
	"compile-and-run-sandbox/internal/routing"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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

	r := mux.NewRouter()

	validate := validator.New()
	translator := getTranslator()

	// register the validator with the translator to get clean readable generated
	// error messages from validation actions. This massively simplifies returning
	// error messages in the future.
	_ = enTranslations.RegisterDefaultTranslations(validate, translator)

	compileHandlers := routing.CompilerHandlers{
		FileHandler: fileHandler,
		Repo:        repo,
		Translator:  translator,
		Validator:   validate,
		Queue:       queueRunner,
	}

	r.HandleFunc("/compile", compileHandlers.HandleCompileRequest).Methods(http.MethodPost)
	r.HandleFunc("/compile/{id}", compileHandlers.HandleGetCompileResponse).Methods(http.MethodGet)

	r.HandleFunc("/languages", routing.HandleListLanguagesSupported).Methods(http.MethodGet)
	r.HandleFunc("/languages/{lang}/template", routing.HandleGetLanguageTemplate).Methods(http.MethodGet)

	if config.GetCurrentEnvironment() == config.DevelopmentEnvironment {
		r.PathPrefix("/").Handler(http.FileServer(http.Dir("./assets/sample-site/")))
	}

	log.Info().Msg("listening on :8080")

	r.Use(mux.CORSMethodMiddleware(r))

	credentialsOk := handlers.AllowCredentials()
	originsOk := handlers.AllowedOrigins([]string{"*"})
	headersOk := handlers.AllowedHeaders([]string{"Content-Type"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	handler := handlers.CORS(credentialsOk, headersOk, originsOk, methodsOk)(r)

	if listenErr := http.ListenAndServe(":8080", handler); listenErr != nil {
		log.Fatal().Err(listenErr).Msg("failed to listen")
	}
}
