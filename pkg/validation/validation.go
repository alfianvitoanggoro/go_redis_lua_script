package validation

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

func FormatValidationError(err error) []string {
	var errs []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			tag := e.Tag()

			switch tag {
			case "required":
				errs = append(errs, fmt.Sprintf("%s is required", field))
			case "email":
				errs = append(errs, fmt.Sprintf("%s must be a valid email", field))
			case "min":
				errs = append(errs, fmt.Sprintf("%s must have minimum length %s", field, e.Param()))
			case "max":
				errs = append(errs, fmt.Sprintf("%s must have maximum length %s", field, e.Param()))
			default:
				errs = append(errs, fmt.Sprintf("%s is invalid (%s)", field, tag))
			}
		}
	}
	return errs
}
