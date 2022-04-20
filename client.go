package splunk

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client for the Splunk REST API
type Client struct {
	httpClient *http.Client
	baseURL    url.URL
	username   string
	password   string
	token      string
	sessionKey string
}

// NewClient creates a new REST API client using url as the API base URL
func NewClient(baseurl string) *Client {
	u, err := url.Parse(baseurl)
	if err != nil {
		u, _ = url.Parse("http://localhost:8089")
	}
	return &Client{baseURL: *u, httpClient: http.DefaultClient}
}

func (c *Client) client() *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}
	return http.DefaultClient
}

// SetHTTPClient sets HTTP client to be used for requests (default: http
func (c *Client) SetHTTPClient(hc *http.Client) *Client {
	c.httpClient = hc
	return c
}

// SetUserPass configures Client to use a username and password for authorization.
func (c *Client) SetUserPass(username, password string) *Client {
	c.username, c.password, c.token = username, password, ""
	return c
}

// SetToken configures Client to use a token for authorization.
func (c *Client) SetToken(token string) *Client {
	c.username, c.password, c.token = "", "", token
	return c
}

// Authenticate performs an authentication using the configuration mehtods
func (c *Client) Authenticate() error {
	values := make(url.Values)
	header := make(http.Header)
	if c.username != "" && c.password != "" {
		values.Add("username", c.username)
		values.Add("password", c.password)
	} else if c.token != "" {
		header.Add("Authorization", "Bearer "+c.token)
	} else {
		return errors.New("no authentication method configured")
	}

	jm, err := c.Do("POST", nil, "auth/login", values)
	if err != nil {
		return err
	}
	info := make(map[string]string)
	if err := json.Unmarshal(jm, &info); err != nil {
		return errors.New("can't deocde response")
	}
	if key, ok := info["sessionKey"]; ok {
		c.sessionKey = key
	} else {
		return errors.New("no session key")
	}
	return nil
}

// Deauthenticate discards of the session key
func (c *Client) Deauthenticate() {
	// TODO: remove session on Splunk server
	c.sessionKey = ""
}

func (c *Client) doRaw(method string, ns *Namespace, path string, params url.Values) (io.ReadCloser, error) {
	u := c.baseURL
	u.Path = ns.String() + path
	req, _ := http.NewRequest(method, u.String(), strings.NewReader(params.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if c.sessionKey != "" {
		req.Header.Add("Authorization", "Splunk "+c.sessionKey)
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %+v failed: %v", req, err)
	}
	if resp.StatusCode >= 400 {
		dec := json.NewDecoder(resp.Body)
		defer resp.Body.Close()
		err := APIError{StatusCode: resp.StatusCode}
		dec.Decode(&err)
		return nil, err
	}
	return resp.Body, nil
}

// Do performs a request
func (c *Client) Do(method string, ns *Namespace, path string, params url.Values) (json.RawMessage, error) {
	params.Set("output_mode", "json")
	body, err := c.doRaw(method, ns, path, params)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(body)
	defer body.Close()
	var info json.RawMessage
	if err := dec.Decode(&info); err != nil {
		return nil, errors.New("can't deocde response")
	}
	return info, nil
}

// Get performs an authenticated GET request.
func (c *Client) Get(ns *Namespace, path string, params url.Values) (json.RawMessage, error) {
	return c.Do("GET", ns, path, params)
}

// Post performs an authenticated POST request.
func (c *Client) Post(ns *Namespace, path string, params url.Values) (json.RawMessage, error) {
	return c.Do("POST", ns, path, params)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(ns *Namespace, path string, params url.Values) (json.RawMessage, error) {
	return c.Do("DELETE", ns, path, params)
}

// Namespace contains optional user and app.
//
// The string representataion is "servicesNS/USER/APP/" or "services/"
// (for nil).
//
// Empty strings User or App are translated to "-".
type Namespace struct{ User, App string }

func (ns *Namespace) String() string {
	if ns == nil {
		return "services/"
	}
	user, app := ns.User, ns.App
	if user == "" {
		user = "-"
	}
	if app == "" {
		app = "-"
	}
	return "servicesNS/" + user + "/" + app + "/"
}
