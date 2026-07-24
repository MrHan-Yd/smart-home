package hass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type State struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged string         `json:"last_changed"`
	LastUpdated string         `json:"last_updated"`
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != "" && c.token != ""
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("hass not configured")
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/api/", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("hass status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) GetStates(ctx context.Context) ([]State, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/states", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("hass states %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out []State
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Attributes == nil {
			out[i].Attributes = map[string]any{}
		}
	}
	return out, nil
}

func (c *Client) GetState(ctx context.Context, entityID string) (*State, error) {
	if entityID == "" {
		return nil, fmt.Errorf("entity_id required")
	}
	resp, err := c.do(ctx, http.MethodGet, "/api/states/"+entityID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrEntityNotFound
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("hass state %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var st State
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return nil, err
	}
	if st.Attributes == nil {
		st.Attributes = map[string]any{}
	}
	return &st, nil
}

// CallService posts to /api/services/{domain}/{service}
func (c *Client) CallService(ctx context.Context, domain, service string, data map[string]any) error {
	if domain == "" || service == "" {
		return fmt.Errorf("domain and service required")
	}
	if data == nil {
		data = map[string]any{}
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/services/"+domain+"/"+service, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("hass service %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

var ErrEntityNotFound = fmt.Errorf("entity not found")
