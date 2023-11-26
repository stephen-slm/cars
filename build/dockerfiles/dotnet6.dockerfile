FROM golang:1.19-buster as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM mcr.microsoft.com/dotnet/sdk:6.0

COPY --from=BUILDER /runner /runner

RUN mkdir /template-f
RUN mkdir /template-c

RUN cd /template-f && dotnet new console -lang F# -f net6.0
RUN cd /template-c && dotnet new console -lang c# -f net6.0

RUN rm /template-f/Program.fs
RUN rm /template-c/Program.cs

RUN apt-get  update
RUN apt-get install coreutils
