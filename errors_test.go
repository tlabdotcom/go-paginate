package goresponse

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// MockValidationError implements validator.FieldError for testing
type MockValidationError struct {
	FieldValue string
	TagValue   string
	ParamValue string
}

func (m MockValidationError) Tag() string                       { return m.TagValue }
func (m MockValidationError) ActualTag() string                 { return m.TagValue }
func (m MockValidationError) Namespace() string                 { return m.FieldValue }
func (m MockValidationError) StructNamespace() string           { return m.FieldValue }
func (m MockValidationError) Field() string                     { return m.FieldValue }
func (m MockValidationError) StructField() string               { return m.FieldValue }
func (m MockValidationError) Value() interface{}                { return nil }
func (m MockValidationError) Param() string                     { return m.ParamValue }
func (m MockValidationError) Kind() reflect.Kind                { return reflect.String }
func (m MockValidationError) Type() reflect.Type                { return reflect.TypeOf("") }
func (m MockValidationError) Error() string                     { return "" }
func (m MockValidationError) Translate(ut ut.Translator) string { return "" }

// TestNewStandardErrorResponse tests the creation of new error responses
func TestNewStandardErrorResponse(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		expectedCode    int
		expectedMessage string
	}{
		{
			name:            "BadRequest",
			statusCode:      http.StatusBadRequest,
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "We couldn't process your request due to invalid input",
		},
		{
			name:            "NotFound",
			statusCode:      http.StatusNotFound,
			expectedCode:    http.StatusNotFound,
			expectedMessage: "The requested resource couldn't be found",
		},
		{
			name:            "InternalServerError",
			statusCode:      http.StatusInternalServerError,
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "An unexpected error occurred. Our team has been notified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewStandardErrorResponse(tt.statusCode)
			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Message)
			assert.Empty(t, response.Errors)
		})
	}
}

// TestResetErrors tests the error reset functionality
func TestResetErrors(t *testing.T) {
	response := NewStandardErrorResponse(http.StatusBadRequest)
	response.Errors = append(response.Errors, map[string]string{
		"field":   "test",
		"message": "test error",
	})

	assert.NotEmpty(t, response.Errors)
	response.ResetErrors()
	assert.Empty(t, response.Errors)
}

// TestAddError tests various error types handling
func TestAddError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedField   string
		expectedMessage string
	}{
		{
			name: "ValidationError",
			err: validator.ValidationErrors{
				MockValidationError{
					FieldValue: "Email",
					TagValue:   "required",
				},
			},
			expectedField:   "email",
			expectedMessage: "Please provide email",
		},
		{
			name: "UnmarshalTypeError",
			err: &json.UnmarshalTypeError{
				Field: "Age",
				Type:  reflect.TypeOf(0),
			},
			expectedField:   "age",
			expectedMessage: "Invalid value for Age. Expected int",
		},
		{
			name:            "DatabaseNoRows",
			err:             sql.ErrNoRows,
			expectedField:   "database",
			expectedMessage: "We couldn't find what you're looking for",
		},
		{
			name:            "GeneralError",
			err:             errors.New("test error"),
			expectedField:   "general",
			expectedMessage: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewStandardErrorResponse(http.StatusBadRequest)
			response.AddError(tt.err)

			assert.NotEmpty(t, response.Errors)
			assert.Equal(t, tt.expectedField, response.Errors[0]["field"])
			assert.Equal(t, tt.expectedMessage, response.Errors[0]["message"])
		})
	}
}

// TestGetValidationErrorMessage tests validation error message generation
func TestGetValidationErrorMessage(t *testing.T) {
	tests := []struct {
		name            string
		validationErr   MockValidationError
		expectedMessage string
	}{
		{
			name: "Required",
			validationErr: MockValidationError{
				FieldValue: "Email",
				TagValue:   "required",
			},
			expectedMessage: "Please provide email",
		},
		{
			name: "Email",
			validationErr: MockValidationError{
				FieldValue: "Email",
				TagValue:   "email",
			},
			expectedMessage: "Please enter a valid email address for email",
		},
		{
			name: "MinLength",
			validationErr: MockValidationError{
				FieldValue: "Password",
				TagValue:   "min",
				ParamValue: "8",
			},
			expectedMessage: "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := getValidationErrorMessage(tt.validationErr)
			assert.Equal(t, tt.expectedMessage, message)
		})
	}
}

// TestGetDatabaseErrorMessage tests database error message generation
func TestGetDatabaseErrorResponse(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
		code            int
	}{
		{
			name:            "NoRows",
			err:             sql.ErrNoRows,
			expectedMessage: "We couldn't find what you're looking for",
			code:            http.StatusNotFound,
		},
		{
			name:            "ConnectionClosed",
			err:             sql.ErrConnDone,
			expectedMessage: "We're having trouble connecting to our database. Please try again",
			code:            http.StatusInternalServerError,
		},
		{
			name:            "UniqueConstraint",
			err:             errors.New("unique constraint violation"),
			expectedMessage: "This information already exists in our system",
			code:            http.StatusConflict,
		},
		{
			name:            "UnknownError",
			err:             errors.New("unknown error"),
			expectedMessage: "An unexpected database error occurred",
			code:            http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := getDatabaseErrorResponse(tt.err)

			// Check that the status code matches the expected code
			assert.Equal(t, tt.code, code)
			// If no errors returned, ensure expectedMessage is empty
			assert.Equal(t, tt.expectedMessage, msg)

		})
	}
}

// TestHumanizeFieldName tests field name humanization
func TestHumanizeFieldName(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "CamelCase",
			input:          "FirstName",
			expectedOutput: "first name",
		},
		{
			name:           "SingleWord",
			input:          "Name",
			expectedOutput: "name",
		},
		{
			name:           "ComplexCamelCase",
			input:          "UserEmailAddress",
			expectedOutput: "user email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizeFieldName(tt.input)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

// TestToSnakeCase tests camelCase to snake_case conversion
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "CamelCase",
			input:          "FirstName",
			expectedOutput: "first_name",
		},
		{
			name:           "SingleWord",
			input:          "Name",
			expectedOutput: "name",
		},
		{
			name:           "ComplexCamelCase",
			input:          "UserEmailAddress",
			expectedOutput: "user_email_address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

// TestCustomErrorHandler tests the global error handler
func TestCustomErrorHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tests := []struct {
		name            string
		err             error
		expectedCode    int
		expectedMessage string
	}{
		{
			name:            "HTTPError",
			err:             echo.NewHTTPError(http.StatusUnauthorized, "unauthorized access"),
			expectedCode:    http.StatusUnauthorized,
			expectedMessage: "unauthorized access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CustomErrorHandler(tt.err, c)

			var response StandardErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
			assert.Equal(t, tt.expectedMessage, response.Errors[0]["message"])
		})
	}
}

// TestJSON tests JSON response generation with request ID
func TestJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	response := NewStandardErrorResponse(http.StatusBadRequest)
	response.AddMessageError("test", "test error")
	err := response.JSON(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result StandardErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "test-request-id", result.RequestID)
}
