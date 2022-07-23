FROM golang:1.18-buster as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM gcc:12-bullseye

COPY --from=BUILDER /runner /runner
RUN apt-get install coreutils
