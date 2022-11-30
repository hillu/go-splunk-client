package splunk

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
)

// ExportJob is a generator for export searches
type ExportJob struct {
	Header struct {
		Preview    bool
		InitOffset int
		Messages   []struct{ Type, Text string }
		Fields     []string
	}

	// Key/value representation of a single record. Values may be
	// strings or string slices.
	CurrentRow map[string]interface{}
	// Last error
	Error error

	stream  io.ReadCloser
	decoder *json.Decoder

	done bool
}

func newExportJob(stream io.ReadCloser) (*ExportJob, error) {
	ej := &ExportJob{
		stream:  stream,
		decoder: json.NewDecoder(stream),
	}
	runtime.SetFinalizer(ej, (*ExportJob).Close)
	if t, err := ej.decoder.Token(); err != nil {
		return nil, err
	} else if t != json.Delim('{') {
		return nil, errWrongToken("[", t)
	}
	// read document by hand until ` "rows":[ `
	for {
		t, err := ej.decoder.Token()
		if err != nil {
			return nil, err
		}
		k, ok := t.(string)
		if !ok {
			return nil, errWrongToken("<string>", t)
		}
		switch k {
		case "rows":
			if t, err := ej.decoder.Token(); err != nil {
				return nil, err
			} else if t != json.Delim('[') {
				return nil, errWrongToken("[", t)
			}
			return ej, nil
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
	if !ej.decoder.More() {
		// We have reached the end of the "rows":[ ... ] array
		for _, expected := range "]}" {
			if t, err := ej.decoder.Token(); err != nil {
				return ej.setError(err)
			} else if t != json.Delim(expected) {
				return ej.setError(errWrongToken(string(expected), t))
			}
		}
		return ej.setError(nil)
	}
	var values []json.RawMessage
	if err := ej.decoder.Decode(&values); err != nil {
		return ej.setError(err)
	}
	if len(values) != len(ej.Header.Fields) {
		return ej.setError(fmt.Errorf("record length mismatch: %d, expected %d",
			len(values), len(ej.Header.Fields)))
	}
	ej.CurrentRow = make(map[string]interface{}, len(values))
	for i := 0; i < len(values); i++ {
		value, err := decodeValue(values[i])
		if err != nil {
			return ej.setError(err)
		}
		ej.CurrentRow[ej.Header.Fields[i]] = value
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
func (ej *ExportJob) Drain() (r []map[string]interface{}, err error) {
	for ej.Next() {
		r = append(r, ej.CurrentRow)
	}
	ej.Close()
	err = ej.Error
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
func (c *Client) SearchExport(ns *Namespace, query string, options *SearchOptions) (*ExportJob, error) {
	params := options.values()
	params.Set("search", query)
	params.Set("output_mode", "json_rows")
	body, err := c.doRaw("POST", ns, "search/jobs/export", params)
	if err != nil {
		return nil, fmt.Errorf("can't issue search: %v", err)
	}
	return newExportJob(body)
}
