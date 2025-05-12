package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIsJson(t *testing.T) {
	tests := []struct {
		name            string
		reqContentType  string
		reqBody         string
		respContentType string
		respBody        string
		expectedResult  bool
	}{
		{
			name:            "Valid JSON in both request and response with correct content types",
			reqContentType:  "application/json",
			reqBody:         `{"key": "value"}`,
			respContentType: "application/json",
			respBody:        `{"key": "value"}`,
			expectedResult:  true,
		},
		{
			name:            "XML in request and response with correct Content-Type",
			reqContentType:  "application/xml",
			reqBody:         `<key>value</key>`,
			respContentType: "application/xml",
			respBody:        `<key>value</key>`,
			expectedResult:  false,
		},
		{
			name:            "XML in request with JSON in response",
			reqContentType:  "application/xml",
			reqBody:         `<key>value</key>`,
			respContentType: "application/json",
			respBody:        `{"key": "value"}`,
			expectedResult:  true,
		},
		{
			name:            "JSON in request with XML in response",
			reqContentType:  "application/json",
			reqBody:         `{"key": "value"}`,
			respContentType: "application/xml",
			respBody:        `<key>value</key>`,
			expectedResult:  true,
		},
		{
			name:            "Empty request and response bodies and headers",
			reqContentType:  "",
			reqBody:         "",
			respContentType: "",
			respBody:        "",
			expectedResult:  false,
		},
		{
			name:            "No content-type headers with valid JSON in request",
			reqContentType:  "",
			reqBody:         `{"key": "value"}`,
			respContentType: "",
			respBody:        ``,
			expectedResult:  true,
		},
		{
			name:            "No content-type headers with valid JSON in response",
			reqContentType:  "",
			reqBody:         ``,
			respContentType: "",
			respBody:        `{"key": "value"}`,
			expectedResult:  true,
		},
		{
			name:            "No content-type headers with invalid JSON in request",
			reqContentType:  "",
			reqBody:         `{"key": "value"`,
			respContentType: "",
			respBody:        ``,
			expectedResult:  false,
		},
		{
			name:            "No content-type headers with invalid JSON in response",
			reqContentType:  "",
			reqBody:         ``,
			respContentType: "",
			respBody:        `{"key": "value"`,
			expectedResult:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/", strings.NewReader(tt.reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if tt.reqContentType != "" {
				req.Header.Set("Content-Type", tt.reqContentType)
			}

			resp := &http.Response{
				Header: make(http.Header),
				Body:   io.NopCloser(strings.NewReader(tt.respBody)),
			}
			if tt.respContentType != "" {
				resp.Header.Set("Content-Type", tt.respContentType)
			}

			reqAndResp := httpRequestAndResponse{
				request:  req,
				response: resp,
			}

			result := isJson(&reqAndResp)
			if result != tt.expectedResult {
				t.Errorf("isJson() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
