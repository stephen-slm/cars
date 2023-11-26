Below includes the documentation of how to set up and run the platform (cars) locally, this includes building the
language images and setting up the loader and runner. Steps to follow when making changes and how to test execute tests.

- [Prerequisite](#prerequisite)
- [Setup](#setup)
	* [Building the language containers](#building-the-language-containers)
	* [Local database and queue starting](#local-database-and-queue-starting)
	* [Running the API and loader](#running-the-api-and-loader)

# Prerequisite

| Application    | Version                         |
|----------------|---------------------------------|
| Docker         | v20.10.x                        | 
| Docker Compose | v1.29.x                         | 
| Golang         | v1.18.x                         | 
| Make           | Modern                          |
| modd           | https://github.com/cortesi/modd |

# Setup

Running the application is broken up into three parts, building the language containers, running the local queue
services
and database setup, finally setting up and running the API and loader.

| Binary | Description                                                                                                      |
|--------|------------------------------------------------------------------------------------------------------------------|
| API    | The API is the consumer-facing application that does input validation and puts the content into the queue.       |
| Loader | The loader is the application that sets up the containers and communicates with docker directly to run the code. |
| Runner | The runner is what is executed inside the container, this is what enforces limits, runs, and compiles the code.  |

## Building the language containers

Building the language containers is the first step in running the application locally. Each supported language has its
supporting image which is run on user code execution. Each image contains a copy of the runner binary which means
making changes to the runner will require re-creating all the language contains (some workaround for this).

No language is required to be installed directly on the host machine for this application to work. All are done via
docker images.

```bash
# build all images for all supported languages
make build-languages

# Build all images for all supported languages with extra logging
make build-languages/verbose

# Build a single image for a single language 
# This is easier for local runner development.
make build-languages CLANG=rust
```

## Local database and queue starting

CARS requires a local database and queue to work, the application supports setting up a database and queue via a docker
image. This can be done purely via a single docker-compose command. This will set up the queue and database. This must
be in the root of the directory.

```bash
# start the database and queue
docker-compose up -d

# stop the database and queue
docker-compose down 
```

## Running the API and loader

Finally, the loader and the API should be executed outside the docker-compose set up, this allows faster development and
turn around since the loader cannot be executed within docker as it has to communicate with the docker engine.

First, ensure `modd` is installed, this is what will be used to restart the application on start.

```bash
go install github.com/cortesi/modd/cmd/modd
```

Next, execute `modd` in the root directly after the database and queue setup. If all is completed correctly then
`INF listening on :8080` will be outputted into the console. You should now be able to access `http://localhost:8080`
in the browser to view the API sample site and run an execution to validate the setup.
