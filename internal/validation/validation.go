package validation

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

func TranslateError(err error, trans ut.Translator) (errs []string) {
	if err == nil {
		return nil
	}

	validationErrors := validator.ValidationErrors{}

	if errors.As(err, &validationErrors) {
		for _, e := range validationErrors {
			translatedErr := e.Translate(trans)
			errs = append(errs, translatedErr)
		}
	}

	return errs
}
