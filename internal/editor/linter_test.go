package editor

import (
	"strings"
	"testing"
)

func TestLintTrailingWhitespace(t *testing.T) {
	results := LintCSharp("int x = 1;   \n")
	found := false
	for _, r := range results {
		if r.Message == "Trailing whitespace" {
			found = true
		}
	}
	if !found {
		t.Error("expected trailing whitespace warning")
	}
}

func TestLintUnclosedBrace(t *testing.T) {
	results := LintCSharp("class Foo {\n")
	found := false
	for _, r := range results {
		if r.Severity == 1 && strings.Contains(r.Message, "unclosed") {
			found = true
		}
	}
	if !found {
		t.Error("expected unclosed brace error")
	}
}

func TestLintMissingSemicolon(t *testing.T) {
	results := LintCSharp("return 42\n")
	found := false
	for _, r := range results {
		if strings.Contains(r.Message, "semicolon") {
			found = true
		}
	}
	if !found {
		t.Error("expected missing semicolon warning")
	}
}

func TestLintCleanCode(t *testing.T) {
	code := "using System;\n\nnamespace Foo\n{\n    public class Bar\n    {\n    }\n}\n"
	results := LintCSharp(code)
	errors := 0
	for _, r := range results {
		if r.Severity <= 2 {
			errors++
		}
	}
	if errors > 0 {
		for _, r := range results {
			t.Logf("Line %d: [%d] %s", r.Line, r.Severity, r.Message)
		}
		t.Errorf("clean code should have no errors/warnings, got %d", errors)
	}
}

func TestLintSkipsComments(t *testing.T) {
	code := "// return without semicolon\n"
	results := LintCSharp(code)
	for _, r := range results {
		if strings.Contains(r.Message, "semicolon") {
			t.Error("should not flag missing semicolon in comments")
		}
	}
}
