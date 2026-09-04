package templates

// ControllerUtilsTemplate renders the response formatters and validation logic.
const ControllerUtilsTemplate = `package controller

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

// ErrorDetail represents a single validation issue
type ErrorDetail struct {
	Field string ` + "`json:\"field\"`" + `
	Issue string ` + "`json:\"issue\"`" + `
}

// ValidateStruct validates a struct using go-playground/validator
func ValidateStruct(m interface{}) []*ErrorDetail {
	var errors []*ErrorDetail
	err := validate.Struct(m)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorDetail
			element.Field = err.Field()
			element.Issue = err.Tag()
			errors = append(errors, &element)
		}
	}
	return errors
}

// SendError sends a standardized error response
func SendError(c *fiber.Ctx, status int, code string, message string, details interface{}) error {
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// SendSuccess sends a standardized success response
func SendSuccess(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(fiber.Map{
		"success": true,
		"data":    data,
	})
}
`
