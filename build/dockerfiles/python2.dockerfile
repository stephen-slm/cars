FROM golang:1.21-bullseye as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM pypy:2.7-buster

COPY --from=BUILDER /runner /runner

RUN apt-get  update
RUN apt-get install coreutils
