package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/namsral/flag"
)

type flags struct {
	sqsQueue string
}

func main() {
	args := flags{}

	flag.StringVar(&args.sqsQueue, "sqs-queue", "", "The name of the SQS queue that will be pushed to.")

	flag.Parse()

	r := mux.NewRouter()
	http.Handle("/", r)
}
