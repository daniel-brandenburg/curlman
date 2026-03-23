package asserter

import (
	"testing"

	"github.com/danielbrandenburg/curlman/internal/parser"
	"github.com/danielbrandenburg/curlman/internal/runner"
)

// resp builds a minimal runner.Response.
func resp(status int, body string, headers map[string]string) runner.Response {
	if headers == nil {
		headers = map[string]string{}
	}
	return runner.Response{StatusCode: status, Body: body, Headers: headers}
}

// block builds a single-assertion block.
func block(a parser.Assertion) parser.AssertionBlock {
	return parser.AssertionBlock{Assertions: []parser.Assertion{a}}
}

func assertStatus(code int) parser.Assertion {
	return parser.Assertion{Kind: parser.AssertStatus, StatusCode: code}
}

func assertHeader(key, val string) parser.Assertion {
	return parser.Assertion{Kind: parser.AssertHeader, HeaderKey: key, HeaderVal: val}
}

func assertBody(path, op, expected string) parser.Assertion {
	return parser.Assertion{Kind: parser.AssertBody, JSONPath: path, Operator: op, Expected: expected}
}

// ---- STATUS ----

func TestEvaluate_StatusPass(t *testing.T) {
	results := Evaluate(block(assertStatus(200)), resp(200, "", nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_StatusFail(t *testing.T) {
	results := Evaluate(block(assertStatus(201)), resp(200, "", nil))
	if results[0].Passed {
		t.Error("expected fail")
	}
	if results[0].Message == "" {
		t.Error("expected failure message")
	}
}

// ---- HEADER ----

func TestEvaluate_HeaderPass(t *testing.T) {
	h := map[string]string{"content-type": "application/json"}
	results := Evaluate(block(assertHeader("content-type", "application/json")), resp(200, "", h))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_HeaderPartialMatch(t *testing.T) {
	// asserter uses Contains, so "application/json" matches "application/json; charset=utf-8"
	h := map[string]string{"content-type": "application/json; charset=utf-8"}
	results := Evaluate(block(assertHeader("content-type", "application/json")), resp(200, "", h))
	if !results[0].Passed {
		t.Errorf("expected pass for partial header match: %s", results[0].Message)
	}
}

func TestEvaluate_HeaderMissing(t *testing.T) {
	results := Evaluate(block(assertHeader("x-custom", "value")), resp(200, "", nil))
	if results[0].Passed {
		t.Error("expected fail for missing header")
	}
}

func TestEvaluate_HeaderKeyLowercased(t *testing.T) {
	// asserter lowercases header keys before lookup
	h := map[string]string{"x-custom": "token123"}
	results := Evaluate(block(assertHeader("X-Custom", "token123")), resp(200, "", h))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

// ---- BODY == ----

func TestEvaluate_BodyEqualsStringPass(t *testing.T) {
	body := `{"name": "alice"}`
	results := Evaluate(block(assertBody("$.name", "==", `"alice"`)), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyEqualsStringFail(t *testing.T) {
	body := `{"name": "bob"}`
	results := Evaluate(block(assertBody("$.name", "==", `"alice"`)), resp(200, body, nil))
	if results[0].Passed {
		t.Error("expected fail")
	}
}

func TestEvaluate_BodyEqualsNumberPass(t *testing.T) {
	body := `{"count": 42}`
	results := Evaluate(block(assertBody("$.count", "==", "42")), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyEqualsBoolPass(t *testing.T) {
	body := `{"ok": true}`
	results := Evaluate(block(assertBody("$.ok", "==", "true")), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyTypeMismatch(t *testing.T) {
	// Raw JSON string "5" (with quotes) vs number 5 (no quotes in raw)
	body := `{"count": "5"}`
	results := Evaluate(block(assertBody("$.count", "==", "5")), resp(200, body, nil))
	if results[0].Passed {
		t.Error("expected fail: string '\"5\"' raw != bare 5")
	}
}

// ---- BODY exists / not exists ----

func TestEvaluate_BodyExistsPass(t *testing.T) {
	body := `{"field": "value"}`
	results := Evaluate(block(assertBody("$.field", "exists", "")), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyExistsFail(t *testing.T) {
	body := `{"other": "value"}`
	results := Evaluate(block(assertBody("$.field", "exists", "")), resp(200, body, nil))
	if results[0].Passed {
		t.Error("expected fail for missing field")
	}
}

func TestEvaluate_BodyNullFieldExists(t *testing.T) {
	// null JSON value — the key exists, so "exists" should pass
	body := `{"field": null}`
	results := Evaluate(block(assertBody("$.field", "exists", "")), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("null field should exist: %s", results[0].Message)
	}
}

func TestEvaluate_BodyNotExistsPass(t *testing.T) {
	body := `{"other": "x"}`
	results := Evaluate(block(assertBody("$.missing", "not exists", "")), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyNotExistsFail(t *testing.T) {
	body := `{"field": "x"}`
	results := Evaluate(block(assertBody("$.field", "not exists", "")), resp(200, body, nil))
	if results[0].Passed {
		t.Error("expected fail: field present but asserted not exists")
	}
}

// ---- BODY != ----

func TestEvaluate_BodyNotEqualsPass(t *testing.T) {
	body := `{"status": "ok"}`
	results := Evaluate(block(assertBody("$.status", "!=", `"error"`)), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyNotEqualsFail(t *testing.T) {
	body := `{"status": "error"}`
	results := Evaluate(block(assertBody("$.status", "!=", `"error"`)), resp(200, body, nil))
	if results[0].Passed {
		t.Error("expected fail: value equals the 'not equal' target")
	}
}

// ---- BODY contains ----

func TestEvaluate_BodyContainsStringPass(t *testing.T) {
	body := `{"message": "Hello, world!"}`
	results := Evaluate(block(assertBody("$.message", "contains", `"world"`)), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyContainsStringFail(t *testing.T) {
	body := `{"message": "Hello!"}`
	results := Evaluate(block(assertBody("$.message", "contains", `"world"`)), resp(200, body, nil))
	if results[0].Passed {
		t.Error("expected fail")
	}
}

func TestEvaluate_BodyContainsArrayPass(t *testing.T) {
	body := `{"tags": ["a", "b", "c"]}`
	results := Evaluate(block(assertBody("$.tags", "contains", `"b"`)), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

// ---- Nested / array paths ----

func TestEvaluate_BodyNestedPath(t *testing.T) {
	body := `{"user": {"name": "Alice"}}`
	results := Evaluate(block(assertBody("$.user.name", "==", `"Alice"`)), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_BodyArrayIndex(t *testing.T) {
	body := `{"items": ["first", "second"]}`
	results := Evaluate(block(assertBody("$.items[0]", "==", `"first"`)), resp(200, body, nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

// ---- Edge cases ----

func TestEvaluate_EmptyBodyStatusOnly(t *testing.T) {
	// 204 No Content — status assertion should still work with empty body
	results := Evaluate(block(assertStatus(204)), resp(204, "", nil))
	if !results[0].Passed {
		t.Errorf("expected pass: %s", results[0].Message)
	}
}

func TestEvaluate_MultipleAssertions(t *testing.T) {
	body := `{"id": 1, "name": "alice"}`
	b := parser.AssertionBlock{
		Assertions: []parser.Assertion{
			assertStatus(200),
			assertBody("$.id", "==", "1"),
			assertBody("$.name", "==", `"alice"`),
			assertBody("$.missing", "not exists", ""),
		},
	}
	results := Evaluate(b, resp(200, body, nil))
	for i, r := range results {
		if !r.Passed {
			t.Errorf("assertion %d failed: %s", i, r.Message)
		}
	}
}

func TestToGJSONPath_Conversions(t *testing.T) {
	cases := []struct{ in, want string }{
		{"$.name", "name"},
		{"$.user.name", "user.name"},
		{"$.items[0]", "items.0"},
		{"$.items[0].id", "items.0.id"},
		{"$", "@this"},
	}
	for _, c := range cases {
		got := toGJSONPath(c.in)
		if got != c.want {
			t.Errorf("toGJSONPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
