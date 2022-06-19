//go:generate stringer -type=ContainerStatus
//go:generate stringer -type=ContainerTestStatus

package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	"compile-and-run-sandbox/internal/sandbox/unix"
)

type ContainerTestStatus int
type ContainerStatus int

const (
	NoTest ContainerTestStatus = iota
	TestNotRan
	TestFailed
	TestPassed
)

const (
	// NotRan - The test case has not yet executed. This is the default case for the test. And
	// should only be updated if and when the test has run and exceeded or ran and failed.
	NotRan ContainerStatus = iota

	Created
	Running
	Killing
	Killed
	Finished

	MemoryConstraintExceeded
	TimeLimitExceeded
	ProvidedTestFailed
	CompilationFailed
	RunTimeError
	NonDeterministicError
)

type Test struct {
	// The internal id of the test, this will be used to ensure that when the response comes
	// through that there is a related id to match it up with th request.
	ID string
	// The standard input data that will be used with the given code file. This can be used for when
	// projects require that a given code input should  be executing after reading input. e.g. taking
	// in a input and performing actions on it.
	StdinData []string
	// The expected standard output for the test case. After execution of the standard input, and
	// the data has been returned. This is what we are going to ensure the given test case matches
	// before providing a result.
	ExpectedStdoutData []string
}

type Request struct {
	// The internal id of the request, this will be used to ensure that when the
	// response comes through that there is a related id to match it up with the
	// request.
	ID string
	// The ExecutionProfile describes the configuration of the container duration
	// execution, including its upper and lower limits. Its max compile and
	// runtime values.
	ExecutionProfile *Profile
	// The given path that would be mounted and shared with the given docker
	// container. This is where the container will be reading the source code
	// from and writing the response too. Once this has been completed, this
	// is the path to files that
	// will be cleaned up.
	Path string
	// The source code that will be executed, this is the code that will be
	// written to the path and mounted to the docker container.
	SourceCode string
	// The reference details of the compilerName that will be running the code.
	// Including details of the language, compilerName name (or interrupter)
	// and the name of the given output file.
	Compiler *LanguageCompiler
	// The related test that will be executed with the sandbox, comparing a
	// given input with a given output. This is an optional part since the
	// process could just be completing the code and not actually testing
	// anything.
	Test *Test
}

type ExecutionParameters struct {
	CompileSteps   []string      `json:"compileSteps"`
	CompileTimeout time.Duration `json:"compileTimeoutSec"`
	ID             string        `json:"id"`
	Language       string        `json:"language"`
	Run            string        `json:"runSteps"`
	RunTimeout     time.Duration `json:"runTimeoutSec"`
	StandardInput  string        `json:"standardInput"`
}

func (e2 *ExecutionParameters) MarshalZerologObject(e *zerolog.Event) {
	e.Str("id", e2.ID).
		Str("language", e2.Language).
		Bool("compiled", len(e2.CompileSteps) > 0)
}

type ExecutionResponse struct {
	Runtime        int64           `json:"runTime"`
	Status         ContainerStatus `json:"status"`
	CompileTime    int64           `json:"compileTime"`
	Output         []string        `json:"output"`
	CompilerOutput []string        `json:"compilerOutput"`
}

type Response struct {
	// The raw output that was produced by the sandbox compiler running.
	CompilerOutput []string

	// The raw output that was produced by the sandbox.
	Output []string

	// The given status of the sandbox.
	Status ContainerStatus

	// The complete runtime of the container in milliseconds.
	Runtime time.Duration

	// The complete compile time of the container in milliseconds (if not interpreter)
	CompileTime time.Duration

	// The result for the test if it was provided.
	TestStatus ContainerTestStatus
}

type Container struct {
	ID     string
	status ContainerStatus
	events []*events.Message

	executionResponse *ExecutionResponse
	complete          chan string

	client  *client.Client
	request *Request
}

func NewSandboxContainer(request *Request, dockerClient *client.Client) *Container {
	return &Container{
		ID:      "",
		status:  NotRan,
		client:  dockerClient,
		request: request,
		events:  []*events.Message{},
	}
}

// Run the sandbox container with the given configuration options.
func (d *Container) Run(ctx context.Context) (id string, complete <-chan string, err error) {
	d.complete = make(chan string, 1)

	if err := d.prepare(ctx); err != nil {
		_ = d.cleanup()
		return "", d.complete, err
	}

	if err := d.execute(ctx); err != nil {
		_ = d.cleanup()
		return "", d.complete, err
	}

	return d.ID, d.complete, nil
}

// prepare the sandbox environment for execution, creates the temp file locations, writes down
// / the source code file and ensures that all properties are correct and valid for execution.
// / If all is prepared properly, no error will be returned.
func (d *Container) prepare(_ context.Context) error {
	// Create the temporary directory that will be used for storing the source code, standard
	// input and then the location in which the compilerName will write the standard output and the
	// standard error output. After the data is written and returned, the location will be
	// deleted.
	if err := os.MkdirAll(d.request.Path, 0o750); err != nil {
		return errors.Wrap(err, "failed to make required directories")
	}

	sourceFilePath := filepath.Join(d.request.Path, d.request.Compiler.SourceFile)

	// Go through the process of writing down the source file to disk, this will be used
	// and read again when gathering the results.
	sourceFile, sourceFileErr := os.Create(sourceFilePath)

	if sourceFileErr != nil {
		return errors.Wrap(sourceFileErr, "failed to create source file")
	}

	defer func(sourceFile *os.File) {
		_ = sourceFile.Close()
	}(sourceFile)

	if _, writeErr := sourceFile.WriteString(d.request.SourceCode + "\r\n"); writeErr != nil {
		return errors.Wrap(writeErr, "failed to write source code")
	}

	inputFilePath := filepath.Join(d.request.Path, d.request.Compiler.InputFile)

	// Go through the process of writing down the input file to disk, this will be used
	// and read again when gathering the results.
	inputFile, inputFileErr := os.Create(inputFilePath)

	if inputFileErr != nil {
		return errors.Wrap(inputFileErr, "failed to create input file")
	}

	defer func(inputFile *os.File) {
		_ = inputFile.Close()
	}(inputFile)

	if d.request.Test != nil {
		for _, s := range d.request.Test.StdinData {
			if _, writeErr := inputFile.WriteString(fmt.Sprintf("%s\n", s)); writeErr != nil {
				return errors.Wrap(writeErr, "failed to write standard in data")
			}
		}
	}

	runnerConfig := filepath.Join(d.request.Path, "runner.json")

	parameters := ExecutionParameters{
		ID:             d.request.ID,
		Language:       d.request.Compiler.Language,
		RunTimeout:     d.request.ExecutionProfile.CodeTimeout,
		CompileTimeout: d.request.ExecutionProfile.CompileTimeout,
		StandardInput:  d.request.Compiler.InputFile,
		CompileSteps:   d.request.Compiler.compileSteps,
		Run:            d.request.Compiler.runSteps,
	}

	runnerFile, runnerError := os.Create(runnerConfig)

	if runnerError != nil {
		return errors.Wrap(runnerError, "failed to create runner configuration")
	}

	defer runnerFile.Close()

	runnerJSONBytes, _ := json.Marshal(parameters)
	if _, writeErr := runnerFile.Write(runnerJSONBytes); writeErr != nil {
		return errors.Wrap(writeErr, "failed to write runner configuration")
	}

	return nil

}

// execute the sandbox environment, building up the arguments, creating the container and starting
// it. Everything after this point will be based on the stream of data being produced by the
// docker stream.
func (d *Container) execute(ctx context.Context) error {
	commandLine := []string{"/runner"}

	// The working directory just be in a unix based absolute format otherwise its not
	// going to work as expected and thus needs to be converted to ensure that it is in
	// that format.
	// var workingDirectory = ConvertPathToUnix(this._path)
	workingDirectory := unix.ConvertPathToUnix(d.request.Path)

	create, err := d.client.ContainerCreate(
		ctx,
		&container.Config{
			Entrypoint:      commandLine,
			Image:           d.request.Compiler.VirtualMachineName,
			NetworkDisabled: true,
			WorkingDir:      "/input",
		},
		&container.HostConfig{
			Runtime:    d.request.ExecutionProfile.Runtime.String(),
			AutoRemove: d.request.ExecutionProfile.AutoRemove,
			Binds:      []string{fmt.Sprintf("%s:/input", workingDirectory)},
			Resources: container.Resources{
				Memory:     d.request.ExecutionProfile.Memory.Bytes(),
				MemorySwap: d.request.ExecutionProfile.MemorySwap.Bytes(),
			},
		},
		nil,
		nil,
		d.request.ID,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create container")
	}

	d.ID = create.ID

	if err := d.client.ContainerStart(ctx, d.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start the container")
	}

	return nil
}

// cleanup will remove all the files related to this container on call.
func (d *Container) cleanup() error {
	close(d.complete)

	if d.request.Path != "" {
		if removeErr := os.RemoveAll(d.request.Path); removeErr != nil {
			return errors.Wrap(removeErr, "failed to clean up temp directory")
		}
	}

	return nil
}

func (d *Container) getSandboxRunnerOutput() (*ExecutionResponse, error) {
	path := filepath.Join(d.request.Path, "runner-out.json")
	fileBytes, err := os.ReadFile(path)

	if err != nil {
		return nil, errors.Wrap(err, "failed to open runner-out.json file")
	}

	var params ExecutionResponse
	_ = json.Unmarshal(fileBytes, &params)

	d.status = params.Status

	return &params, nil
}

func (d *Container) AddDockerEventMessage(event *events.Message) {
	log.Info().
		Str("action", event.Action).
		Str("containerID", d.ID[:10]).
		Str("from", event.From).
		Str("requestID", d.request.ID).
		Str("status", event.Status).
		Str("type", event.Type).
		Msg("handling container incoming docker event")

	d.updateStatusFromDockerEvent(event.Status)
	d.events = append(d.events, event)
}

func (d *Container) updateStatusFromDockerEvent(status string) {
	switch status {
	case "create":
		d.handleContainerCreated()
	case "start":
		d.handleContainerStarted()
	case "kill":
		d.handleContainerKilling()
	case "die":
		d.handleContainerKilled()
	case "destroy":
		d.handleContainerRemoved()
	default:
		fmt.Printf("unhandled status %s for container %s\n", status, d.ID)
	}
}

// Handles the case in which the given container has been created.
func (d *Container) handleContainerCreated() {
	d.status = Created
}

// Handles the case in which the given container has been started.
func (d *Container) handleContainerStarted() {
	d.status = Running
}

// Handles the case in which the given container is being killed.
func (d *Container) handleContainerKilling() {
	d.status = Killing
}

// Handles the case in which the given container has been killed.
func (d *Container) handleContainerKilled() {
	d.status = Killed
}

// Handles the case in which the given container has been removed.
func (d *Container) handleContainerRemoved() {
	defer func(d *Container) {
		_ = d.cleanup()
	}(d)

	output, _ := d.getSandboxRunnerOutput()
	d.executionResponse = output

}

// GetResponse - Get the response of the sandbox, can only be called once in removed state.
func (d *Container) GetResponse() *Response {
	testStatus := NoTest

	// the test is specified and the status is finished so lets go and verify
	// that the test passed by verifying the out content with the expected content.
	if d.status == Finished && d.request.Test != nil {
		testStatus = TestPassed

		if len(d.executionResponse.Output) != len(d.request.Test.ExpectedStdoutData) {
			testStatus = TestFailed
		} else {
			for i, expectedData := range d.request.Test.ExpectedStdoutData {
				if d.executionResponse.Output[i] != expectedData {
					testStatus = TestFailed
					break
				}
			}
		}
	}

	return &Response{
		CompilerOutput: d.executionResponse.CompilerOutput,
		Output:         d.executionResponse.Output,
		Status:         d.executionResponse.Status,
		TestStatus:     testStatus,
		Runtime:        time.Duration(d.executionResponse.Runtime) * time.Nanosecond,
		CompileTime:    time.Duration(d.executionResponse.CompileTime) * time.Nanosecond,
	}
}
