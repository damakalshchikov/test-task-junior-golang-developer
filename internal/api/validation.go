package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
)

var validate = newValidator()

type bodyKey[T any] struct{}

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})

	v.RegisterCustomTypeFunc(func(field reflect.Value) any {
		if value, ok := field.Interface().(models.MonthYear); ok {
			return value.Time
		}
		return nil
	}, models.MonthYear{})

	return v
}

func validateBody[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body T

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if err := validate.Struct(body); err != nil {
			writeError(w, http.StatusBadRequest, validationMessage(err))
			return
		}

		ctx := context.WithValue(r.Context(), bodyKey[T]{}, body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bodyFrom[T any](ctx context.Context) T {
	body, _ := ctx.Value(bodyKey[T]{}).(T)
	return body
}

func validationMessage(err error) string {
	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return "invalid request body"
	}

	messages := make([]string, 0, len(fieldErrors))
	for _, fieldError := range fieldErrors {
		messages = append(messages, fieldMessage(fieldError))
	}

	return strings.Join(messages, "; ")
}

func fieldMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()

	switch fieldError.Tag() {
	case "required":
		return field + " is required"
	case "min":
		return field + " must be greater than or equal to " + fieldError.Param()
	case "gtefield":
		return field + " must not be before start_date"
	default:
		return field + " is invalid"
	}
}
