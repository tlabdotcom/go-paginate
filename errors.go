package goresponse

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// StandardErrorResponse defines the structure of the error response
type StandardErrorResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Errors  []map[string]string `json:"errors"`
}

// NewStandardErrorResponse creates a new instance of StandardErrorResponse with the provided status code
func NewStandardErrorResponse(statusCode int) *StandardErrorResponse {
	return &StandardErrorResponse{
		Code:    statusCode,
		Message: http.StatusText(statusCode),
		Errors:  []map[string]string{},
	}
}

// ResetErrors resets the errors slice in the response
func (ser *StandardErrorResponse) ResetErrors() {
	ser.Errors = []map[string]string{}
}

// AddError adds an error to the response, handles validation and database errors
func (ser *StandardErrorResponse) AddError(err error) *StandardErrorResponse {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, validationErr := range validationErrors {
			message := getValidationErrorMessage(validationErr)
			ser.Errors = append(ser.Errors, map[string]string{
				"field":   validationErr.Field(),
				"message": message,
			})
		}
	} else if unmarshalErr, ok := err.(*json.UnmarshalTypeError); ok {
		message := fmt.Sprintf("Unmarshal type error: expected %s but got %s", unmarshalErr.Type.String(), unmarshalErr.Value)
		ser.Errors = append(ser.Errors, map[string]string{
			"field":   unmarshalErr.Field,
			"message": message,
		})
	} else if dbErr := checkDatabaseError(err); dbErr != "" {
		ser.Errors = append(ser.Errors, map[string]string{
			"field":   "-",
			"message": dbErr,
		})
	} else {
		ser.Errors = append(ser.Errors, map[string]string{
			"field":   "-",
			"message": err.Error(),
		})
	}
	return ser
}

// AddMessageError adds a manual error to the response
func (ser *StandardErrorResponse) AddMessageError(field string, message string) *StandardErrorResponse {
	ser.Errors = append(ser.Errors, map[string]string{
		"field":   field,
		"message": message,
	})
	return ser
}

// JSON sends the response as JSON with appropriate status code and message
func (ser *StandardErrorResponse) JSON(c echo.Context) error {
	if ser.Code == 422 {
		ser.Message = "Validation Failed"
	} else {
		ser.Message = http.StatusText(ser.Code)
	}
	return c.JSON(ser.Code, ser)
}

// CustomErrorHandler is a custom error handler for handling errors globally
func CustomErrorHandler(err error, c echo.Context) {
	newErr := NewStandardErrorResponse(http.StatusInternalServerError)
	newErr.ResetErrors()

	if httpErr, ok := err.(*echo.HTTPError); ok {
		newErr.Code = httpErr.Code
		if httpErr.Code == http.StatusUnauthorized {
			newErr.AddMessageError("authorization", "Unauthorized access").JSON(c)
		} else if httpErr.Code == http.StatusBadRequest {
			newErr.AddMessageError("bad_request", httpErr.Message.(string)).JSON(c)
		} else {
			newErr.AddMessageError("internal", httpErr.Message.(string)).JSON(c)
		}
	} else {
		newErr.AddMessageError("internal", err.Error()).JSON(c)
	}
}

// getValidationErrorMessage returns a user-friendly message for a validation error
func getValidationErrorMessage(validationErr validator.FieldError) string {
	switch validationErr.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", validationErr.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", validationErr.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", validationErr.Field(), validationErr.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", validationErr.Field(), validationErr.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", validationErr.Field(), validationErr.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", validationErr.Field(), validationErr.Param())
	default:
		return fmt.Sprintf("%s is not valid", validationErr.Field())
	}
}

// checkDatabaseError returns a user-friendly message for database errors
func checkDatabaseError(err error) string {
	if errors.Is(err, sql.ErrNoRows) {
		return "No matching record found"
	}
	if errors.Is(err, sql.ErrConnDone) {
		return "Database connection was closed"
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "unique constraint") {
		return "A record with the same value already exists"
	}
	if strings.Contains(errMsg, "foreign key constraint") {
		return "A foreign key constraint violation"
	}
	if strings.Contains(errMsg, "not-null constraint") {
		return "A required field is missing"
	}
	if strings.Contains(errMsg, "invalid input syntax for") {
		return "Invalid input syntax"
	}
	return ""
}
