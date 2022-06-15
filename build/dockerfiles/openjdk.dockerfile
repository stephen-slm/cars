FROM golang:1.18 as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM openjdk:18-buster

RUN apt-get -y update



RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get -qq -y install curl unzip zip coreutils

# Scala
RUN curl -fL https://github.com/coursier/launchers/raw/master/cs-x86_64-pc-linux.gz | gzip -d > /cs
RUN chmod +x /cs
RUN /cs setup -y

RUN /cs install scala:3.1.2 && /cs install scalac:3.1.2

RUN mv /root/.local/share/coursier/bin/scala /scala
RUN mv /root/.local/share/coursier/bin/scalac /scalac

COPY --from=BUILDER /runner /runner
