//go:build e2e

// Test suite designed to target every single language and every single
// supported implementation for that language in a way that will continue to
// allow additional new languages in the future.

package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"compile-and-run-sandbox/internal/memory"
)

var manager *ContainerManager
var manOnce sync.Once

func setup(t *testing.T, ctx context.Context) *ContainerManager {
	LoadEmbeddedTestFiles()

	manOnce.Do(func() {
		dockerClient, dockerErr := client.NewClientWithOpts(client.FromEnv)
		assert.Nil(t, dockerErr, "docker is required")

		manager = NewSandboxContainerManager(dockerClient, 10)
		go manager.Start(ctx)
	})

	return manager
}

func TestCompleteBasicExecution(t *testing.T) {
	manager := setup(t, context.Background())

	for languageName, compiler := range Compilers {
		t.Run(fmt.Sprintf("basic execution check for %s", languageName), func(t *testing.T) {
			languageName := languageName
			compiler := compiler

			t.Parallel()

			pathingUUID := uuid.New()

			request := Request{
				ID:               uuid.New().String(),
				ExecutionProfile: GetProfileForMachine(),
				Path:             filepath.Join(os.TempDir(), "executions", "raw", pathingUUID.String()),

				// template code that is given to the clients are the most basic
				// `should` work implementations and thus are used here in this
				// base example.
				SourceCode: mustGetCompilerTestTemplateByLanguage(t, "simple", languageName),
				Compiler:   compiler,

				// No tests are checked in this example.
				Test: nil,
			}

			id, complete, err := manager.AddContainer(context.Background(), &request)

			assert.NoError(t, err)
			assert.NotNil(t, id)

			<-complete

			result := manager.getContainer(id).GetResponse()

			// Verify that it finished and we had no tests.
			assert.Equal(t, Finished.String(), result.Status.String(),
				"output: %s\noutput-error: %s\ncompiler: %s",
				strings.Join(result.Output, "\n"),
				strings.Join(result.OutputError, "\n"),
				strings.Join(result.CompilerOutput, "\n"),
			)

			assert.Equal(t, NoTest.String(), result.TestStatus.String())

			assert.True(t, len(result.Output) > 0)
			assert.Equal(t, "Hello, World!", result.Output[0])

			assert.NoError(t, manager.RemoveContainer(context.Background(), id, false))
		})
	}
}

func TestCompleteMultiFunctionExecution(t *testing.T) {
	manager := setup(t, context.Background())

	for languageName, compiler := range Compilers {
		t.Run(fmt.Sprintf("multi-functional execution check for %s", languageName), func(t *testing.T) {
			languageName := languageName
			compiler := compiler

			t.Parallel()

			pathingUUID := uuid.New()

			request := Request{
				ID:               uuid.New().String(),
				ExecutionProfile: GetProfileForMachine(),
				Path:             filepath.Join(os.TempDir(), "executions", "raw", pathingUUID.String()),

				// Pull the source code from a pre-generated list of possible
				// values for complex implementations. Ensuring that we wrap
				// the correct implementations if needed.
				SourceCode: mustGetCompilerTestTemplateByLanguage(t, "multi", languageName),
				Compiler:   compiler,

				// No tests are checked in this example.
				Test: nil,
			}

			id, complete, err := manager.AddContainer(context.Background(), &request)

			assert.NoError(t, err)
			assert.NotNil(t, id)

			<-complete

			result := manager.getContainer(id).GetResponse()

			// Verify that it finished and we had no tests.
			assert.Equal(t, Finished.String(), result.Status.String(),
				"output: %s\noutput-error: %s\ncompiler: %s",
				strings.Join(result.Output, "\n"),
				strings.Join(result.OutputError, "\n"),
				strings.Join(result.CompilerOutput, "\n"),
			)

			assert.Equal(t, NoTest.String(), result.TestStatus.String())

			// assert.ould lean on a more complete output but this should do for now.
			assert.True(t, len(result.Output) > 0)
			assert.Equal(t, "Hello, World!", strings.Join(result.Output, "\n"))

			assert.NoError(t, manager.RemoveContainer(context.Background(), id, false))
		})
	}
}

func TestCompleteTimeBoundExecution(t *testing.T) {
	manager := setup(t, context.Background())

	for languageName, compiler := range Compilers {
		t.Run(fmt.Sprintf("time-bound execution check for %s", languageName), func(t *testing.T) {
			languageName := languageName
			compiler := compiler

			t.Parallel()

			pathingUUID := uuid.New()

			request := Request{
				ID: uuid.New().String(),
				ExecutionProfile: &Profile{
					AutoRemove:      true,
					CodeTimeout:     time.Millisecond * 100,
					CompileTimeout:  time.Second * 5,
					ContainerMemory: memory.Gigabyte,
					ExecutionMemory: memory.Gigabyte,
				},

				Path: filepath.Join(os.TempDir(), "executions", "raw", pathingUUID.String()),

				// Pull the source code from a pre-generated list of possible
				// values for complex implementations. Ensuring that we wrap
				// the correct implementations if needed.
				SourceCode: mustGetCompilerTestTemplateByLanguage(t, "time", languageName),
				Compiler:   compiler,

				// No tests are checked in this example.
				Test: nil,
			}

			id, complete, err := manager.AddContainer(context.Background(), &request)

			assert.NoError(t, err)
			assert.NotNil(t, id)

			<-complete

			result := manager.getContainer(id).GetResponse()

			// Verify that it finished and we had no tests.
			assert.Equal(t, TimeLimitExceeded.String(), result.Status.String(),
				"output: %s\noutput-error: %s\ncompiler: %s",
				strings.Join(result.Output, "\n"),
				strings.Join(result.OutputError, "\n"),
				strings.Join(result.CompilerOutput, "\n"),
			)

			assert.NoError(t, manager.RemoveContainer(context.Background(), id, false))
		})
	}
}

func TestCompleteMemoryBoundExecution(t *testing.T) {
	manager := setup(t, context.Background())

	for languageName, compiler := range Compilers {
		t.Run(fmt.Sprintf("memory execution check for %s", languageName), func(t *testing.T) {
			languageName := languageName
			compiler := compiler

			t.Parallel()

			pathingUUID := uuid.New()

			request := Request{
				ID: uuid.New().String(),
				ExecutionProfile: &Profile{
					AutoRemove:      true,
					CodeTimeout:     time.Second,
					CompileTimeout:  time.Second * 5,
					ContainerMemory: memory.Gigabyte,
					ExecutionMemory: memory.Megabyte * 50,
				},

				Path: filepath.Join(os.TempDir(), "executions", "raw", pathingUUID.String()),

				// Pull the source code from a pre-generated list of possible
				// values for complex implementations. Ensuring that we wrap
				// the correct implementations if needed.
				SourceCode: mustGetCompilerTestTemplateByLanguage(t, "memory", languageName),
				Compiler:   compiler,

				// No tests are checked in this example.
				Test: nil,
			}

			id, complete, err := manager.AddContainer(context.Background(), &request)

			assert.NoError(t, err)
			assert.NotNil(t, id)

			<-complete

			result := manager.getContainer(id).GetResponse()

			// Verify that it finished and we had no tests.
			assert.Equal(t, MemoryConstraintExceeded.String(), result.Status.String(),
				"output: %s\noutput-error: %s\ncompiler: %s",
				strings.Join(result.Output, "\n"),
				strings.Join(result.OutputError, "\n"),
				strings.Join(result.CompilerOutput, "\n"),
			)

			assert.NoError(t, manager.RemoveContainer(context.Background(), id, false))
		})
	}
}
