package splunk

import (
	"net/url"
	"strconv"
	"time"
)

// SearchOptions encapsulates options passed to search jobs.
type SearchOptions struct{ v url.Values }

// Earliest sets the earliest_time parameter
func (so *SearchOptions) Earliest(timespec string) *SearchOptions {
	return so.Set("earliest_time", timespec)
}

// Latest sets the latest_time parameter
func (so *SearchOptions) Latest(timespec string) *SearchOptions {
	return so.Set("latest_time", timespec)
}

// IndexEarliest sets the index_earliest parameter
func (so *SearchOptions) IndexEarliest(timespec string) *SearchOptions {
	return so.Set("index_earliest", timespec)
}

// IndexLatest sets the index_latest parameter
func (so *SearchOptions) IndexLatest(timespec string) *SearchOptions {
	return so.Set("index_latest", timespec)
}

// Timeout adds a timeout parameter
func (so *SearchOptions) Timeout(timeout time.Duration) *SearchOptions {
	return so.Set("timeout", strconv.Itoa(int(timeout.Seconds())))
}

// Count adds a count parameter
func (so *SearchOptions) Count(n int) *SearchOptions {
	return so.Set("count", strconv.Itoa(n))
}

// RequiredField adds an rf parameter
func (so *SearchOptions) RequiredField(fieldname string) *SearchOptions {
	return so.Add("rt", fieldname)
}

// Add adds a generic key/value parameter
func (so *SearchOptions) Add(key, value string) *SearchOptions {
	if so.v == nil {
		so.v = make(url.Values)
	}
	so.v.Add(key, value)
	return so
}

// Set sets a generic key/value parameter
func (so *SearchOptions) Set(key, value string) *SearchOptions {
	if so.v == nil {
		so.v = make(url.Values)
	}
	so.v.Set(key, value)
	return so
}

func (so *SearchOptions) values() url.Values {
	if so == nil {
		return make(url.Values)
	}
	return so.v
}
