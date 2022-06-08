package parser

import (
	"github.com/namsral/flag"
	"github.com/rs/zerolog/log"
)

type Arguments struct {
	DatabaseConn            string
	MaxConcurrentContainers int
	SqsQueue                string

	NsqAddress string
	NsqChannel string
	NsqPort    int
	NsqTopic   string
}

func ParseDefaultConfigurationArguments() Arguments {
	args := Arguments{}

	flag.StringVar(&args.DatabaseConn, "database-connection-string", "host=database user=root password=root port=54320 dbname=compile TimeZone=UTC", "")
	flag.IntVar(&args.MaxConcurrentContainers, "max-concurrent-containers", 5, "")
	flag.StringVar(&args.SqsQueue, "sqs-queue", "", "")

	flag.StringVar(&args.NsqAddress, "nsq-address", "nsqd", "")
	flag.StringVar(&args.NsqChannel, "nsq-channel", "main", "")
	flag.IntVar(&args.NsqPort, "nsq-port", 4150, "")
	flag.StringVar(&args.NsqTopic, "nsq-topic", "containers", "")

	flag.Parse()
	log.Info().Msgf("%+v parsed arguments", args)

	return args
}
