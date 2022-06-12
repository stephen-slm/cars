package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

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

func runProject(ctx context.Context, params *sandbox.ExecutionParameters) (runOutput []string, runtimeNano int64, err error) {
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
		// trim the last new line if any to correctly allow testing of output
		trimmedOutput := strings.TrimSuffix(string(output), "\n")
		runOutput = strings.Split(trimmedOutput, "\n")
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

	var compilerOutput, runOutput []string
	var runtime, compileTime int64
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

		runOutput, runtime, runtimeErr = runProject(ctx, &params)

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

	resp, _ := json.Marshal(sandbox.ExecutionResponse{
		Runtime:        runtime,
		CompileTime:    compileTime,
		Output:         runOutput,
		CompilerOutput: compilerOutput,
		Status:         responseCode,
	})

	_ = os.WriteFile(fmt.Sprintf("/input/%s", "runner-out.json"), resp, os.ModePerm)
}
