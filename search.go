package splunk

import (
	"encoding/json"
	"fmt"
	"time"
)

// SearchResult contains the result from a Search operation.
type SearchResult struct {
	Preview    bool
	InitOffset int
	Messages   []struct{ Type, Text string }
	Fields     []struct{ Name, Type string }
	Results    []SearchResultRow
	// highlighted
}

// SearchResultRow represents a single entry of a search result set
type SearchResultRow struct {
	Header SearchResultRowHeader
	Body   map[string]string
}

// UnmarshalJSON is there to satisfy the
func (row *SearchResultRow) UnmarshalJSON(raw []byte) error {
	if err := json.Unmarshal(raw, &row.Header); err != nil {
		return err
	}
	var interim map[string]interface{}
	if err := json.Unmarshal(raw, &interim); err != nil {
		return fmt.Errorf("SearchResultBody: %v", err)
	}
	row.Body = make(map[string]string)
	for k, v := range interim {
		if len(k) > 0 && k[0] == '_' {
			continue
		}
		if vs, ok := v.(string); ok {
			row.Body[k] = vs
		}
	}
	return nil
}

// SearchResultRowHeader represents Splunk's internal fields
// associated with a search result entry.
type SearchResultRowHeader struct {
	Bkt        string    `json:"_bkt"`
	CD         string    `json:"_cd"`
	Indextime  string    `json:"_indextime"`
	KV         string    `json:"_kv"`
	Raw        string    `json:"_raw"`
	Serial     string    `json:"_serial"`
	SI         []string  `json:"_si"`
	SourceType string    `json:"_sourcetype"`
	Subsecond  string    `json:"_subsecond"`
	Time       time.Time `json:"_time"`
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
