package consumer

import (
	consumerv1 "compile-and-run-sandbox/internal/gen/pb/content/consumer/v1"
)

type Server struct {
	consumerv1.UnimplementedConsumerServiceServer
}
