package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
)

func isJson(req_and_resp *httpRequestAndResponse) bool {
	for _, headers := range []http.Header{req_and_resp.request.Header, req_and_resp.response.Header} {
		mediaType, _, err := mime.ParseMediaType(headers.Get("Content-Type"))
		if err == nil && mediaType == "application/json" {
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
