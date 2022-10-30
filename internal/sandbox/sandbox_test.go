//go:build e2e

package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type SandboxSuite struct {
	ctx context.Context
	suite.Suite
	manager *ContainerManager

	id        uuid.UUID
	request   Request
	container *Container
}

// CompilerTestTemplate is a mapping between the compiler template name and
// the kind of test and then the source code for that language. e.g something
// like the following:
//
// multi-functional => go => sourceCode
var CompilerTestTemplate = map[string]map[string]string{}

// mustGetCompilerTestTemplateByLanguage will return the language compiler test
// template for the given provided language or panic.
func mustGetCompilerTestTemplateByLanguage(t *testing.T, test, language string) string {
	if testFolder, ok := CompilerTestTemplate[strings.ToLower(test)]; ok {
		if template, ok := testFolder[strings.ToLower(language)]; ok {
			return template
		}
	}

	msg := fmt.Sprintf("language does not have supporting template "+
		"for test %s and language %s", test, language)

	t.Skipf(msg)
	return ""
}

var testOnce sync.Once

func LoadEmbeddedTestFiles() {
	testOnce.Do(func() {
		folders, _ := os.ReadDir("templates/tests")

		for _, folder := range folders {
			if !folder.IsDir() {
				continue
			}

			CompilerTestTemplate[folder.Name()] = map[string]string{}
			files, _ := os.ReadDir(fmt.Sprintf("templates/tests/%s", folder.Name()))

			for _, file := range files {
				path := fmt.Sprintf("templates/tests/%s/%s", folder.Name(), file.Name())
				data, err := content.ReadFile(path)

				lang := strings.Split(file.Name(), ".")[0]

				if err != nil {
					log.Warn().Str("lang", lang).
						Str("test-kind", folder.Name()).
						Err(err).
						Msg("language does not have a template")
					continue
				}

				CompilerTestTemplate[folder.Name()][lang] = string(data)
			}
		}
	})
}

func (s *SandboxSuite) SetupTest() {
	LoadEmbeddedTemplateFiles()
	LoadEmbeddedTestFiles()

	dockerClient, dockerErr := client.NewClientWithOpts(client.FromEnv)
	s.Nil(dockerErr, "docker is required")

	s.id = uuid.New()
	s.ctx = context.Background()
	s.manager = NewSandboxContainerManager(dockerClient, 10)

	s.request = Request{
		ID:               uuid.New().String(),
		ExecutionProfile: GetProfileForMachine(),
		Path:             filepath.Join(os.TempDir(), "executions", "raw", s.id.String()),
		SourceCode:       mustGetCompilerTemplateByLanguage("python"),
		Compiler:         mustGetCompilerByLanguage("python"),
		Test: &Test{
			ID:                 s.id.String(),
			StdinData:          []string{"first line", "second line"},
			ExpectedStdoutData: []string{"third line", "fourth line"},
		},
	}

	s.container = NewSandboxContainer(&s.request, s.manager.dockerClient)
	s.container.complete = make(chan string, 1)
}

// run after each test
func (s *SandboxSuite) TearDownTest() {
	defer s.container.cleanup() // nolint // test allow clean up
}

func (s *SandboxSuite) TestSimpleExecution() {
	s.Run("container should run provided code snippet to completion", func() {
		// remove the test, this is not going to perform this kind of testing
		// but validating tests will be performed in another set of tests.
		s.request.Test = nil

		go s.manager.Start(s.ctx)
		defer s.manager.Stop()

		id, complete, err := s.manager.AddContainer(s.ctx, &s.request)

		s.NoError(err)
		s.NotNil(id)

		<-complete

		result := s.manager.getContainer(id).GetResponse()

		s.Equal(Finished.String(), result.Status.String())
		s.Equal(NoTest.String(), result.TestStatus.String())
		s.Equal("Hello World!", result.Output[0])

		s.NoError(s.manager.RemoveContainer(s.ctx, id, false))
	})
}

func TestSandboxTestSuite(t *testing.T) {
	suite.Run(t, new(SandboxSuite))
}
