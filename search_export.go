package splunk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// ExportJob is a generator for export searches
type ExportJob struct {
	Header struct {
		Preview    bool
		InitOffset int
		Messages   []struct{ Type, Text string }
		Fields     []string
	}

	CurrentRow map[string]string
	Error      error

	stream  io.ReadCloser
	decoder *json.Decoder

	buffered *bufio.Reader
	done     bool
}

func newExportJob(stream io.ReadCloser) (*ExportJob, error) {
	ej := ExportJob{
		stream:  stream,
		decoder: json.NewDecoder(stream),
	}
	if t, err := ej.decoder.Token(); err != nil {
		return nil, err
	} else if t != json.Delim('{') {
		return nil, fmt.Errorf("expected '{', got %v", t)
	}
	// read document by hand until ` "rows":[ `
	for {
		t, err := ej.decoder.Token()
		if err != nil {
			return nil, err
		}
		k, ok := t.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %v", t)
		}
		switch k {
		case "rows":
			if t, err := ej.decoder.Token(); err != nil {
				return nil, err
			} else if t != json.Delim('[') {
				return nil, fmt.Errorf("expected '[', got %v", t)
			}
			return &ej, nil
		case "preview":
			if err := ej.decoder.Decode(&ej.Header.Preview); err != nil {
				return nil, err
			}
		case "init_offset":
			if err := ej.decoder.Decode(&ej.Header.InitOffset); err != nil {
				return nil, err
			}
		case "messages":
			if err := ej.decoder.Decode(&ej.Header.Messages); err != nil {
				return nil, err
			}
			for _, m := range ej.Header.Messages {
				if m.Type == "FATAL" {
					return nil, fmt.Errorf("Fatal server response %s", m.Text)
				}
			}
		case "fields":
			if err := ej.decoder.Decode(&ej.Header.Fields); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unexpected key '%s'", k)
		}
	}
}

// Next attempts to read the next record and sets CurrentRow or Err.
// If an error occurs, ej is stopped.
func (ej *ExportJob) Next() bool {
	if ej.done {
		return false
	}
	var values []string
	if !ej.decoder.More() {
		if t, err := ej.decoder.Token(); err != nil {
			return ej.setError(err)
		} else if t != json.Delim(']') {
			return ej.setError(fmt.Errorf("expected ']', got %v", t))
		}
		if t, err := ej.decoder.Token(); err != nil {
			return ej.setError(err)
		} else if t != json.Delim('}') {
			return ej.setError(fmt.Errorf("expected '}', got %v", t))
		}
		return ej.setError(nil)
	}
	if err := ej.decoder.Decode(&values); err != nil {
		return ej.setError(err)
	}
	if len(values) != len(ej.Header.Fields) {
		return ej.setError(fmt.Errorf("record length mismatch: %d, expected %d",
			len(values), len(ej.Header.Fields)))
	}
	ej.CurrentRow = make(map[string]string, len(values))
	for i := 0; i < len(values); i++ {
		ej.CurrentRow[ej.Header.Fields[i]] = values[i]
	}
	return true
}

// DRY for Next method
func (ej *ExportJob) setError(err error) bool {
	ej.Error = err
	ej.Close()
	return false
}

// Drain returns all remaining records from the export search. This may block.
func (ej *ExportJob) Drain() (r []map[string]string) {
	for ej.Next() {
		r = append(r, ej.CurrentRow)
	}
	return
}

// Close stops the ExportJob.
func (ej *ExportJob) Close() {
	if !ej.done {
		ej.done = true
		ej.stream.Close()
	}
}

// SearchExport performs an "export search".
func (c *Client) SearchExport(query string, options *SearchOptions) (*ExportJob, error) {
	params := options.values()
	params.Set("search", query)
	params.Set("output_mode", "json_rows")
	body, err := c.doRaw("POST", "services/search/jobs/export", params)
	if err != nil {
		return nil, fmt.Errorf("can't issue search: %v", err)
	}
	return newExportJob(body)
}
