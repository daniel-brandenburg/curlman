package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// AssertionKind identifies the type of assertion.
type AssertionKind int

const (
	AssertStatus AssertionKind = iota
	AssertHeader
	AssertBody
)

// Assertion represents a single assertion directive.
type Assertion struct {
	Kind       AssertionKind
	StatusCode int    // STATUS
	HeaderKey  string // HEADER key
	HeaderVal  string // HEADER expected value
	JSONPath   string // BODY path (e.g. "$.name")
	Operator   string // "==", "!=", "contains", "exists", "not exists"
	Expected   string // right-hand side
}

// AssertionBlock groups assertions under a named block.
type AssertionBlock struct {
	Name       string
	Assertions []Assertion
}

// ParseAssert parses a .assert file content into assertion blocks.
func ParseAssert(content string) ([]AssertionBlock, error) {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	var blocks []AssertionBlock

	if !strings.Contains(content, "###") {
		// Single unnamed block
		assertions, err := parseAssertLines("", content)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, AssertionBlock{Name: "", Assertions: assertions})
		return blocks, nil
	}

	parts := strings.Split(content, "###")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.Index(part, "\n")
		var name, body string
		if idx == -1 {
			name = strings.TrimSpace(part)
			body = ""
		} else {
			name = strings.TrimSpace(part[:idx])
			body = part[idx+1:]
		}

		assertions, err := parseAssertLines(name, body)
		if err != nil {
			return nil, fmt.Errorf("block %q: %w", name, err)
		}
		blocks = append(blocks, AssertionBlock{Name: name, Assertions: assertions})
	}

	return blocks, nil
}

func parseAssertLines(blockName, content string) ([]Assertion, error) {
	var assertions []Assertion
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		a, err := parseAssertion(line)
		if err != nil {
			return nil, fmt.Errorf("line %q: %w", line, err)
		}
		assertions = append(assertions, a)
	}
	return assertions, nil
}

func parseAssertion(line string) (Assertion, error) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 0 {
		return Assertion{}, fmt.Errorf("empty assertion")
	}

	keyword := strings.ToUpper(parts[0])
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}

	switch keyword {
	case "STATUS":
		code, err := strconv.Atoi(rest)
		if err != nil {
			return Assertion{}, fmt.Errorf("invalid status code: %q", rest)
		}
		return Assertion{Kind: AssertStatus, StatusCode: code}, nil

	case "HEADER":
		colonIdx := strings.Index(rest, ":")
		if colonIdx == -1 {
			return Assertion{}, fmt.Errorf("invalid HEADER assertion: %q", rest)
		}
		key := strings.TrimSpace(rest[:colonIdx])
		val := strings.TrimSpace(rest[colonIdx+1:])
		return Assertion{Kind: AssertHeader, HeaderKey: key, HeaderVal: val}, nil

	case "BODY":
		return parseBodyAssertion(rest)

	default:
		return Assertion{}, fmt.Errorf("unknown assertion keyword: %q", keyword)
	}
}

func parseBodyAssertion(rest string) (Assertion, error) {
	// rest = "<jsonpath> <operator> [value]"
	// Detect "not exists" before single-token operators
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return Assertion{}, fmt.Errorf("invalid BODY assertion: %q", rest)
	}

	jsonPath := parts[0]

	// Check for two-word operator "not exists"
	if len(parts) >= 3 && strings.ToLower(parts[1]) == "not" && strings.ToLower(parts[2]) == "exists" {
		return Assertion{
			Kind:     AssertBody,
			JSONPath: jsonPath,
			Operator: "not exists",
		}, nil
	}

	operator := strings.ToLower(parts[1])
	switch operator {
	case "exists":
		return Assertion{
			Kind:     AssertBody,
			JSONPath: jsonPath,
			Operator: "exists",
		}, nil
	case "==", "!=", "contains":
		if len(parts) < 3 {
			return Assertion{}, fmt.Errorf("operator %q requires a value", operator)
		}
		// Trim jsonPath then whitespace then operator from the front of rest to
		// recover the expected value without splitting on the operator literal.
		after := strings.TrimPrefix(rest, jsonPath)
		after = strings.TrimPrefix(strings.TrimSpace(after), parts[1])
		expected := strings.TrimSpace(after)
		return Assertion{
			Kind:     AssertBody,
			JSONPath: jsonPath,
			Operator: operator,
			Expected: expected,
		}, nil
	default:
		return Assertion{}, fmt.Errorf("unknown BODY operator: %q", operator)
	}
}
