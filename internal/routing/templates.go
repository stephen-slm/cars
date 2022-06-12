package routing

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"compile-and-run-sandbox/internal/sandbox"
)

func HandleGetLanguageTemplate(w http.ResponseWriter, r *http.Request) {
	language, ok := mux.Vars(r)["lang"]

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	template, ok := sandbox.CompilerTemplate[language]

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, _ = w.Write([]byte(template))
}

type supportedLanguages struct {
	// The language code send during the compile request, this is not the same as
	// the display name. This is also the code used to get the template.
	LanguageCode string `json:"language_code"`
	// The display name the user can be shown and will understand for example
	// the display name could be C# and the code would be csharp.
	DisplayName string `json:"display_name"`
}

func HandleListLanguagesSupported(w http.ResponseWriter, _ *http.Request) {
	supported := make([]supportedLanguages, 0, len(sandbox.Compilers))

	for langCode, compiler := range sandbox.Compilers {
		supportedLang := supportedLanguages{
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

	handleJSONResponse(w, supported, http.StatusOK)
}
