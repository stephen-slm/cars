FROM golang:1.18 as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM pypy:2.7-slim

COPY --from=BUILDER /runner /runner
RUN apt-get install coreutils
