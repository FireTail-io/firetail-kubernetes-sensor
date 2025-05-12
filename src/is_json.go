package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
)

func isJson(req_and_resp *httpRequestAndResponse) bool {
	for _, headers := range []http.Header{req_and_resp.request.Header, req_and_resp.response.Header} {
		contentTypeHeader := headers.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
		if err == nil && mediaType == "application/json" {
			return true
		}
		if strings.HasSuffix(mediaType, "+json") {
			return true
		}
	}

	bodyBytes, err := io.ReadAll(req_and_resp.request.Body)
	req_and_resp.request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes)))
	if err != nil {
		return false
	}
	var v map[string]interface{}
	if json.Unmarshal(bodyBytes, &v) == nil {
		return true
	}

	bodyBytes, err = io.ReadAll(req_and_resp.response.Body)
	req_and_resp.response.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes)))
	if err != nil {
		return false
	}
	return json.Unmarshal(bodyBytes, &v) == nil
}
