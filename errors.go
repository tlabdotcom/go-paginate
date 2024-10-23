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
	Code      int                 `json:"code"`
	Message   string              `json:"message"`
	Errors    []map[string]string `json:"errors"`
	RequestID string              `json:"request_id,omitempty"`
}

// HTTPError represents custom error types
type HTTPError struct {
	Code     int
	Message  string
	Internal error
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return e.Message
}

// NewStandardErrorResponse creates a new instance of StandardErrorResponse
func NewStandardErrorResponse(statusCode int) *StandardErrorResponse {
	return &StandardErrorResponse{
		Code:    statusCode,
		Message: getDefaultMessageForStatus(statusCode),
		Errors:  []map[string]string{},
	}
}

// getDefaultMessageForStatus returns user-friendly messages for HTTP status codes
func getDefaultMessageForStatus(code int) string {
	switch code {
	case http.StatusBadRequest:
		return "We couldn't process your request due to invalid input"
	case http.StatusUnauthorized:
		return "Please authenticate to access this resource"
	case http.StatusForbidden:
		return "You don't have permission to access this resource"
	case http.StatusNotFound:
		return "The requested resource couldn't be found"
	case http.StatusConflict:
		return "This operation conflicts with an existing resource"
	case http.StatusUnprocessableEntity:
		return "The submitted data failed validation"
	case http.StatusTooManyRequests:
		return "You've exceeded the allowed number of requests. Please try again later"
	case http.StatusInternalServerError:
		return "An unexpected error occurred. Our team has been notified"
	case http.StatusServiceUnavailable:
		return "The service is temporarily unavailable. Please try again later"
	default:
		return http.StatusText(code)
	}
}

// ResetErrors resets the errors slice in the response
func (ser *StandardErrorResponse) ResetErrors() {
	ser.Errors = []map[string]string{}
}

// AddError adds an error to the response with improved error handling
func (ser *StandardErrorResponse) AddError(err error) *StandardErrorResponse {
	switch e := err.(type) {
	case validator.ValidationErrors:
		for _, validationErr := range e {
			ser.Errors = append(ser.Errors, map[string]string{
				"field":   toSnakeCase(validationErr.Field()),
				"message": getValidationErrorMessage(validationErr),
			})
		}
	case *json.UnmarshalTypeError:
		ser.Errors = append(ser.Errors, map[string]string{
			"field":   toSnakeCase(e.Field),
			"message": fmt.Sprintf("Invalid value for %s. Expected %s", e.Field, e.Type.String()),
		})
	default:
		if dbErr := getDatabaseErrorMessage(err); dbErr != "" {
			ser.Errors = append(ser.Errors, map[string]string{
				"field":   "database",
				"message": dbErr,
			})
		} else {
			ser.Errors = append(ser.Errors, map[string]string{
				"field":   "general",
				"message": err.Error(),
			})
		}
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

// JSON sends the response with request tracking and documentation
func (ser *StandardErrorResponse) JSON(c echo.Context) error {
	// Add request tracking ID if available
	if reqID := c.Request().Header.Get("X-Request-ID"); reqID != "" {
		ser.RequestID = reqID
	}
	return c.JSON(ser.Code, ser)
}

// CustomErrorHandler handles errors globally with improved context
func CustomErrorHandler(err error, c echo.Context) {
	var statusCode int
	var message string

	switch e := err.(type) {
	case *echo.HTTPError:
		statusCode = e.Code
		message = fmt.Sprintf("%v", e.Message)
	case *HTTPError:
		statusCode = e.Code
		message = e.Message
	default:
		statusCode = http.StatusInternalServerError
		message = "An unexpected error occurred"
	}

	resp := NewStandardErrorResponse(statusCode)
	errResp := resp.AddMessageError("error", message).JSON(c)

	if errResp != nil {
		// Log the error and handle it
		c.Logger().Errorf("Failed to send JSON response: %v", errResp)

		// Check and handle the error from c.JSON
		if jsonErr := c.JSON(http.StatusInternalServerError, map[string]string{"field": "internal", "message": "Failed to send error response"}); jsonErr != nil {
			c.Logger().Errorf("Failed to send fallback JSON response: %v", jsonErr)
		}
	}
}

// getValidationErrorMessage returns human-friendly validation error messages
func getValidationErrorMessage(validationErr validator.FieldError) string {
	fieldName := humanizeFieldName(validationErr.Field())

	switch validationErr.Tag() {
	case "required":
		return fmt.Sprintf("Please provide %s", fieldName)
	case "email":
		return fmt.Sprintf("Please enter a valid email address for %s", fieldName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fieldName, validationErr.Param())
	case "max":
		return fmt.Sprintf("%s cannot be longer than %s characters", fieldName, validationErr.Param())
	case "gte":
		return fmt.Sprintf("%s must be %s or greater", fieldName, validationErr.Param())
	case "lte":
		return fmt.Sprintf("%s must be %s or less", fieldName, validationErr.Param())
	case "url":
		return fmt.Sprintf("Please enter a valid URL for %s", fieldName)
	case "datetime":
		return fmt.Sprintf("Please enter a valid date and time for %s", fieldName)
	default:
		return fmt.Sprintf("%s has an invalid value", fieldName)
	}
}

// getDatabaseErrorMessage returns user-friendly database error messages
func getDatabaseErrorMessage(err error) string {
	if errors.Is(err, sql.ErrNoRows) {
		return "We couldn't find what you're looking for"
	}
	if errors.Is(err, sql.ErrConnDone) {
		return "We're having trouble connecting to our database. Please try again"
	}

	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "unique constraint"):
		return "This information already exists in our system"
	case strings.Contains(errMsg, "foreign key constraint"):
		return "This operation references invalid or non-existent data"
	case strings.Contains(errMsg, "not-null constraint"):
		return "Required information is missing"
	case strings.Contains(errMsg, "invalid input syntax"):
		return "The provided data format is invalid"
	default:
		return ""
	}
}

// humanizeFieldName converts camelCase field names to human-readable format
func humanizeFieldName(field string) string {
	words := strings.Split(toSnakeCase(field), "_")
	for i, word := range words {
		words[i] = strings.ToLower(word)
	}
	return strings.Join(words, " ")
}

// toSnakeCase converts camelCase to snake_case
func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
