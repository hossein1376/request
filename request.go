package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
)

// Request includes all the necessary data for creating an HTTP request. Method can be a string; or be one of the
// predefined ones by this module. URL must be the full address with all the prefix and suffixes.
// Header, Cookies and Body are not mandatory and might be filled based on the requirements.
// Params is a map for providing URL-encoded query parameters.
type Request struct {
	Method  Method
	URL     string
	Header  http.Header
	Cookies []*http.Cookie
	Body    []byte
	Params  map[string]string
}

// Response consists of some of the HTTP response data.
type Response struct {
	Body       []byte
	Header     http.Header
	Cookies    []*http.Cookie
	StatusCode int
}

// Send sends an HTTP request based on the [Request]. It uses the provided [http.Client] in order to reuse the client.
// It returns [Response] if successful, or an error otherwise.
func Send(ctx context.Context, client *http.Client, r Request) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, bytes.NewBuffer(r.Body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header = r.Header
	for _, c := range r.Cookies {
		req.AddCookie(c)
	}

	q := req.URL.Query()
	for k, v := range r.Params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	response := &Response{
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Cookies:    resp.Cookies(),
	}
	return response, nil
}

// SendParse is intended for use cases which caller is sure about the response structure. Optionally, caller can provide
// a number of acceptable status codes. Function will return an error if the response's status code is not in them.
//
// This function requires the caller to specify the response type. Return value will be a pointer of that type, or an
// error if something goes wrong.
func SendParse[T any](ctx context.Context, client *http.Client, r Request, acceptable ...int) (*T, error) {
	resp, err := Send(ctx, client, r)
	if err != nil {
		return nil, err
	}

	if len(acceptable) != 0 && !slices.Contains(acceptable, resp.StatusCode) {
		return nil, fmt.Errorf("unacceptable status code: %d", resp.StatusCode)
	}

	t := new(T)
	err = json.Unmarshal(resp.Body, t)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return t, nil
}
