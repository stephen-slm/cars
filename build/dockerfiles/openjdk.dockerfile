FROM golang:1.19-buster as BUILDER

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /runner ./cmd/services/cars-runner/main.go

FROM openjdk:18-buster

ARG SCALA_VERSION="3.1.2"
ARG KOTLIN_VERSION="1.7.0"

# Installing basic packages
RUN apt-get update && \
	apt-get install -y zip unzip curl coreutils && \
	rm -rf /var/lib/apt/lists/* && \
	rm -rf /tmp/*

# Downloading SDKMAN!
RUN curl -s "https://get.sdkman.io" | bash

# Installing Java and Maven, removing some unnecessary SDKMAN files
RUN bash -c "source $HOME/.sdkman/bin/sdkman-init.sh && \
    yes | sdk install scala $SCALA_VERSION && \
    yes | sdk install kotlin $KOTLIN_VERSION && \
    rm -rf $HOME/.sdkman/archives/* && \
    rm -rf $HOME/.sdkman/tmp/* && \
    ln -s /root/.sdkman/candidates/scala/current/bin/scala /scala && \
    ln -s /root/.sdkman/candidates/scala/current/bin/scalac /scalac && \
    ln -s /root/.sdkman/candidates/kotlin/current/bin/kotlin /kotlin && \
    ln -s /root/.sdkman/candidates/kotlin/current/bin/kotlinc /kotlinc"

COPY --from=BUILDER /runner /runner
