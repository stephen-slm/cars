package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SimpleSandboxSuite struct {
	ctx context.Context
	suite.Suite
	manager *ContainerManager
	request Request
}

func (suite *SimpleSandboxSuite) SetupTest() {
	dockerClient, dockerErr := client.NewClientWithOpts(client.FromEnv)
	suite.Nil(dockerErr, "docker is required")

	suite.ctx = context.Background()
	suite.manager = NewSandboxContainerManager(dockerClient, 10)
	suite.request = Request{
		ID:               uuid.New().String(),
		ExecutionProfile: sandbox.GetProfileForMachine(),
		Path:             filepath.Join(os.TempDir(), "executions", "raw", compileMsg.ID),
		SourceCode:       string(sourceCode),
		Compiler:         compiler,
		Test:             nil,
	}
}

func (suite *SimpleSandboxSuite) TestContainerPrepare() {
	container := NewSandboxContainer(&suite.request, suite.manager.dockerClient)
	container.complete = make(chan string, 1)

	defer container.cleanup()

	require.NotNil(suite.T(), container.prepare(suite.ctx))

	suite.Run("should create path directory", func() {
		stats, err := os.Stat(suite.request.Path)

		require.NotNil(suite.T(), err)
		require.True(suite.T(), stats.IsDir())
	})

	suite.Run("should create source file with its contents", func() {

	})

	suite.Run("should create input file with its contents", func() {

	})

	suite.Run("should create runner.json file with its contents", func() {

	})
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleSandboxSuite))
}
