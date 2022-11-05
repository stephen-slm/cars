package sandbox

import (
	"context"
	"encoding/json"
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

type SimpleSandboxSuite struct {
	ctx context.Context
	suite.Suite
	manager *ContainerManager

	id        uuid.UUID
	request   Request
	container *Container
}

func (s *SimpleSandboxSuite) SetupTest() {
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
		SourceCode:       mustGetCompilerTestTemplateByLanguage(s.T(), "simple", "python"),
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
func (s *SimpleSandboxSuite) TearDownTest() {
	defer s.container.cleanup() // nolint // test allow clean up
}

func (s *SimpleSandboxSuite) TestContainerPrepare() {
	s.Run("should create path directory", func() {
		s.NoError(s.container.prepare(s.ctx))

		stats, err := os.Stat(s.request.Path)

		s.NoError(err)
		s.True(stats.IsDir())
	})

	s.Run("should create source file with its contents", func() {
		s.NoError(s.container.prepare(s.ctx))

		sourceFilePath := filepath.Join(s.request.Path, s.request.Compiler.SourceFile)
		stats, err := os.Stat(sourceFilePath)

		s.NoError(err)
		s.True(!stats.IsDir())

		content, fileErr := os.ReadFile(sourceFilePath)
		s.NoError(fileErr)

		s.Equal(strings.TrimSpace(string(content)),
			strings.TrimSpace(s.request.SourceCode))
	})

	s.Run("should create input file with its contents", func() {
		s.NoError(s.container.prepare(s.ctx))

		inputFile := filepath.Join(s.request.Path, s.request.Compiler.InputFile)
		stats, err := os.Stat(inputFile)

		s.NoError(err)
		s.True(!stats.IsDir())

		content, fileErr := os.ReadFile(inputFile)
		s.NoError(fileErr)

		actual := ""

		for i, testData := range s.request.Test.StdinData {
			actual += testData

			if i != len(s.request.Test.StdinData)-1 {
				actual += "\n"
			}
		}

		s.Equal(strings.TrimSpace(string(content)), actual)
	})

	s.Run("should create input file with no contents with no test", func() {
		s.request.Test = nil

		s.NoError(s.container.prepare(s.ctx))

		inputFile := filepath.Join(s.request.Path, s.request.Compiler.InputFile)
		stats, err := os.Stat(inputFile)

		s.NoError(err)
		s.True(!stats.IsDir())

		content, fileErr := os.ReadFile(inputFile)
		s.NoError(fileErr)

		s.Equal(strings.TrimSpace(string(content)), "")
	})

	s.Run("should create runner.json file with its contents", func() {
		s.NoError(s.container.prepare(s.ctx))

		runnerFile := filepath.Join(s.request.Path, "runner.json")
		stats, err := os.Stat(runnerFile)

		s.NoError(err)
		s.True(!stats.IsDir())

		content, fileErr := os.ReadFile(runnerFile)
		s.NoError(fileErr)

		parameters := ExecutionParameters{
			ID:              s.request.ID,
			Language:        s.request.Compiler.Language,
			RunTimeout:      s.request.ExecutionProfile.CodeTimeout,
			CompileTimeout:  s.request.ExecutionProfile.CompileTimeout,
			StandardInput:   s.request.Compiler.InputFile,
			CompileSteps:    s.request.Compiler.compileSteps,
			Run:             s.request.Compiler.runSteps,
			ExecutionMemory: s.request.ExecutionProfile.ExecutionMemory,
		}

		var runner ExecutionParameters
		s.NoError(json.Unmarshal(content, &runner))

		s.Equal(runner, parameters)
	})
}

func TestSimpleSandboxTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleSandboxSuite))
}
