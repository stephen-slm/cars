FROM golang:1.19-buster as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM mcr.microsoft.com/dotnet/sdk:6.0

COPY --from=BUILDER /runner /runner

RUN mkdir /projectf
RUN mkdir /projectc

RUN cd /projectf && dotnet new console -lang F# -f net6.0
RUN cd /projectc && dotnet new console -lang c# -f net6.0

RUN rm /projectf/Program.fs
RUN rm /projectc/Program.cs

RUN apt-get  update
RUN apt-get install coreutils
