package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
)

func isJson(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && mediaType == "application/json" {
		return true
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes)))
	var v map[string]interface{}
	return json.Unmarshal(bodyBytes, &v) == nil
}
