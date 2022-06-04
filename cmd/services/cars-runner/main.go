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
	now := time.Now()

	hasSteps := len(params.CompileSteps) > 0

	defer func() {
		if hasSteps {
			compileTimeNano = time.Since(now).Nanoseconds()
		}
	}()

	if hasSteps {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for _, step := range params.CompileSteps {
		// Create the command with our context
		command := strings.Split(step, " ")

		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stderr = os.Stderr

		// This time we can simply use Output() to get the result.
		out, cmdErr := cmd.Output()
		fmt.Println(out)

		// We want to check the context error to see if the timeout was executed.
		// The error returned by cmd.Output() will be OS specific based on what
		// happens when a process is killed.
		if ctx.Err() == context.DeadlineExceeded {
			err = ctx.Err()
			return
		}

		err = cmdErr
	}

	return
}

func runProject(ctx context.Context, params *sandbox.ExecutionParameters) (runtimeNano int64, err error) {
	now := time.Now()

	defer func() {
		runtimeNano = time.Since(now).Nanoseconds()
	}()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(params.RunTimeoutSec)*time.Second)
	defer cancel()

	for _, step := range params.RunSteps {
		// Create the command with our context
		command := strings.Split(step, " ")

		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		output, cmdErr := cmd.CombinedOutput()

		if len(output) != 0 {
			fmt.Print(string(output))
		}

		// We want to check the context error to see if the timeout was executed.
		// The error returned by cmd.Output() will be OS specific based on what
		// happens when a process is killed.
		if ctx.Err() == context.DeadlineExceeded {
			err = ctx.Err()
			return
		}

		err = cmdErr
	}

	return
}

func main() {
	if _, err := os.Stat("/input/runner.json"); errors.Is(err, os.ErrNotExist) {
		log.Fatalln("runner.json configuration file does not exist and container cannot be executed.")
	}

	fileBytes, compileError := os.ReadFile("/input/runner.json")

	if compileError != nil {
		log.Fatalln("runner.json failed to be read", compileError)
	}

	var params sandbox.ExecutionParameters
	_ = json.Unmarshal(fileBytes, &params)

	responseCode := sandbox.Finished

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	compileTime, compileError := compileProject(ctx, &params)

	if compileError != nil {
		if compileError == context.DeadlineExceeded {
			responseCode = sandbox.TimeLimitExceeded
			return
		}

		responseCode = sandbox.CompilationFailed
	}

	runTime, runTimeError := runProject(ctx, &params)

	if runTimeError != nil {
		if runTimeError == context.DeadlineExceeded {
			responseCode = sandbox.TimeLimitExceeded
			return
		}

		responseCode = sandbox.RunTimeError
	}

	fmt.Println("*-COMPILE::EOF-*", runTime, compileTime, int(responseCode))
}
