package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
)

func isJson(reqAndResp *httpRequestAndResponse, maxContentLength int64) bool {
	for _, headers := range []http.Header{reqAndResp.request.Header, reqAndResp.response.Header} {
		contentTypeHeader := headers.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
		if err == nil && mediaType == "application/json" {
			return true
		}
		if strings.HasSuffix(mediaType, "+json") {
			return true
		}
	}

	if reqAndResp.request.ContentLength <= maxContentLength {
		bodyBytes, err := io.ReadAll(reqAndResp.request.Body)
		reqAndResp.request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes)))
		if err != nil {
			return false
		}
		var v interface{}
		if json.Unmarshal(bodyBytes, &v) == nil {
			return true
		}
	}

	if reqAndResp.response.ContentLength <= maxContentLength {
		bodyBytes, err := io.ReadAll(reqAndResp.response.Body)
		reqAndResp.response.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes)))
		if err != nil {
			return false
		}
		var v interface{}
		if json.Unmarshal(bodyBytes, &v) == nil {
			return true
		}
	}

	return false
}
