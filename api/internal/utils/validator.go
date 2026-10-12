package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

func FormatValidationError(err error) map[string]string {
	errs := make(map[string]string)

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldErr := range validationErrors {
			field := fieldErr.Field()
			label := strings.ReplaceAll(field, "_", " ")

			switch fieldErr.Tag() {
			case "required":
				errs[field] = fmt.Sprintf("%s is required", label)
			case "email":
				errs[field] = "invalid email address format"
			case "min":
				errs[field] = fmt.Sprintf("%s must be at least %s characters", label, fieldErr.Param())
			case "max":
				errs[field] = fmt.Sprintf("%s cannot exceed %s characters", label, fieldErr.Param())
			case "len":
				errs[field] = fmt.Sprintf("%s must be exactly %s characters", label, fieldErr.Param())
			case "numeric":
				errs[field] = fmt.Sprintf("%s must contain only numbers", label)
			default:
				errs[field] = fmt.Sprintf("invalid %s", label)
			}
		}
	}

	return errs
}
