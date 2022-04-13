package splunk

import (
	"os"
	"testing"
)

func makeAuthorizedTestClient(t *testing.T) *Client {
	username := os.Getenv("SPLUNK_USERNAME")
	password := os.Getenv("SPLUNK_PASSWORD")
	baseurl := os.Getenv("SPLUNK_BASEURL")
	if username == "" || password == "" || baseurl == "" {
		t.Log("Set SPLUNK_{USERNAME,PASSWORD,BASEURL}")
		t.Skip()
	}
	c := NewClient(baseurl).SetUserPass(username, password)
	if err := c.Authenticate(); err != nil {
		t.Fatalf("authentication failed: against %s %v", c.baseURL.String(), err)
	}
	return c
}

func TestAuth(t *testing.T) {
	c := makeAuthorizedTestClient(t)
	sr, err := c.SearchBlocking("search index=_internal sourcetype=splunkd | fields * | head 10", nil)
	if err != nil {
		t.Errorf("search failed: %v", err)
	}
	t.Logf("SearchResult = %#v", sr)
	{
		uc := *c
		uc.sessionKey = "invalid"
		_, err := uc.SearchBlocking("search index=_internal sourcetype=splunkd | fields * | head 10", nil)
		if err == nil {
			t.Error("unauthenticated search succeeded (but shouldn't have)")
		}
		t.Logf("error  = %+v", err)
	}
	_, err = c.SearchBlocking("gobbledigoop", nil)
	if err == nil {
		t.Error("search with invalid syntax succeeed (but shouldn't have)")
	}
}

func TestExport(t *testing.T) {
	c := makeAuthorizedTestClient(t)
	ej, err := c.SearchExport(`makeresults count=300 | eval foo="bar" | head 200`, nil)
	if err != nil {
		t.Fatalf("SearchExport: %+v", err)
	}
	var count int
	for ej.Next() {
		t.Logf("export: %#v", ej.CurrentRow)
		count++
	}
	if ej.Error != nil {
		t.Logf("export: error=%+v", ej.Error)
	}
	if count != 200 {
		t.Errorf("count = %d, expected 200", count)
	}

	if _, err := c.SearchExport(`gobbledigook`, nil); err == nil {
		t.Errorf("export: failed to report error")
	} else {
		t.Logf("export: error=%+v", err)
	}

	if _, err := c.SearchExport(`makeresults count=1 | eval gobbledigook`, nil); err == nil {
		t.Errorf("export: failed to report error")
	} else {
		t.Logf("export: error=%+v", err)
	}

	ej, err = c.SearchExport(`makeresults count=1 | eval foo=1 | table _time foo bar`, nil)
	if err != nil {
		t.Errorf("export: unexpected error %+v", err)
	} else {
		results := ej.Drain()
		t.Logf("results: %#v", results)
	}

	ej, err = c.SearchExport(`search index=* | head 1`, nil)
	if err != nil {
		t.Errorf("export: unexpected error %+v", err)
	} else if results := ej.Drain(); ej.Error != nil {
		t.Errorf("export: unexpected error %+v", ej.Error)
	} else if len(results) != 1 {
		t.Error("export: short read")
	} else {
		t.Logf("results: %#v", results)
	}
}
