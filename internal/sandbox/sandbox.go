package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	"compile-and-run-sandbox/internal/sandbox/unix"
)

type SandboxTestStatus int
type SandboxStatus int

const (
	NoTest SandboxTestStatus = iota
	TestFailed
	TestPassed
)

const (
	// NotRan - The test case has not yet executed. This is the default case for the test. And
	// should only be updated if and when the test has run and exceeded or ran and failed.
	NotRan SandboxStatus = iota

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
	// The internal id of the request, this will be used to ensure that when the response comes
	// through that there is a related id to match it up with th request.
	ID string
	// The max amount of timeout for the given executed code, if the code docker container is running
	// for longer than the given timeout then the code is rejected. This is used to ensure that the
	// source code is not running for longer than required.
	Timeout int

	// The max amount of timeout for a given container to execute the entire code including compiling.
	// if this is not set then it will be based on timeout + 50%.
	ContainerTimeout int

	// The upper limit of the max amount of memory that the given execution can perform. By default, the upper
	// limit of the amount of mb the given execution can run with.
	MemoryConstraint int64
	// The given path that would be mounted and shared with the given docker container. This is where
	// the container will be reading the source code from and writing the response too. Once this has
	// been completed, this is the path to files that will be cleaned up.
	Path string
	// The source code that will be executed, this is the code that will be written to the path and
	// mounted to the docker container.
	SourceCode []string
	// The reference details of the compilerName that will be running the code. Including details of the
	// language, compilerName name (or interrupter) and the name of the given output file.
	Compiler LanguageCompiler
	// The related test that will be executed with the sandbox, comparing a given input with
	// a given output. This is a optional part since the process could just be completing the
	// code and not actually testing anything.
	Test *Test
}

type ExecutionParameters struct {
	language      string   `json:"language"`
	StandardOut   string   `json:"standardOut"`
	StandardInput string   `json:"standardInput"`
	CompileSteps  []string `json:"compileSteps"`
	Run           string   `json:"runSteps"`
	RunTimeoutSec int      `json:"runTimeoutSec"`
}

type Response struct {
	// The raw output that was produced by the sandbox.
	Output []string

	// The given status of the sandbox.
	Status SandboxStatus

	// The complete runtime of the container in milliseconds.
	RuntimeMs time.Duration

	// The complete compile time of the container in milliseconds (if not interpreter)
	CompileMs time.Duration

	// The result for the test if it was provided.
	TestStatus SandboxTestStatus
}

type SandboxContainer struct {
	ID     string
	status SandboxStatus
	events []events.Message

	runtimeMs time.Duration
	compileMs time.Duration

	output []string

	complete chan string

	client  *client.Client
	request Request
}

func NewSandboxContainer(request Request, client *client.Client) *SandboxContainer {
	return &SandboxContainer{
		ID:      "",
		status:  NotRan,
		client:  client,
		request: request,
		events:  []events.Message{},
	}
}

// Run the sandbox container with the given configuration options.
func (d *SandboxContainer) Run(ctx context.Context) (string, <-chan string, error) {
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
func (d *SandboxContainer) prepare(_ context.Context) error {
	// Create the temporary directory that will be used for storing the source code, standard
	// input and then the location in which the compilerName will write the standard output and the
	// standard error output. After the data is written and returned, the location will be
	// deleted.
	if err := os.MkdirAll(d.request.Path, 0750); err != nil {
		return err
	}

	sourceFileName := "source"
	sourceFilePath := filepath.Join(d.request.Path, sourceFileName)

	// Go through the process of writing down the source file to disk, this will be used
	// and read again when gathering the results.
	sourceFile, sourceFileErr := os.Create(sourceFilePath)
	defer sourceFile.Close()

	if sourceFileErr != nil {
		return sourceFileErr
	}

	for _, s := range d.request.SourceCode {
		if _, writeErr := sourceFile.WriteString(s + "\r\n"); writeErr != nil {
			return writeErr
		}
	}

	inputFilePath := filepath.Join(d.request.Path, d.request.Compiler.InputFile)

	// Go through the process of writing down the input file to disk, this will be used
	// and read again when gathering the results.
	inputFile, inputFileErr := os.Create(inputFilePath)
	defer inputFile.Close()

	if inputFileErr != nil {
		return inputFileErr
	}

	if d.request.Test != nil {
		for _, s := range d.request.Test.StdinData {
			if _, writeErr := inputFile.WriteString(fmt.Sprintf("%s\n", s)); writeErr != nil {
				return writeErr
			}
		}
	}

	// Create the standard output file and standard error output file, these will be directed
	// towards when the source code file is compiled or the interpreted file is executed.
	sourceOut := filepath.Join(d.request.Path, d.request.Compiler.OutputFile)
	runnerConfig := filepath.Join(d.request.Path, "runner.json")

	OutFile, standardErr := os.Create(sourceOut)
	defer OutFile.Close()

	if standardErr != nil {
		return standardErr
	}

	parameters := ExecutionParameters{
		language:      d.request.Compiler.language,
		RunTimeoutSec: d.request.Timeout,
		StandardOut:   d.request.Compiler.OutputFile,
		StandardInput: d.request.Compiler.InputFile,
		CompileSteps:  d.request.Compiler.compileSteps,
		Run:           d.request.Compiler.runSteps,
	}

	runnerFile, runnerError := os.Create(runnerConfig)
	defer runnerFile.Close()

	if runnerError != nil {
		return runnerError
	}

	runnerJsonBytes, _ := json.Marshal(parameters)
	if _, writeErr := runnerFile.Write(runnerJsonBytes); writeErr != nil {
		return writeErr
	}

	return nil

}

// execute the sandbox environment, building up the arguments, creating the container and starting
// it. Everything after this point will be based on the stream of data being produced by the
// docker stream.
func (d *SandboxContainer) execute(ctx context.Context) error {
	commandLine := []string{"/runner"}

	// The working directory just be in a unix based absolute format otherwise its not
	// going to work as expected and thus needs to be converted to ensure that it is in
	// that format.
	// var workingDirectory = ConvertPathToUnix(this._path)
	workingDirectory := unix.ConvertPathToUnix(d.request.Path)
	containerTimeout := d.request.ContainerTimeout

	create, err := d.client.ContainerCreate(
		ctx,
		&container.Config{
			Entrypoint:      commandLine,
			Image:           d.request.Compiler.VirtualMachineName,
			NetworkDisabled: true,
			StopTimeout:     &containerTimeout,
			WorkingDir:      "/input",
		},
		&container.HostConfig{
			AutoRemove: true,
			Binds:      []string{fmt.Sprintf("%s:/input", workingDirectory)},
			Resources: container.Resources{
				Memory: d.request.MemoryConstraint * 1000000,
			},
		},
		nil,
		nil,
		"",
	)

	if err != nil {
		return err
	}

	d.ID = create.ID

	return d.client.ContainerStart(ctx, d.ID, types.ContainerStartOptions{})
}

// cleanup will remove all the files related to this container on call.
func (d *SandboxContainer) cleanup() error {
	close(d.complete)

	if d.request.Path != "" {
		return os.RemoveAll(d.request.Path)
	}

	return nil
}

// Loads the response of the sandbox execution, the standard output.
func (d *SandboxContainer) getSandboxStandardOutput() ([]string, error) {
	path := filepath.Join(d.request.Path, d.request.Compiler.OutputFile)
	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "*-COMPILE::EOF-*") {
			// let's get the runtime/compile complexity, we can use this to
			// determine if the execution went passed the time limit.
			splitLine := strings.Split(line, " ")

			runtime := splitLine[1]
			compile := splitLine[2]
			code := splitLine[3]

			runtimeDuration, _ := strconv.Atoi(runtime)
			compileDuration, _ := strconv.Atoi(compile)
			statusCode, _ := strconv.Atoi(code)

			d.runtimeMs = time.Duration(runtimeDuration) * time.Nanosecond
			d.compileMs = time.Duration(compileDuration) * time.Nanosecond
			d.status = SandboxStatus(statusCode)

			return lines, nil
		}

		lines = append(lines, line)
	}

	return lines, nil
}

func (d *SandboxContainer) AddDockerEventMessage(event events.Message) {
	d.updateStatusFromDockerEvent(event.Status)
	d.events = append(d.events, event)
}

func (d *SandboxContainer) updateStatusFromDockerEvent(status string) {
	switch status {
	case "create":
		d.handleContainerCreated()
		break
	case "start":
		d.handleContainerStarted()
		break
	case "kill":
		d.handleContainerKilling()
		break
	case "die":
		d.handleContainerKilled()
		break
	case "destroy":
		d.handleContainerRemoved()
		break
	default:
		fmt.Printf("unhandled status %s for container %s\n", status, d.ID)
		break
	}
}

// Handles the case in which the given container has been created.
func (d *SandboxContainer) handleContainerCreated() {
	d.status = Created
}

// Handles the case in which the given container has been started.
func (d *SandboxContainer) handleContainerStarted() {
	d.status = Running
}

// Handles the case in which the given container is being killed.
func (d *SandboxContainer) handleContainerKilling() {
	d.status = Killing
}

// Handles the case in which the given container has been killed.
func (d *SandboxContainer) handleContainerKilled() {
	d.status = Killed
}

// Handles the case in which the given container has been removed.
func (d *SandboxContainer) handleContainerRemoved() {
	defer d.cleanup()

	output, _ := d.getSandboxStandardOutput()
	d.output = output
}

// GetResponse - Get the response of the sandbox, can only be called once in removed state.
func (d *SandboxContainer) GetResponse() *Response {
	testStatus := NoTest

	// the test is specified and the status is finished so lets go and verify
	// that the test passed by verifying the out content with the expected content.
	if d.status == Finished && d.request.Test != nil {
		testStatus = TestPassed

		if len(d.output) != len(d.request.Test.ExpectedStdoutData) {
			testStatus = TestFailed
		} else {
			for i, expectedData := range d.request.Test.ExpectedStdoutData {
				if d.output[i] != expectedData {
					testStatus = TestFailed
					break
				}
			}
		}
	}

	return &Response{
		Output:     d.output,
		Status:     d.status,
		TestStatus: testStatus,
		RuntimeMs:  d.runtimeMs,
		CompileMs:  d.compileMs,
	}
}
