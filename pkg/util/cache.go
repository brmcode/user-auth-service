package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GenerateCacheKey generates a cache key based on the input parameters
func GenerateCacheKey(prefix string, params any) string {
	return fmt.Sprintf("%s:%v", prefix, params)
}

// GenerateCacheParams generates a cache params based on the input parameters
func GenerateCacheKeyParams(params ...any) string {
	strs := make([]string, len(params))
	for i, param := range params {
		strs[i] = fmt.Sprintf("%v", param)
	}

	return strings.Join(strs, "-")
}

// Serialize marshals the input data into an array of bytes
func Serialize(data any) ([]byte, error) {
	return json.Marshal(data)
}

// Deserialize unmarshals the input data into the output interface
func Deserialize(data []byte, output any) error {
	return json.Unmarshal(data, output)
}
