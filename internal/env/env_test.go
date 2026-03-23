package env

import (
	"os"
	"testing"
)

// ---- Load ----

func TestLoad_EmptyPathReturnsEmptyMap(t *testing.T) {
	vars, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty map, got %v", vars)
	}
}

func TestLoad_MissingFileReturnsEmptyMap(t *testing.T) {
	vars, err := Load("/nonexistent/path/to/.env.nope")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty map, got %v", vars)
	}
}

func TestLoad_BasicKeyValue(t *testing.T) {
	f := writeTempEnv(t, "KEY=value\n")
	vars, err := Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["KEY"] != "value" {
		t.Errorf("expected KEY=value, got %v", vars)
	}
}

func TestLoad_ValueWithSpaces(t *testing.T) {
	f := writeTempEnv(t, `KEY=hello world`+"\n")
	vars, err := Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["KEY"] != "hello world" {
		t.Errorf("expected 'hello world', got %q", vars["KEY"])
	}
}

func TestLoad_CommentsIgnored(t *testing.T) {
	f := writeTempEnv(t, "# this is a comment\nKEY=val\n")
	vars, err := Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["KEY"] != "val" {
		t.Errorf("expected KEY=val, got %v", vars)
	}
	if _, ok := vars["# this is a comment"]; ok {
		t.Error("comment line should not be parsed as a key")
	}
}

func TestLoad_MultipleKeys(t *testing.T) {
	f := writeTempEnv(t, "A=1\nB=2\nC=3\n")
	vars, err := Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["A"] != "1" || vars["B"] != "2" || vars["C"] != "3" {
		t.Errorf("unexpected vars: %v", vars)
	}
}

// ---- Substitute ----

func TestSubstitute_SingleVar(t *testing.T) {
	result := Substitute("http://{{HOST}}/api", map[string]string{"HOST": "localhost:8080"})
	if result != "http://localhost:8080/api" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestSubstitute_MultipleVars(t *testing.T) {
	result := Substitute("{{SCHEME}}://{{HOST}}/{{PATH}}", map[string]string{
		"SCHEME": "https",
		"HOST":   "example.com",
		"PATH":   "api/v1",
	})
	if result != "https://example.com/api/v1" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestSubstitute_UnknownVarLeftIntact(t *testing.T) {
	result := Substitute("http://{{UNKNOWN}}/api", map[string]string{})
	if result != "http://{{UNKNOWN}}/api" {
		t.Errorf("unknown var should be left intact, got: %q", result)
	}
}

func TestSubstitute_EmptyVarsMapUnchanged(t *testing.T) {
	input := "GET http://{{BASE_URL}}/health"
	result := Substitute(input, map[string]string{})
	if result != input {
		t.Errorf("expected unchanged string, got: %q", result)
	}
}

func TestSubstitute_VarAppearsMultipleTimes(t *testing.T) {
	result := Substitute("{{X}} and {{X}} again", map[string]string{"X": "hello"})
	if result != "hello and hello again" {
		t.Errorf("all occurrences should be replaced, got: %q", result)
	}
}

func TestSubstitute_NoVarsUnchanged(t *testing.T) {
	input := "plain string without vars"
	result := Substitute(input, map[string]string{"KEY": "val"})
	if result != input {
		t.Errorf("string without vars should be unchanged, got: %q", result)
	}
}

func TestSubstitute_PartialVarsReplaced(t *testing.T) {
	// One var known, one unknown
	result := Substitute("{{KNOWN}} {{UNKNOWN}}", map[string]string{"KNOWN": "yes"})
	if result != "yes {{UNKNOWN}}" {
		t.Errorf("unexpected result: %q", result)
	}
}

// ---- helpers ----

func writeTempEnv(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.env")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
