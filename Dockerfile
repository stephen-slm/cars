FROM golang:1.18-alpine

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /app/loader

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN go build -o ./loader ./cmd/services/cars-loader/

# Run the binary program produced by `go install`
CMD ["./loader"]
