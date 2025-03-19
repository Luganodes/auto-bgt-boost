package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cenkalti/backoff/v5"
)

type RequestRepository interface {
	Get(ctx context.Context, url string, headers map[string]string) ([]byte, error)
	Post(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error)
}

type Config struct {
	AllowedStatuses []int
	DefaultHeaders  map[string]string
}

func DefaultConfig() *Config {
	return &Config{
		AllowedStatuses: []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusAccepted,
		},
		DefaultHeaders: map[string]string{
			"Accept":       "application/json",
			"User-Agent":   "bgt_boost/1.0",
			"Content-Type": "application/json",
		},
	}
}

type requestRepository struct {
	client *http.Client
	config *Config
}

type RequestError struct {
	StatusCode int
	Body       []byte
	URL        string
	Method     string
	Err        error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("request failed: method=%s, url=%s, status=%d, error=%v, body=%s",
		e.Method, e.URL, e.StatusCode, e.Err, string(e.Body))
}

func NewRequestRepository(allowedStatuses []int) RequestRepository {
	config := DefaultConfig()
	config.AllowedStatuses = append(config.AllowedStatuses, allowedStatuses...)

	transport := http.DefaultTransport.(*http.Transport)
	transport.MaxIdleConns = 100
	return &requestRepository{
		client: &http.Client{
			Transport: transport,
		},
		config: config,
	}
}

func (rr *requestRepository) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	return rr.doRequest(ctx, http.MethodGet, url, headers, nil)
}

func (rr *requestRepository) Post(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
	}
	return rr.doRequest(ctx, http.MethodPost, url, headers, jsonBody)
}

func (rr *requestRepository) doRequest(ctx context.Context, method, url string, headers map[string]string, body []byte) ([]byte, error) {
	operation := func() ([]byte, error) {
		req, err := rr.createRequest(ctx, method, url, headers, body)
		if err != nil {
			return nil, backoff.Permanent(fmt.Errorf("failed to create request: %w", err))
		}

		resp, err := rr.client.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, err := rr.handleResponse(resp, method, url)
		if err != nil {
			return nil, err
		}
		return respBody, nil
	}

	responseBody, err := backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
	if err != nil {
		return nil, err
	}
	return responseBody, nil
}

func (rr *requestRepository) createRequest(ctx context.Context, method, url string, headers map[string]string, body []byte) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	for k, v := range rr.config.DefaultHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func (rr *requestRepository) handleResponse(resp *http.Response, method, url string) ([]byte, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if !rr.isAllowedStatus(resp.StatusCode) {
		return body, &RequestError{
			StatusCode: resp.StatusCode,
			Body:       body,
			URL:        url,
			Method:     method,
			Err:        fmt.Errorf("unexpected status code: %d", resp.StatusCode),
		}
	}
	return body, nil
}

func (rr *requestRepository) isAllowedStatus(statusCode int) bool {
	for _, code := range rr.config.AllowedStatuses {
		if statusCode == code {
			return true
		}
	}
	return false
}
