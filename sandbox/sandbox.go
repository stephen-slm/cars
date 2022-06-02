package sandbox

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	"compile-and-run-sandbox/sandbox/unix"
)

type SandboxTestResult int
type SandboxStatus int

const (
	NoTest SandboxTestResult = iota
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
	// The reference details of the compiler that will be running the code. Including details of the
	// language, compiler name (or interrupter) and the name of the given output file.
	Compiler LanguageCompiler
	// The related test that will be executed with the sandbox, comparing a given input with
	// a given output. This is a optional part since the process could just be completing the
	// code and not actually testing anything.
	Test *Test
}

type Response struct {
	// The raw output that was produced by the sandbox.
	StandardOutput []string

	// The raw error output that was produced by the sandbox.
	StandardErrorOutput []string

	// The given status of the sandbox.
	Status SandboxStatus

	// The result for the test if it was provided.
	TestStatus SandboxTestResult
}

type SandboxContainer struct {
	containerID string
	status      SandboxStatus
	events      []events.Message

	client  *client.Client
	request Request
}

func NewSandboxContainer(request Request, client *client.Client) *SandboxContainer {
	return &SandboxContainer{
		containerID: "",
		status:      NotRan,
		client:      client,
		request:     request,
		events:      []events.Message{},
	}
}

// Run the sandbox container with the given configuration options.
func (d SandboxContainer) Run(ctx context.Context) error {
	if err := d.prepare(ctx); err != nil {
		_ = d.cleanup()
		return err
	}

	if err := d.execute(ctx); err != nil {
		_ = d.cleanup()
		return err
	}

	return nil
}

// prepare the sandbox environment for execution, creates the temp file locations, writes down
// / the source code file and ensures that all properties are correct and valid for execution.
// / If all is prepared properly, no error will be returned.
func (d SandboxContainer) prepare(_ context.Context) error {
	// Create the temporary directory that will be used for storing the source code, standard
	// input and then the location in which the compiler will write the standard output and the
	// standard error output. After the data is written and returned, the location will be
	// deleted.
	if err := os.MkdirAll(d.request.Path, os.ModeDir); err != nil {
		return err
	}

	sourceFileName := fmt.Sprintf("%s.source", d.request.Compiler.language)
	sourceFilePath := filepath.Join(d.request.Path, sourceFileName)

	// Go through the process of writing down the source file to disk, this will be used
	// and read again when gathering the results.
	sourceFile, sourceFileErr := os.Create(sourceFilePath)

	if sourceFileErr != nil {
		return sourceFileErr
	}

	for _, s := range d.request.SourceCode {
		if _, writeErr := sourceFile.WriteString(s); writeErr != nil {
			return writeErr
		}
	}

	inputFileName := fmt.Sprintf("%s.input", d.request.Compiler.language)
	inputFilePath := filepath.Join(d.request.Path, inputFileName)

	// Go through the process of writing down the input file to disk, this will be used
	// and read again when gathering the results.
	inputFile, inputFileErr := os.Create(inputFilePath)

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
	sourceStandardOut := filepath.Join(d.request.Path, d.request.Compiler.StandardOutputFile)
	sourceErrorOut := filepath.Join(d.request.Path, d.request.Compiler.StandardErrorFile)

	if _, standardErr := os.Create(sourceStandardOut); standardErr != nil {
		return standardErr
	}

	if _, errOut := os.Create(sourceErrorOut); errOut != nil {
		return errOut
	}

	// finally copy in the script file that will be executed to execute the program
	dir, _ := os.Getwd()
	bytesRead, err := ioutil.ReadFile(filepath.Join(dir, "/dockerfiles/script.sh"))

	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(d.request.Path, "script.sh"), bytesRead, 0644)
}

// execute the sandbox environment, building up the arguments, creating the container and starting
// it. Everything after this point will be based on the stream of data being produced by the
// docker stream.
func (d SandboxContainer) execute(ctx context.Context) error {
	language := d.request.Compiler.language
	compilerEntry := d.request.Compiler.compiler
	compileBinary := ""

	if !d.request.Compiler.interpreter {
		compilerEntry = fmt.Sprintf("%s.out.o", language)
	}

	commandLine := []string{
		"sh", "./script.sh",
		compilerEntry,
		fmt.Sprintf("%s.source", language),
		fmt.Sprintf("%s.input", language),
		compileBinary,
		d.request.Compiler.AdditionalArguments,
		d.request.Compiler.StandardOutputFile,
		d.request.Compiler.StandardErrorFile,
	}

	// The working directory just be in a unix based absolute format otherwise its not
	// going to work as expected and thus needs to be converted to ensure that it is in
	// that format.
	// var workingDirectory = ConvertPathToUnix(this._path)
	workingDirectory := unix.ConvertPathToUnix(d.request.Path)

	containerConfig := container.Config{
		Image:           d.request.Compiler.VirtualMachineName,
		WorkingDir:      "/input",
		Entrypoint:      commandLine,
		NetworkDisabled: true,
		StopTimeout:     &d.request.Timeout,
	}

	hostConfig := container.HostConfig{
		Binds:      []string{fmt.Sprintf("%s:/input", workingDirectory)},
		AutoRemove: true,
		Resources: container.Resources{
			Memory: d.request.MemoryConstraint * 1000000,
		},
	}

	create, err := d.client.ContainerCreate(ctx, &containerConfig,
		&hostConfig, nil, nil, "")

	if err != nil {
		return err
	}

	d.containerID = create.ID

	return d.client.ContainerStart(ctx, d.containerID, types.ContainerStartOptions{})
}

// cleanup will remove all the files related to this container on call.
func (d SandboxContainer) cleanup() error {
	if d.request.Path != "" {
		return os.RemoveAll(d.request.Path)
	}
	return nil
}

// Loads the response of the sandbox execution, the standard output.
func (d SandboxContainer) getSandboxStandardOutput() ([]string, error) {
	path := filepath.Join(d.request.Path, d.request.Compiler.StandardOutputFile)
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "*-COMPILE::EOF-*") {
			return lines, nil
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// Loads the response of the sandbox error execution, the standard error output.
func (d SandboxContainer) getSandboxStandardErrorOutput() ([]string, error) {
	path := filepath.Join(d.request.Path, d.request.Compiler.StandardErrorFile)
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())

		if len(lines) > 50 {
			break
		}
	}

	return lines, nil
}

func (d SandboxContainer) AddDockerEventMessage(event events.Message) {
	d.updateStatusFromDockerEvent(event.Status)
	d.events = append(d.events, event)
}

func (d SandboxContainer) updateStatusFromDockerEvent(status string) {
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
		fmt.Printf("unhandled status %s for container %s\n", status, d.containerID)
		break
	}
}

// Handles the case in which the given container has been created.
func (d SandboxContainer) handleContainerCreated() {
	d.status = Created
}

// Handles the case in which the given container has been started.
func (d SandboxContainer) handleContainerStarted() {
	d.status = Running
}

// Handles the case in which the given container is being killed.
func (d SandboxContainer) handleContainerKilling() {
	d.status = Killing
}

// Handles the case in which the given container has been killed.
func (d SandboxContainer) handleContainerKilled() {
	// Ensure that the status is the last thing updated, since d will trigger the
	// event, and we don't want the worker service knowing we are "killed" until
	// ready.
	d.status = Killed
}

// Handles the case in which the given container has been removed.
func (d SandboxContainer) handleContainerRemoved() {
	d.status = Finished
}

// GetResponse - Get the response of the sandbox, can only be called once in removed state.
func (d SandboxContainer) GetResponse() Response {
	defer func(d SandboxContainer) {
		_ = d.cleanup()
	}(d)

	if d.status == TimeLimitExceeded || d.status == MemoryConstraintExceeded {
		return Response{
			StandardOutput:      nil,
			StandardErrorOutput: nil,
			Status:              d.status,
		}
	}

	errorOutput, _ := d.getSandboxStandardErrorOutput()
	standardOut, _ := d.getSandboxStandardOutput()

	testStatus := NoTest

	// the test is specified and the status is finished so lets go and verify
	// that the test passed by verifying the out content with the expected content.
	if d.status == Finished && d.request.Test != nil {
		testStatus = TestPassed

		for i, expectedData := range d.request.Test.ExpectedStdoutData {
			if standardOut[i] != expectedData {
				testStatus = TestFailed
				break
			}
		}
	}

	return Response{
		StandardOutput:      errorOutput,
		StandardErrorOutput: standardOut,
		Status:              d.status,
		TestStatus:          testStatus,
	}
}
