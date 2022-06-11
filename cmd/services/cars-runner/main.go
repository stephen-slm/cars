package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"compile-and-run-sandbox/internal/sandbox"
)

func compileProject(ctx context.Context, params *sandbox.ExecutionParameters) (compileTimeNano int64, err error) {
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

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	timeAtExecution = time.Now()

	for _, step := range params.CompileSteps {
		// Create the command with our context
		command := strings.Split(step, " ")
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)

		// This time we can simply use Output() to get the result.
		output, cmdErr := cmd.CombinedOutput()

		if len(output) != 0 {
			fmt.Print(string(output))
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

func runProject(ctx context.Context, params *sandbox.ExecutionParameters) (runtimeNano int64, err error) {
	// this has to be defined here since we always want this total time
	// and the total time is determined in to defer func.
	var timeAtExecution time.Time

	defer func() {
		runtimeNano = time.Since(timeAtExecution).Nanoseconds()
	}()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(params.RunTimeoutSec)*time.Second)
	defer cancel()

	// Create the command with our context
	command := strings.Split(params.Run, " ")

	inputFile, _ := os.Open(fmt.Sprintf("/input/%s", params.StandardInput))
	defer inputFile.Close()

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = inputFile

	timeAtExecution = time.Now()
	output, cmdErr := cmd.CombinedOutput()

	if len(output) != 0 {
		fmt.Print(string(output))
	}

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = ctx.Err()
		return
	}

	err = cmdErr

	return
}

func main() {
	if _, err := os.Stat("/input/runner.json"); errors.Is(err, os.ErrNotExist) {
		log.Fatalln("runner.json configuration file does not exist and container cannot be executed.")
	}

	fileBytes, runnerFileErr := os.ReadFile("/input/runner.json")

	if runnerFileErr != nil {
		log.Fatalln("runner.json failed to be read", runnerFileErr)
	}

	var params sandbox.ExecutionParameters
	_ = json.Unmarshal(fileBytes, &params)

	responseCode := sandbox.Finished

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var runtime, compileTime int64
	var compileErr, runtimeErr error

	// configure the file for th compiled output, this is the text
	// outputted when the compiler is running.
	compileTarget, _ := os.Create(fmt.Sprintf("/input/%s", params.CompilerOut))
	defer compileTarget.Close()

	os.Stdout = compileTarget

	compileTime, compileErr = compileProject(ctx, &params)

	if compileErr != nil {
		if errors.Is(compileErr, context.DeadlineExceeded) {
			responseCode = sandbox.TimeLimitExceeded
		} else {
			responseCode = sandbox.CompilationFailed
		}
	}

	// output file for the actual execution
	executionTarget, _ := os.Create(fmt.Sprintf("/input/%s", params.StandardOut))
	defer executionTarget.Close()

	os.Stdout = executionTarget

	if responseCode == sandbox.Finished {

		runtime, runtimeErr = runProject(ctx, &params)

		if runtimeErr != nil {
			if errors.Is(runtimeErr, context.DeadlineExceeded) {
				responseCode = sandbox.TimeLimitExceeded
			} else {
				responseCode = sandbox.RunTimeError
			}
		}
	}

	fmt.Println("*-COMPILE::EOF-*", runtime, compileTime, int(responseCode))
}
