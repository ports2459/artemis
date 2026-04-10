package editor

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	// Force color output so lipgloss styles produce ANSI sequences in tests
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want Language
	}{
		{"foo.cs", LangCSharp},
		{"bar.CS", LangCSharp},
		{"config.json", LangJSON},
		{"settings.toml", LangTOML},
		{"script.lua", LangLua},
		{"readme.md", LangNone},
		{"main.go", LangNone},
		{"noext", LangNone},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %d, want %d", tt.path, got, tt.want)
		}
	}
}

func TestHighlightCSharpKeywords(t *testing.T) {
	input := "public class MyClass"
	output := HighlightLine(input, LangCSharp)
	if output == input {
		t.Error("expected highlighted output to differ from input for C# keywords")
	}
}

func TestHighlightCSharpString(t *testing.T) {
	input := `var x = "hello world";`
	output := HighlightLine(input, LangCSharp)
	if output == input {
		t.Error("expected highlighted output to differ from input for C# string")
	}
}

func TestHighlightCSharpComment(t *testing.T) {
	input := "// this is a comment"
	output := HighlightLine(input, LangCSharp)
	if output == input {
		t.Error("expected highlighted output to differ from input for C# comment")
	}
}

func TestHighlightNone(t *testing.T) {
	input := "public class MyClass"
	output := HighlightLine(input, LangNone)
	if output != input {
		t.Errorf("LangNone should return input unchanged, got %q", output)
	}
}

func TestHighlightJSONKeys(t *testing.T) {
	input := `"name": "value"`
	output := HighlightLine(input, LangJSON)
	if output == input {
		t.Error("expected highlighted output to differ from input for JSON")
	}
}

func TestHighlightTOMLComment(t *testing.T) {
	input := "# this is a comment"
	output := HighlightLine(input, LangTOML)
	if output == input {
		t.Error("expected highlighted output to differ from input for TOML comment")
	}
}

func TestHighlightLuaKeywords(t *testing.T) {
	input := "local function hello()"
	output := HighlightLine(input, LangLua)
	if output == input {
		t.Error("expected highlighted output to differ from input for Lua keywords")
	}
}

func TestHighlightContentMultiLine(t *testing.T) {
	input := "public class Foo\n// comment\nint x = 42;"
	output := HighlightContent(input, LangCSharp)
	if output == input {
		t.Error("expected HighlightContent to produce styled output")
	}
}
