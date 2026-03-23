package parser

import (
	"strings"
	"testing"
)

// ---- ParseHTTP ----

func TestParseHTTP_SingleRequest(t *testing.T) {
	reqs, err := ParseHTTP("GET http://example.com\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	r := reqs[0]
	if r.Name != "" || r.Method != "GET" || r.URL != "http://example.com" {
		t.Errorf("unexpected request: %+v", r)
	}
	if len(r.Headers) != 0 {
		t.Errorf("expected no headers, got %v", r.Headers)
	}
	if r.Body != "" {
		t.Errorf("expected empty body, got %q", r.Body)
	}
}

func TestParseHTTP_WithHeaders(t *testing.T) {
	input := "POST http://example.com/api\nContent-Type: application/json\nAccept: */*\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Method != "POST" {
		t.Errorf("expected POST, got %q", reqs[0].Method)
	}
	if reqs[0].Headers["Content-Type"] != "application/json" {
		t.Errorf("unexpected Content-Type: %q", reqs[0].Headers["Content-Type"])
	}
	if reqs[0].Headers["Accept"] != "*/*" {
		t.Errorf("unexpected Accept: %q", reqs[0].Headers["Accept"])
	}
}

func TestParseHTTP_HeaderWithColonInValue(t *testing.T) {
	input := "GET http://example.com\nAuthorization: Bearer tok:en:val\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Headers["Authorization"] != "Bearer tok:en:val" {
		t.Errorf("colon in header value corrupted: %q", reqs[0].Headers["Authorization"])
	}
}

func TestParseHTTP_WithJSONBody(t *testing.T) {
	input := "POST http://example.com\nContent-Type: application/json\n\n{\"name\": \"alice\"}\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Body != `{"name": "alice"}` {
		t.Errorf("unexpected body: %q", reqs[0].Body)
	}
}

func TestParseHTTP_WithMultilineBody(t *testing.T) {
	input := "POST http://example.com\n\n{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(reqs[0].Body, `"a": 1`) {
		t.Errorf("multiline body not preserved: %q", reqs[0].Body)
	}
}

func TestParseHTTP_MultipleNamedBlocks(t *testing.T) {
	input := "### create\nPOST http://example.com/users\n\n### list\nGET http://example.com/users\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}
	if reqs[0].Name != "create" || reqs[0].Method != "POST" {
		t.Errorf("first block: %+v", reqs[0])
	}
	if reqs[1].Name != "list" || reqs[1].Method != "GET" {
		t.Errorf("second block: %+v", reqs[1])
	}
}

func TestParseHTTP_NamedBlockWithBody(t *testing.T) {
	input := "### create\nPOST http://example.com\nContent-Type: application/json\n\n{\"x\": 1}\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Body != `{"x": 1}` {
		t.Errorf("unexpected body: %q", reqs[0].Body)
	}
}

func TestParseHTTP_URLTrimmed(t *testing.T) {
	reqs, err := ParseHTTP("GET   http://example.com/path  \n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].URL != "http://example.com/path" {
		t.Errorf("URL not trimmed: %q", reqs[0].URL)
	}
}

func TestParseHTTP_EmptyInputReturnsError(t *testing.T) {
	_, err := ParseHTTP("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestParseHTTP_MissingMethodURLReturnsError(t *testing.T) {
	_, err := ParseHTTP("### block\nGET\n")
	if err == nil {
		t.Error("expected error for single-token request line")
	}
}

func TestParseHTTP_HeaderCasePreserved(t *testing.T) {
	input := "GET http://example.com\nX-Custom-Header: value\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Headers["X-Custom-Header"] != "value" {
		t.Errorf("header key case not preserved: %v", reqs[0].Headers)
	}
}

func TestParseHTTP_CRLFNormalized(t *testing.T) {
	input := "GET http://example.com\r\nX-H: v\r\n"
	reqs, err := ParseHTTP(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqs[0].Headers["X-H"] != "v" {
		t.Errorf("CRLF not handled: %v", reqs[0].Headers)
	}
}

// ---- ParseAssert ----

func TestParseAssert_Status(t *testing.T) {
	blocks, err := ParseAssert("STATUS 200\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 || len(blocks[0].Assertions) != 1 {
		t.Fatalf("unexpected result: %+v", blocks)
	}
	a := blocks[0].Assertions[0]
	if a.Kind != AssertStatus || a.StatusCode != 200 {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_Header(t *testing.T) {
	blocks, err := ParseAssert("HEADER content-type: application/json\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Kind != AssertHeader || a.HeaderKey != "content-type" || a.HeaderVal != "application/json" {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_HeaderColonInValue(t *testing.T) {
	blocks, err := ParseAssert("HEADER x-token: tok:en:value\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.HeaderVal != "tok:en:value" {
		t.Errorf("colon in header value corrupted: %q", a.HeaderVal)
	}
}

func TestParseAssert_BodyEqualsString(t *testing.T) {
	blocks, err := ParseAssert(`BODY $.name == "alice"` + "\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Kind != AssertBody || a.JSONPath != "$.name" || a.Operator != "==" || a.Expected != `"alice"` {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_BodyEqualsNumber(t *testing.T) {
	blocks, err := ParseAssert("BODY $.count == 42\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Operator != "==" || a.Expected != "42" {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_BodyEqualsBool(t *testing.T) {
	for _, tc := range []struct{ input, expected string }{
		{"BODY $.ok == true\n", "true"},
		{"BODY $.ok == false\n", "false"},
	} {
		blocks, err := ParseAssert(tc.input)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", tc.input, err)
		}
		if blocks[0].Assertions[0].Expected != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, blocks[0].Assertions[0].Expected)
		}
	}
}

func TestParseAssert_BodyExists(t *testing.T) {
	blocks, err := ParseAssert("BODY $.field exists\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Operator != "exists" || a.Expected != "" {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_BodyNotExists(t *testing.T) {
	blocks, err := ParseAssert("BODY $.field not exists\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Operator != "not exists" {
		t.Errorf("expected 'not exists', got %q", a.Operator)
	}
}

func TestParseAssert_BodyNotEquals(t *testing.T) {
	blocks, err := ParseAssert(`BODY $.status != "error"` + "\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Operator != "!=" || a.Expected != `"error"` {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_BodyContains(t *testing.T) {
	blocks, err := ParseAssert(`BODY $.message contains "hello"` + "\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.Operator != "contains" || a.Expected != `"hello"` {
		t.Errorf("unexpected assertion: %+v", a)
	}
}

func TestParseAssert_MultipleNamedBlocks(t *testing.T) {
	input := "### create\nSTATUS 201\n\n### list\nSTATUS 200\nBODY $.count exists\n"
	blocks, err := ParseAssert(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Name != "create" || blocks[0].Assertions[0].StatusCode != 201 {
		t.Errorf("first block: %+v", blocks[0])
	}
	if blocks[1].Name != "list" || len(blocks[1].Assertions) != 2 {
		t.Errorf("second block: %+v", blocks[1])
	}
}

func TestParseAssert_UnnamedBlock(t *testing.T) {
	blocks, err := ParseAssert("STATUS 200\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocks[0].Name != "" {
		t.Errorf("expected empty name, got %q", blocks[0].Name)
	}
}

func TestParseAssert_CommentsIgnored(t *testing.T) {
	input := "# this is a comment\nSTATUS 200\n# another comment\n"
	blocks, err := ParseAssert(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks[0].Assertions) != 1 {
		t.Errorf("expected 1 assertion (comments skipped), got %d", len(blocks[0].Assertions))
	}
}

func TestParseAssert_UnknownKeywordReturnsError(t *testing.T) {
	_, err := ParseAssert("FOOBAR 200\n")
	if err == nil {
		t.Error("expected error for unknown keyword, got nil")
	}
}

func TestParseAssert_MalformedStatusReturnsError(t *testing.T) {
	_, err := ParseAssert("STATUS notanumber\n")
	if err == nil {
		t.Error("expected error for non-numeric status, got nil")
	}
}

func TestParseAssert_BodyMissingOperatorReturnsError(t *testing.T) {
	_, err := ParseAssert("BODY $.field\n")
	if err == nil {
		t.Error("expected error for BODY with missing operator, got nil")
	}
}

func TestParseAssert_BodyUnknownOperatorReturnsError(t *testing.T) {
	_, err := ParseAssert("BODY $.field like value\n")
	if err == nil {
		t.Error("expected error for unknown BODY operator, got nil")
	}
}

func TestParseAssert_BodyPathPreserved(t *testing.T) {
	blocks, err := ParseAssert("BODY $.user.address.city == \"London\"\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.JSONPath != "$.user.address.city" {
		t.Errorf("path corrupted: %q", a.JSONPath)
	}
}

func TestParseAssert_BodyArrayIndex(t *testing.T) {
	blocks, err := ParseAssert("BODY $.items[0] exists\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := blocks[0].Assertions[0]
	if a.JSONPath != "$.items[0]" || a.Operator != "exists" {
		t.Errorf("unexpected assertion: %+v", a)
	}
}
