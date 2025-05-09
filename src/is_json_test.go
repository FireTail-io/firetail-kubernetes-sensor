package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestIsJson(t *testing.T) {
	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedResult bool
	}{
		{
			name:           "Valid JSON with correct Content-Type",
			contentType:    "application/json",
			body:           `{"key": "value"}`,
			expectedResult: true,
		},
		{
			name:           "Invalid JSON with correct Content-Type",
			contentType:    "application/json",
			body:           `{"key": "value"`,
			expectedResult: true,
		},
		{
			name:           "Valid JSON with incorrect Content-Type",
			contentType:    "text/plain",
			body:           `{"key": "value"}`,
			expectedResult: true,
		},
		{
			name:           "Empty body with correct Content-Type",
			contentType:    "application/json",
			body:           ``,
			expectedResult: true,
		},
		{
			name:           "No Content-Type header",
			contentType:    "",
			body:           `{"key": "value"}`,
			expectedResult: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			result := isJson(req)
			if result != tt.expectedResult {
				t.Errorf("isJson() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
