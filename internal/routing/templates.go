package routing

import (
	"net/http"

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
