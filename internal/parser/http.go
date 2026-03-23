package parser

import (
	"fmt"
	"strings"
)

// Request represents a single HTTP request block.
type Request struct {
	Name    string
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// ParseHTTP parses a .http file content into a slice of Requests.
func ParseHTTP(content string) ([]Request, error) {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Split by ### separator
	var blocks []string
	var names []string

	if !strings.Contains(content, "###") {
		// Single unnamed block
		blocks = []string{content}
		names = []string{""}
	} else {
		parts := strings.Split(content, "###")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// First line is the block name
			idx := strings.Index(part, "\n")
			var name, body string
			if idx == -1 {
				name = strings.TrimSpace(part)
				body = ""
			} else {
				name = strings.TrimSpace(part[:idx])
				body = part[idx+1:]
			}
			names = append(names, name)
			blocks = append(blocks, body)
		}
	}

	var requests []Request
	for i, block := range blocks {
		req, err := parseBlock(names[i], block)
		if err != nil {
			return nil, fmt.Errorf("block %q: %w", names[i], err)
		}
		requests = append(requests, req)
	}

	return requests, nil
}

func parseBlock(name, block string) (Request, error) {
	block = strings.TrimSpace(block)
	lines := strings.Split(block, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return Request{}, fmt.Errorf("empty request block")
	}

	// First line: METHOD URL
	parts := strings.SplitN(strings.TrimSpace(lines[0]), " ", 2)
	if len(parts) != 2 {
		return Request{}, fmt.Errorf("invalid request line: %q", lines[0])
	}
	method := strings.TrimSpace(parts[0])
	url := strings.TrimSpace(parts[1])

	headers := map[string]string{}
	bodyLines := []string{}
	inBody := false

	for _, line := range lines[1:] {
		if !inBody {
			if strings.TrimSpace(line) == "" {
				inBody = true
				continue
			}
			// Parse header: Key: Value
			colonIdx := strings.Index(line, ":")
			if colonIdx == -1 {
				return Request{}, fmt.Errorf("invalid header line: %q", line)
			}
			key := strings.TrimSpace(line[:colonIdx])
			val := strings.TrimSpace(line[colonIdx+1:])
			headers[key] = val
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return Request{
		Name:    name,
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	}, nil
}
