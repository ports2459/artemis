package editor

import (
	"testing"
)

func TestFormatCSharpBasic(t *testing.T) {
	input := "namespace Foo\n{\npublic class Bar\n{\n}\n}"
	expected := "namespace Foo\n{\n    public class Bar\n    {\n    }\n}"
	result := FormatCSharp(input)
	if result != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormatCSharpNested(t *testing.T) {
	input := "class Foo\n{\nvoid Bar()\n{\nint x = 1;\n}\n}"
	expected := "class Foo\n{\n    void Bar()\n    {\n        int x = 1;\n    }\n}"
	result := FormatCSharp(input)
	if result != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormatCSharpPreservesBlankLines(t *testing.T) {
	input := "using System;\n\nnamespace Foo\n{\n}"
	expected := "using System;\n\nnamespace Foo\n{\n}"
	result := FormatCSharp(input)
	if result != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormatCSharpBracesInStrings(t *testing.T) {
	input := "class Foo\n{\nstring s = \"{hello}\";\n}"
	expected := "class Foo\n{\n    string s = \"{hello}\";\n}"
	result := FormatCSharp(input)
	if result != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormatCSharpBracesInComments(t *testing.T) {
	input := "class Foo\n{\n// { this brace should be ignored\nint x;\n}"
	expected := "class Foo\n{\n    // { this brace should be ignored\n    int x;\n}"
	result := FormatCSharp(input)
	if result != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormatCSharpAlreadyFormatted(t *testing.T) {
	input := "namespace Foo\n{\n    public class Bar\n    {\n    }\n}"
	result := FormatCSharp(input)
	if result != input {
		t.Errorf("already-formatted code should not change\ngot:\n%s", result)
	}
}

func TestFormatNonCSharp(t *testing.T) {
	input := "some lua code"
	result := Format(input, LangLua)
	if result != input {
		t.Error("non-C# should return unchanged")
	}
}
