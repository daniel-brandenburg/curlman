package runner

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/danielbrandenburg/curlman/internal/parser"
)

// Response holds the parsed HTTP response.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Duration   time.Duration
}

// Run executes an HTTP request using the Go standard library.
func Run(req parser.Request) (Response, error) {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return Response{}, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading response body: %w", err)
	}

	headers := map[string]string{}
	for k, vals := range resp.Header {
		headers[strings.ToLower(k)] = strings.Join(vals, ", ")
	}

	return Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       strings.TrimSpace(string(bodyBytes)),
		Duration:   duration,
	}, nil
}
