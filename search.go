package splunk

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SearchResultHeader is a representation of the header returned in
// search_operations (output_mode=json)
type SearchResultHeader struct {
	Preview    bool
	InitOffset int `json:"init_offset"`
	Messages   []struct{ Type, Text string }
	Fields     []struct{ Name, Type string }
}

// SearchResult is the result type returned by SearchBlocking
type SearchResult struct {
	SearchResultHeader
	Body []map[string]interface{}
}

func errWrongToken(expected string, got json.Token) error {
	return fmt.Errorf("expected %s - got %v", expected, got)
}

// decodeValue decodes a JSON-encoded field value into either string
// or []string.
func decodeValue(raw json.RawMessage) (interface{}, error) {
	var (
		elem interface{}
		err  error
	)
	if len(raw) > 0 && raw[0] == '[' {
		var v []string
		err = json.Unmarshal(raw, &v)
		elem = v
	} else {
		var v string
		err = json.Unmarshal(raw, &v)
		elem = v
	}
	if err != nil {
		return nil, fmt.Errorf("Can't deserialize as value: %s", string(raw))
	}
	return elem, nil
}

// UnmarshalJSON decodes search result header + body (fields)
func (rs *SearchResult) UnmarshalJSON(raw []byte) error {
	var proto struct {
		SearchResultHeader
		Results json.RawMessage
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&proto); err != nil {
		return err
	}
	rs.SearchResultHeader = proto.SearchResultHeader

	dec = json.NewDecoder(bytes.NewReader(proto.Results))
	if t, err := dec.Token(); err != nil {
		return err
	} else if t != json.Delim('[') {
		return errWrongToken("[", t)
	}

	for dec.More() {
		if t, err := dec.Token(); err != nil {
			return err
		} else if t != json.Delim('{') {
			return errWrongToken("{", t)
		}
		row := make(map[string]interface{})
		for dec.More() {
			var key string
			if t, err := dec.Token(); err != nil {
				return err
			} else if s, ok := t.(string); !ok {
				return errWrongToken("<string>", t)
			} else {
				key = s
			}
			var rawValue json.RawMessage
			if err := dec.Decode(&rawValue); err != nil {
				return err
			}
			value, err := decodeValue(rawValue)
			if err != nil {
				return err
			}
			row[key] = value
		}
		if t, err := dec.Token(); err != nil {
			return err
		} else if t != json.Delim('}') {
			return errWrongToken("}", t)
		}
		rs.Body = append(rs.Body, row)
	}

	if t, err := dec.Token(); err != nil {
		return err
	} else if t != json.Delim(']') {
		return errWrongToken("]", t)
	}

	return nil
}

// SearchBlocking executes a blocking (exec_mode="oneshot") search
func (c *Client) SearchBlocking(query string, options *SearchOptions) (*SearchResult, error) {
	params := options.values()
	params.Set("exec_mode", "oneshot")
	params.Set("search", query)
	jd, err := c.Post("services/search/jobs", params)
	if err != nil {
		return nil, fmt.Errorf("can't issue search: %v", err)
	}
	var results SearchResult
	if err := json.Unmarshal(jd, &results); err != nil {
		return nil, fmt.Errorf("can't unmarshal search result: %s %v", string(jd), err)
	}
	return &results, nil
}
