package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// ListRequest encapsulates a single listing query against GCS.
type ListRequest struct {
	Bucket    string
	Prefix    string
	Delimiter string
	PageToken string
	PageSize  int
}

// Object represents the subset of metadata we care about from GCS.
type Object struct {
	Name string `json:"name"`
}

// ListResponse mirrors the payload from the JSON API.
type ListResponse struct {
	Objects      []Object
	Prefixes     []string
	NextPageToken string
}

// Client exposes the minimal listing interface used by the query service.
type Client interface {
	List(ctx context.Context, req ListRequest) (*ListResponse, error)
}

// HTTPClient implements Client by calling the public JSON API directly.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(httpClient *http.Client) *HTTPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &HTTPClient{
		baseURL:    "https://storage.googleapis.com/storage/v1",
		httpClient: httpClient,
	}
}

type apiResponse struct {
	Items         []Object `json:"items"`
	Prefixes      []string `json:"prefixes"`
	NextPageToken string   `json:"nextPageToken"`
	Error         *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// List issues a GET request to the JSON API and decodes the response.
func (c *HTTPClient) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	if req.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	values := url.Values{}
	if req.Prefix != "" {
		values.Set("prefix", req.Prefix)
	}
	if req.Delimiter != "" {
		values.Set("delimiter", req.Delimiter)
	}
	if req.PageToken != "" {
		values.Set("pageToken", req.PageToken)
	}
	if req.PageSize > 0 {
		values.Set("maxResults", strconv.Itoa(req.PageSize))
	}

	endpoint := fmt.Sprintf("%s/b/%s/o?%s", c.baseURL, req.Bucket, values.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
		return nil, fmt.Errorf("storage api error: status=%d body=%s", httpResp.StatusCode, string(body))
	}

	var payload apiResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if payload.Error != nil {
		return nil, fmt.Errorf("storage api error: code=%d msg=%s", payload.Error.Code, payload.Error.Message)
	}

	return &ListResponse{
		Objects:      payload.Items,
		Prefixes:     payload.Prefixes,
		NextPageToken: payload.NextPageToken,
	}, nil
}
