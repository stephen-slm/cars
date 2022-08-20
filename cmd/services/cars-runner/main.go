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
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/memory"
	"compile-and-run-sandbox/internal/pid"
	"compile-and-run-sandbox/internal/sandbox"
)

func determineExecutionError(err error) sandbox.ContainerStatus {
	if errors.Is(err, memory.LimitExceeded) {
		return sandbox.MemoryConstraintExceeded
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return sandbox.TimeLimitExceeded
	}

	return sandbox.RunTimeError
}

func compileProject(ctx context.Context, params *sandbox.ExecutionParameters) (compilerOutput []string, compileTimeNano int64, err error) {
	log.Info().Str("id", params.ID).Msg("compile start")

	// this has to be defined here since we always want this total time
	// and the total time is determined in to defer func.
	var timeAtExecution time.Time
	compileTimeNano = 0

	hasSteps := len(params.CompileSteps) > 0

	if !hasSteps {
		return
	}

	defer func() {
		compileTimeNano = time.Since(timeAtExecution).Nanoseconds()

		log.Info().
			Str("id", params.ID).
			Int64("compile-duration-nano", compileTimeNano).
			Msg("completed compile project")
	}()

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

type RunExecution struct {
	standardOutput    []string
	errorOutput       []string
	runtimeNano       int64
	memoryConsumption memory.Memory
}

func runProject(ctx context.Context, params *sandbox.ExecutionParameters) (*RunExecution, error) {
	log.Info().Str("id", params.ID).Msg("run start")

	// this has to be defined here since we always want this total time
	// and the total time is determined in to defer func.
	var timeAtExecution time.Time
	resp := RunExecution{
		memoryConsumption: memory.Byte,
	}

	defer func() {
		resp.runtimeNano = time.Since(timeAtExecution).Nanoseconds()

		log.Info().Str("id", params.ID).
			Int64("runtime-duration-nano", resp.runtimeNano).
			Msg("run complete")
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
	pidDone := make(chan any)
	timeAtExecution = time.Now()

	cmdErr := cmd.Start()

	go func() {
		defer close(pidDone)
		_ = cmd.Wait()
	}()

	pidStats := pid.StreamPid(pidDone, cmd.Process.Pid)

	sampleLogger := log.Sample(&zerolog.BurstSampler{
		Burst:       5,
		Period:      time.Millisecond * 25,
		NextSampler: &zerolog.BasicSampler{N: 10},
	})

	for state := range pidStats {
		if state.Memory > resp.memoryConsumption {
			resp.memoryConsumption = state.Memory

			sampleLogger.Debug().
				Int("pid", cmd.Process.Pid).
				Float64("memory-mb", resp.memoryConsumption.Megabytes()).
				Float64("max-memory-mb", params.ExecutionMemory.Megabytes()).
				Msg("state-metrics")

			if resp.memoryConsumption > params.ExecutionMemory {
				log.Info().
					Int("pid", cmd.Process.Pid).
					Float64("memory-mb", resp.memoryConsumption.Megabytes()).
					Float64("max-memory-mb", params.ExecutionMemory.Megabytes()).
					Msg("state-metrics")

				_ = cmd.Process.Kill()
				return &resp, memory.LimitExceeded
			}
		}
	}

	finalProcessMaxMemory := memory.Byte
	if systemUsage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		finalProcessMaxMemory = memory.Memory(systemUsage.Maxrss * 1024)
	}

	log.Info().
		Int("pid", cmd.Process.Pid).
		Float64("pid-max-memory-mb", resp.memoryConsumption.Megabytes()).
		Float64("container-max-memory-mb", params.ExecutionMemory.Megabytes()).
		Float64("process-state-max-memory-mb", finalProcessMaxMemory.Megabytes()).
		Msg("state-metrics")

	if finalProcessMaxMemory > resp.memoryConsumption {
		resp.memoryConsumption = finalProcessMaxMemory
	}

	// last check for being over the memory limit, this can catch anything that
	// happened after the final recording but the cmd process managed to get
	// determine a higher value.
	if resp.memoryConsumption > params.ExecutionMemory {
		return &resp, memory.LimitExceeded
	}

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
		outputLinesCount++

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
		outputErrLinesCount++

		if outputErrLinesCount >= 1_000 {
			break
		}
	}

	log.Debug().
		Strs("outlines", outputLines).
		Strs("errLines", outputErrLines).
		Msg("output")

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

	log.Info().
		Interface("request", &params).
		Msg("executing incoming request")

	responseCode := sandbox.Finished

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var runExecution = &RunExecution{}
	var compilerOutput []string
	var compileTime int64
	var compileErr, runtimeErr error

	// configure the file for th compiled output, this is the text
	// outputted when the compiler is running.
	compilerOutput, compileTime, compileErr = compileProject(ctx, &params)

	if compileErr != nil {
		log.Error().Err(compileErr).Msg("error occurred when executing compile")
		responseCode = determineExecutionError(runtimeErr)
	}

	// output file for the actual execution
	if responseCode == sandbox.Finished {
		runExecution, runtimeErr = runProject(ctx, &params)

		if runtimeErr != nil {
			log.Error().Err(runtimeErr).Msg("error occurred when running code")
			responseCode = determineExecutionError(runtimeErr)
		}
	}

	executionResponse := sandbox.ExecutionResponse{
		CompileTime:        compileTime,
		CompilerOutput:     compilerOutput,
		Output:             runExecution.standardOutput,
		OutputErr:          runExecution.errorOutput,
		Runtime:            runExecution.runtimeNano,
		RuntimeMemoryBytes: runExecution.memoryConsumption.Bytes(),
		Status:             responseCode,
	}

	log.Debug().Interface("response", &executionResponse).Msg("response")
	resp, _ := json.MarshalIndent(executionResponse, "", "\t")

	_ = os.WriteFile(fmt.Sprintf("/input/%s", "runner-out.json"), resp, os.ModePerm)
}
