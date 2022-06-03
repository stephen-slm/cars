package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	CompilationFailed
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
	Compiler         string   `json:"compiler"`
	SourceFile       string   `json:"sourceFile"`
	StdInFile        string   `json:"stdInFile"`
	Out              string   `json:"out"`
	StandardOut      string   `json:"standardOut"`
	StandardErrorOut string   `json:"standardErrorOut"`
	CompileSteps     []string `json:"compileSteps"`
	RunSteps         []string `json:"runSteps"`
}

type Response struct {
	// The raw output that was produced by the sandbox.
	StandardOutput []string

	// The raw error output that was produced by the sandbox.
	StandardErrorOutput []string

	// The given status of the sandbox.
	Status SandboxStatus

	// The complete runtime of the container in milliseconds.
	RuntimeMs time.Duration

	// The complete compile time of the container in milliseconds (if not interpreter)
	CompileMs time.Duration

	// The result for the test if it was provided.
	TestStatus SandboxTestResult
}

type SandboxContainer struct {
	ID     string
	status SandboxStatus
	events []events.Message

	runtimeMs time.Duration
	compileMs time.Duration

	standardOut []string
	errorOut    []string

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
	if err := os.MkdirAll(d.request.Path, os.ModeDir); err != nil {
		return err
	}

	sourceFileName := fmt.Sprintf("%s.source", d.request.Compiler.language)
	sourceFilePath := filepath.Join(d.request.Path, sourceFileName)

	// Go through the process of writing down the source file to disk, this will be used
	// and read again when gathering the results.
	sourceFile, sourceFileErr := os.Create(sourceFilePath)
	defer sourceFile.Close()

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
	sourceStandardOut := filepath.Join(d.request.Path, d.request.Compiler.StandardOutputFile)
	sourceErrorOut := filepath.Join(d.request.Path, d.request.Compiler.StandardErrorFile)

	standardOutFile, standardErr := os.Create(sourceStandardOut)
	defer standardOutFile.Close()

	if standardErr != nil {
		return standardErr
	}

	errorOutFile, errOut := os.Create(sourceErrorOut)
	defer errorOutFile.Close()

	if errOut != nil {
		return errOut
	}

	// finally copy in the script file that will be executed to execute the program
	for _, s := range []string{"/dockerfiles/script.sh", "/dockerfiles/main.py"} {
		dir, _ := os.Getwd()
		bytesRead, err := ioutil.ReadFile(filepath.Join(dir, s))

		if err != nil {
			return err
		}

		name := strings.Split(s, "/")[2]

		if err := ioutil.WriteFile(filepath.Join(d.request.Path, name), bytesRead, 0644); err != nil {
			return err
		}
	}

	return nil

}

// execute the sandbox environment, building up the arguments, creating the container and starting
// it. Everything after this point will be based on the stream of data being produced by the
// docker stream.
func (d *SandboxContainer) execute(ctx context.Context) error {
	language := d.request.Compiler.language

	compilerEntry := d.request.Compiler.compilerName
	compileBinary := ""

	if !d.request.Compiler.interpreter {
		compileBinary = fmt.Sprintf("%s.out.o", language)
	}

	parameters := ExecutionParameters{
		Compiler:         compilerEntry,
		SourceFile:       fmt.Sprintf("%s.source", language),
		StdInFile:        fmt.Sprintf("%s.input", language),
		Out:              compileBinary,
		StandardOut:      d.request.Compiler.StandardOutputFile,
		StandardErrorOut: d.request.Compiler.StandardErrorFile,
		CompileSteps:     d.request.Compiler.compileSteps,
		RunSteps:         d.request.Compiler.runSteps,
	}

	bytes, _ := json.Marshal(parameters)
	input := string(bytes)

	commandLine := []string{
		"sh",
		"./script.sh",
		input,
		d.request.Compiler.StandardOutputFile,
		d.request.Compiler.StandardErrorFile,
	}

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
		// return os.RemoveAll(d.request.Path)
	}

	return nil
}

// Loads the response of the sandbox execution, the standard output.
func (d *SandboxContainer) getSandboxStandardOutput() ([]string, error) {
	path := filepath.Join(d.request.Path, d.request.Compiler.StandardOutputFile)
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
			runtime := strings.Split(line, " ")[1]
			compile := strings.Split(line, " ")[2]

			runtimeDuration, _ := strconv.Atoi(runtime)
			compileDuration, _ := strconv.Atoi(compile)

			d.runtimeMs = time.Duration(runtimeDuration) * time.Nanosecond
			d.compileMs = time.Duration(compileDuration) * time.Nanosecond

			return lines, nil
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// Loads the response of the sandbox error execution, the standard error output.
func (d *SandboxContainer) getSandboxStandardErrorOutput() ([]string, error) {
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

	_ = file.Close()

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

	errorOutput, _ := d.getSandboxStandardErrorOutput()
	standardOut, _ := d.getSandboxStandardOutput()

	d.errorOut = errorOutput
	d.standardOut = standardOut

	d.status = Finished

	if d.runtimeMs > time.Duration(d.request.Timeout)*time.Second {
		d.status = TimeLimitExceeded
	}
}

// GetResponse - Get the response of the sandbox, can only be called once in removed state.
func (d *SandboxContainer) GetResponse() *Response {
	testStatus := NoTest

	// the test is specified and the status is finished so lets go and verify
	// that the test passed by verifying the out content with the expected content.
	if d.status == Finished && d.request.Test != nil {
		testStatus = TestPassed

		if len(d.standardOut) != len(d.request.Test.ExpectedStdoutData) {
			testStatus = TestFailed
		} else {
			for i, expectedData := range d.request.Test.ExpectedStdoutData {
				if d.standardOut[i] != expectedData {
					testStatus = TestFailed
					break
				}
			}
		}
	}

	return &Response{
		StandardOutput:      d.errorOut,
		StandardErrorOutput: d.standardOut,
		Status:              d.status,
		TestStatus:          testStatus,
		RuntimeMs:           d.runtimeMs,
		CompileMs:           d.compileMs,
	}
}
