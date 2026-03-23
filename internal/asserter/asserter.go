package asserter

import (
	"fmt"
	"strings"

	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/runner"
	"github.com/tidwall/gjson"
)

// Result holds the outcome of a single assertion.
type Result struct {
	Passed  bool
	Message string
}

// Evaluate runs all assertions in a block against the response.
func Evaluate(block parser.AssertionBlock, resp runner.Response) []Result {
	var results []Result
	for _, a := range block.Assertions {
		results = append(results, evalAssertion(a, resp))
	}
	return results
}

func evalAssertion(a parser.Assertion, resp runner.Response) Result {
	switch a.Kind {
	case parser.AssertStatus:
		if resp.StatusCode == a.StatusCode {
			return Result{Passed: true}
		}
		return Result{
			Passed:  false,
			Message: fmt.Sprintf("STATUS: expected %d, got %d", a.StatusCode, resp.StatusCode),
		}

	case parser.AssertHeader:
		key := strings.ToLower(a.HeaderKey)
		val, ok := resp.Headers[key]
		if !ok {
			return Result{
				Passed:  false,
				Message: fmt.Sprintf("HEADER %s: header not present", a.HeaderKey),
			}
		}
		if strings.Contains(val, a.HeaderVal) {
			return Result{Passed: true}
		}
		return Result{
			Passed:  false,
			Message: fmt.Sprintf("HEADER %s: expected to contain %q, got %q", a.HeaderKey, a.HeaderVal, val),
		}

	case parser.AssertBody:
		return evalBodyAssertion(a, resp.Body)

	default:
		return Result{Passed: false, Message: "unknown assertion kind"}
	}
}

// toGJSONPath converts a JSONPath like "$.name" or "$.tags[0]" to gjson format.
func toGJSONPath(path string) string {
	// Strip leading "$."
	if strings.HasPrefix(path, "$.") {
		path = path[2:]
	} else if path == "$" {
		path = "@this"
	}
	// Convert [N] array indexing to .N
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	return path
}

func evalBodyAssertion(a parser.Assertion, body string) Result {
	gpath := toGJSONPath(a.JSONPath)
	result := gjson.Get(body, gpath)

	label := fmt.Sprintf("BODY %s", a.JSONPath)

	switch a.Operator {
	case "exists":
		if result.Exists() {
			return Result{Passed: true}
		}
		return Result{Passed: false, Message: fmt.Sprintf("%s: expected to exist", label)}

	case "not exists":
		if !result.Exists() {
			return Result{Passed: true}
		}
		return Result{Passed: false, Message: fmt.Sprintf("%s: expected to not exist, but found %q", label, result.Raw)}

	case "==":
		actual, expected := resolveComparison(result, a.Expected)
		if actual == expected {
			return Result{Passed: true}
		}
		return Result{Passed: false, Message: fmt.Sprintf("%s: expected %s, got %q", label, a.Expected, actual)}

	case "!=":
		actual, expected := resolveComparison(result, a.Expected)
		if actual != expected {
			return Result{Passed: true}
		}
		return Result{Passed: false, Message: fmt.Sprintf("%s: expected not %s, but got %q", label, a.Expected, actual)}

	case "contains":
		expected := strings.Trim(a.Expected, `"`)
		// Try string contains
		if result.Type == gjson.String {
			if strings.Contains(result.String(), expected) {
				return Result{Passed: true}
			}
			return Result{Passed: false, Message: fmt.Sprintf("%s: expected string to contain %q, got %q", label, expected, result.String())}
		}
		// Try array contains
		if result.IsArray() {
			for _, item := range result.Array() {
				if item.String() == expected || item.Raw == a.Expected {
					return Result{Passed: true}
				}
			}
			return Result{Passed: false, Message: fmt.Sprintf("%s: expected array to contain %q", label, expected)}
		}
		return Result{Passed: false, Message: fmt.Sprintf("%s: cannot use 'contains' on non-string, non-array value", label)}

	default:
		return Result{Passed: false, Message: fmt.Sprintf("%s: unknown operator %q", label, a.Operator)}
	}
}

// resolveComparison returns (actual, expected) values to compare.
// If expected is a quoted string, compare .String(). Otherwise compare .Raw.
func resolveComparison(result gjson.Result, expected string) (string, string) {
	if strings.HasPrefix(expected, `"`) && strings.HasSuffix(expected, `"`) {
		// Quoted string: compare string values
		return result.String(), strings.Trim(expected, `"`)
	}
	// Bare literal: compare raw JSON
	return result.Raw, expected
}
