package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/routing"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/namsral/flag"
)

type flags struct {
	sqsQueue string

	nsqAddress string
	nsqPort    int
}

func configureArgs() flags {
	args := flags{}

	flag.StringVar(&args.sqsQueue, "sqs-queue", "", "")

	flag.StringVar(&args.nsqAddress, "nsq-address", "nsqd", "")
	flag.IntVar(&args.nsqPort, "nsq-port", 4150, "")

	flag.Parse()

	log.Info().Msgf("%+v parsed arguments", args)

	return args
}

func main() {
	log.Info().Msg("starting cars-api")

	args := configureArgs()

	config := nsq.NewConfig()

	address := fmt.Sprintf("%s:%d", args.nsqAddress, args.nsqPort)
	producer, err := nsq.NewProducer(address, config)

	if err != nil {
		log.Fatal().Err(err)
	}

	r := mux.NewRouter()

	r.Handle("/", handlers.
		LoggingHandler(os.Stdout, routing.CompilerHandler{Publisher: producer})).
		Methods("POST")

	log.Info().Msg("listening on :8080")
	log.Fatal().Err(http.ListenAndServe(":8080", handlers.CompressHandler(r)))
}
