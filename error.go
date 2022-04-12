package splunk

import (
	"errors"
	"fmt"
)

// APIError encapsulates error messages and status code received with
// a Splunk REST call
//
// Examples from output_mode=json calls:
//
// "messages": [
//   {
//     "type": "WARN",
//     "text": "call not properly authenticated"
//   }
// ]
//
// "messages": [
//   {
//     "type": "FATAL",
//     "text": "Unknown search command 'asjkdfl'."
//   }
// ]
type APIError struct {
	// HTTP Status Code
	StatusCode int
	Messages   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"messages"`
}

func (e APIError) Error() (s string) {
	for _, m := range e.Messages {
		if s != "" {
			s += "; "
		}
		s += m.Text
	}
	if e.StatusCode != 0 {
		s += fmt.Sprintf("; StatusCode %d", e.StatusCode)
	}
	return
}

var errNotImpl = errors.New("not implemented")
