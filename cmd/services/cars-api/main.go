package main

import (
	"net/http"
	"os"

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

	"github.com/namsral/flag"
)

type flags struct {
	databaseConn string
	sqsQueue     string
	nsqAddress   string
	nsqPort      int
}

func configureArgs() flags {
	args := flags{}

	flag.StringVar(&args.databaseConn, "database-connection-string", "host=localhost user=postgres password=root dbname=compile TimeZone=UTC", "")

	flag.StringVar(&args.sqsQueue, "sqs-queue", "", "")

	flag.StringVar(&args.nsqAddress, "nsq-address", "nsqd", "")
	flag.IntVar(&args.nsqPort, "nsq-port", 4150, "")

	flag.Parse()

	log.Info().Msgf("%+v parsed arguments", args)

	return args
}

func getTranslator() ut.Translator {

	english := en.New()
	uni := ut.New(english, english)
	translator, _ := uni.GetTranslator("en")

	return translator
}

func main() {
	log.Info().Msg("starting cars-api")

	args := configureArgs()

	producer, err := queue.NewNsqProducer(&queue.NsqParams{
		NsqLookupAddress: args.nsqAddress,
		NsqLookupPort:    args.nsqPort,
	})

	if err != nil {
		log.Fatal().Err(err)
	}

	repo, err := repository.NewRepository(args.databaseConn)

	if err != nil {
		log.Fatal().Err(err)
	}

	r := mux.NewRouter()

	validate := validator.New()
	translator := getTranslator()

	// register the validator with the translator to get clean readable generated
	// error messages from validation actions. This massively simplifies returning
	// error messages in the future.
	_ = enTranslations.RegisterDefaultTranslations(validate, translator)

	r.Handle("/", handlers.
		LoggingHandler(os.Stdout, routing.CompilerHandler{
			Db:         repo,
			Publisher:  producer,
			Translator: translator,
			Validator:  validate,
		})).
		Methods("POST")

	log.Info().Msg("listening on :8080")
	log.Fatal().Err(http.ListenAndServe(":8080", handlers.CompressHandler(r)))
}
