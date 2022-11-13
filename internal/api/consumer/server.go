package consumer

import (
	"compile-and-run-sandbox/internal/files"
	consumerv1 "compile-and-run-sandbox/internal/gen/pb/content/consumer/v1"
	"compile-and-run-sandbox/internal/queue"
	"compile-and-run-sandbox/internal/repository"
	"compile-and-run-sandbox/internal/sandbox"
	"context"
	"fmt"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"google.golang.org/protobuf/types/known/emptypb"
	"sort"
	"strings"
)

type Server struct {
	consumerv1.UnimplementedConsumerServiceServer

	FileHandler files.Files
	Repo        repository.Repository
	Translator  ut.Translator
	Validator   *validator.Validate
	Queue       queue.Queue
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
