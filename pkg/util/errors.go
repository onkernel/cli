package util

import (
	"encoding/json"
	"fmt"

	"github.com/onkernel/kernel-go-sdk"
)

type CleanedUpSdkError struct {
	Err     error
	Message string
}

var _ error = &CleanedUpSdkError{}

func (e *CleanedUpSdkError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if _, ok := e.Err.(*kernel.Error); ok {
		// assume we send back JSON with an "error" field, and just show that
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(e.Err.(*kernel.Error).RawJSON()), &m); err != nil {
			return e.Err.Error()
		}
		if v, ok := m["error"]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return e.Err.Error()
}

func (e *CleanedUpSdkError) Unwrap() error {
	return e.Err
}
