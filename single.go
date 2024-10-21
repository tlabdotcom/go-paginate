package response

import "net/http"

// SingleDataResponse defines the structure for API responses with a single data object
type SingleDataResponse struct {
	Code    int         `json:"code"`    // HTTP status code
	Message string      `json:"message"` // Response message
	Data    interface{} `json:"data"`    // Response data
}

// GenerateSingleDataResponse creates a standard successful response
func GenerateSingleDataResponse(data interface{}, message string, statusCode int) *SingleDataResponse {
	// Set default values if necessary
	if message == "" {
		message = "Success"
	}
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return &SingleDataResponse{
		Data:    data,
		Message: message,
		Code:    statusCode,
	}
}
