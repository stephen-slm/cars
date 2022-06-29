package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/memory"
	"compile-and-run-sandbox/internal/pid"
	"compile-and-run-sandbox/internal/sandbox"
)

func compileProject(ctx context.Context, params *sandbox.ExecutionParameters) (compilerOutput []string, compileTimeNano int64, err error) {
	// this has to be defined here since we always want this total time
	// and the total time is determined in to defer func.
	var timeAtExecution time.Time
	compileTimeNano = 0

	hasSteps := len(params.CompileSteps) > 0

	defer func() {
		if hasSteps {
			compileTimeNano = time.Since(timeAtExecution).Nanoseconds()
		}
	}()

	if !hasSteps {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, params.CompileTimeout)
	defer cancel()

	timeAtExecution = time.Now()

	for _, step := range params.CompileSteps {
		// Create the command with our context
		command := strings.Split(step, " ")
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)

		// This time we can simply use Output() to get the result.
		output, cmdErr := cmd.CombinedOutput()

		if len(output) != 0 {
			compilerOutput = strings.Split(string(output), "\n")
		}

		// We want to check the context error to see if the timeout was executed.
		// The error returned by cmd.Output() will be OS specific based on what
		// happens when a process is killed.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = ctx.Err()
			return
		}

		err = cmdErr
	}

	return
}

type RunExuection struct {
	standardOutput []string
	errorOutput    []string
	runtimeNano    int64
}

func runProject(ctx context.Context, params *sandbox.ExecutionParameters) (*RunExuection, error) {
	// this has to be defined here since we always want this total time
	// and the total time is determined in to defer func.
	var timeAtExecution time.Time
	resp := RunExuection{}

	defer func() {
		resp.runtimeNano = time.Since(timeAtExecution).Nanoseconds()
	}()

	ctx, cancel := context.WithTimeout(ctx, params.RunTimeout)
	defer cancel()

	// Create the command with our context
	command := strings.Split(params.Run, " ")

	inputFile, _ := os.Open(fmt.Sprintf("/input/%s", params.StandardInput))
	defer inputFile.Close()

	outputFile, _ := os.Create("/input/run-standard-output")
	outputErrFile, _ := os.Create("/input/run-error-output")

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	cmd.Stdin = inputFile
	cmd.Stdout = outputFile
	cmd.Stderr = outputErrFile

	// execution channel used to determine when to stop reading memory
	// information from the PID. Channel will be closed once the wait has
	// completed fully.
	waitNotification := make(chan any)

	timeAtExecution = time.Now()

	maxMemoryConsumption := memory.Byte
	cmdErr := cmd.Start()

	go func(ch <-chan any) {
		for {
			select {
			case <-ch:
				break
			default:
			}

			state, err := pid.GetStat(cmd.Process.Pid)

			if err != nil {
				log.Error().Err(err).Msg("failed to get pid stats")
			}

			fmt.Printf("pid %d - cpu %f - memory %dmb\n", cmd.Process.Pid, state.CPU, state.Memory.Megabytes())
			time.Sleep(10 * time.Millisecond)

			if maxMemoryConsumption < state.Memory {
				maxMemoryConsumption = state.Memory
			}
		}
	}(waitNotification)

	_ = cmd.Wait()
	close(waitNotification)

	fmt.Printf("pid: %d - max memory %dmb\n", cmd.ProcessState.Pid(), maxMemoryConsumption.Megabytes())

	// close the file after writing to allow full reading from the start
	// current implementation does not allow writing and then reading from the
	// start
	outputFile.Close()
	outputErrFile.Close()

	outputFile, _ = os.Open("/input/run-standard-output")
	outputErrFile, _ = os.Open("/input/run-error-output")

	// only take the first 1k from both the error output and the standard output
	// this is by design to stop the chance of people abusing the system and
	// writing infinitely to the output streams until it crashes the memory
	// on read.
	scanner := bufio.NewScanner(outputFile)
	scanner.Split(bufio.ScanLines)

	outputLines := make([]string, 0)
	var outputLinesCount int

	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
		outputLinesCount += 1

		if outputLinesCount >= 1_000 {
			break
		}
	}

	outputErrLines := make([]string, 0)
	var outputErrLinesCount int

	scanner = bufio.NewScanner(outputErrFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		outputErrLines = append(outputErrLines, scanner.Text())
		outputErrLinesCount += 1

		if outputErrLinesCount >= 1_000 {
			break
		}
	}

	fmt.Println("outlines", outputLinesCount, outputLines)
	fmt.Println("errLines", outputErrLinesCount, outputErrLines)

	resp.standardOutput = outputLines
	resp.errorOutput = outputErrLines

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &resp, ctx.Err()
	}

	return &resp, cmdErr
}

func main() {
	if _, err := os.Stat("/input/runner.json"); errors.Is(err, os.ErrNotExist) {
		log.Fatal().Err(err).Msg("runner.json configuration file does not exist and container cannot be executed.")
	}

	fileBytes, runnerFileErr := os.ReadFile("/input/runner.json")

	if runnerFileErr != nil {
		log.Fatal().Err(runnerFileErr).Msg("runner.json failed to be read")
	}

	var params sandbox.ExecutionParameters
	_ = json.Unmarshal(fileBytes, &params)

	responseCode := sandbox.Finished

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var runExecution = &RunExuection{}
	var compilerOutput []string
	var compileTime int64
	var compileErr, runtimeErr error

	// configure the file for th compiled output, this is the text
	// outputted when the compiler is running.
	compilerOutput, compileTime, compileErr = compileProject(ctx, &params)

	if compileErr != nil {
		log.Error().Err(compileErr).Object("request", &params).
			Msg("error occurred when executing compile")

		if errors.Is(compileErr, context.DeadlineExceeded) {
			responseCode = sandbox.TimeLimitExceeded
		} else {
			responseCode = sandbox.CompilationFailed
		}
	}

	// output file for the actual execution
	if responseCode == sandbox.Finished {

		runExecution, runtimeErr = runProject(ctx, &params)

		if runtimeErr != nil {
			log.Error().Err(runtimeErr).
				Object("request", &params).
				Msg("error occurred when running code")

			if errors.Is(runtimeErr, context.DeadlineExceeded) {
				responseCode = sandbox.TimeLimitExceeded
			} else {
				responseCode = sandbox.RunTimeError
			}
		}
	}

	resp, _ := json.MarshalIndent(sandbox.ExecutionResponse{
		CompileTime:    compileTime,
		CompilerOutput: compilerOutput,
		Output:         runExecution.standardOutput,
		Runtime:        runExecution.runtimeNano,
		Status:         responseCode,
	}, "", "\t")

	fmt.Printf("%s\n", resp)

	_ = os.WriteFile(fmt.Sprintf("/input/%s", "runner-out.json"), resp, os.ModePerm)
}
