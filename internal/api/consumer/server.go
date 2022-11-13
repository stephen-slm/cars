package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"compile-and-run-sandbox/internal/files"
	consumerv1 "compile-and-run-sandbox/internal/gen/pb/content/consumer/v1"
	"compile-and-run-sandbox/internal/queue"
	"compile-and-run-sandbox/internal/repository"
	"compile-and-run-sandbox/internal/sandbox"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	consumerv1.UnimplementedConsumerServiceServer

	FileHandler files.Files
	Repo        repository.Repository
	Translator  ut.Translator
	Validator   *validator.Validate
	Queue       queue.Queue
}

func (s Server) CompileQueueRequest(_ context.Context, direct *consumerv1.CompileRequest) (*consumerv1.CompileQueueResponse, error) {
	compiler := sandbox.Compilers[direct.Language]
	requestID := uuid.NewString()

	_ = s.FileHandler.WriteFile(&files.File{
		ID:   requestID,
		Name: compiler.SourceFile,
		Data: []byte(direct.Source),
	})

	bytes, _ := json.Marshal(queue.CompileMessage{
		ID:                 requestID,
		Language:           direct.Language,
		StdinData:          direct.StandardInData,
		ExpectedStdoutData: direct.ExpectedStandardOutData,
	})

	err := s.Queue.SubmitMessageToQueue(bytes)

	if err != nil {
		log.Error().Err(err)
		return nil, fmt.Errorf("failed to execute compile request")
	}

	dbErr := s.Repo.InsertExecution(&repository.Execution{
		ID:         requestID,
		Language:   direct.Language,
		Status:     sandbox.NotRan.String(),
		TestStatus: sandbox.TestNotRan.String(),
	})

	if dbErr != nil {
		log.Error().Err(dbErr).Msg("failed to create execution record")
		return nil, fmt.Errorf("failed to create execution record")
	}

	return &consumerv1.CompileQueueResponse{
		Id: requestID,
	}, nil
}

func (s Server) GetSupportedLanguages(_ context.Context, _ *emptypb.Empty) (*consumerv1.GetSupportedLanguagesResponse, error) {
	supported := make([]*consumerv1.SupportedLanguage, 0, len(sandbox.Compilers))

	for langCode, compiler := range sandbox.Compilers {
		supportedLang := &consumerv1.SupportedLanguage{
			LanguageCode: langCode,
			DisplayName:  compiler.Language,
		}

		if compiler.Compiler != "" && !strings.EqualFold(compiler.Compiler, langCode) {
			supportedLang.DisplayName = fmt.Sprintf("%s (%s)", compiler.Language, compiler.Compiler)
		}

		supported = append(supported, supportedLang)
	}

	sort.Slice(supported, func(i, j int) bool {
		return supported[i].DisplayName < supported[j].DisplayName
	})

	return &consumerv1.GetSupportedLanguagesResponse{
		Languages: supported,
	}, nil

}

func (s Server) GetTemplate(_ context.Context, in *consumerv1.GetTemplateRequest) (*consumerv1.GetTemplateResponse, error) {
	if template, ok := sandbox.CompilerTemplate[in.Language]; ok {
		return &consumerv1.GetTemplateResponse{
			Template: template,
		}, nil
	}

	return nil, fmt.Errorf("template for langauge `%s` does not exist", in.Language)
}

func (s Server) Ping(_ context.Context, _ *emptypb.Empty) (*consumerv1.PingResponse, error) {
	return &consumerv1.PingResponse{Message: "ping"}, nil
}
