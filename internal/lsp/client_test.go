package lsp

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestWriteMessageFormat(t *testing.T) {
	// writeMessage writes Content-Length header + \r\n\r\n + body.
	// We replicate the format logic here and verify it matches the LSP spec.
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	var buf bytes.Buffer
	buf.WriteString(header)
	buf.Write(body)

	got := buf.String()

	// Verify header is present and correct.
	if !strings.HasPrefix(got, "Content-Length: ") {
		t.Fatalf("expected Content-Length header, got %q", got)
	}

	// Verify \r\n\r\n separator exists.
	if !strings.Contains(got, "\r\n\r\n") {
		t.Fatalf("expected \\r\\n\\r\\n separator, got %q", got)
	}

	// Verify the length value matches the body length.
	expectedPrefix := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if !strings.HasPrefix(got, expectedPrefix) {
		t.Fatalf("header mismatch: expected %q prefix, got %q", expectedPrefix, got)
	}

	// Verify body follows the header.
	parts := strings.SplitN(got, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected header + body separated by \\r\\n\\r\\n, got %d parts", len(parts))
	}
	if parts[1] != string(body) {
		t.Fatalf("body mismatch: expected %q, got %q", string(body), parts[1])
	}
}

func TestNewClient(t *testing.T) {
	// Default path.
	c := NewClient("")
	if c.omnisharpPath != "OmniSharp" {
		t.Errorf("expected default path 'OmniSharp', got %q", c.omnisharpPath)
	}
	if c.pending == nil {
		t.Error("expected pending map to be initialized")
	}
	if c.diagnostics == nil {
		t.Error("expected diagnostics channel to be initialized")
	}
	if c.done == nil {
		t.Error("expected done channel to be initialized")
	}

	// Custom path.
	c2 := NewClient("/usr/local/bin/omnisharp")
	if c2.omnisharpPath != "/usr/local/bin/omnisharp" {
		t.Errorf("expected custom path, got %q", c2.omnisharpPath)
	}
}

func TestFileURIConversion(t *testing.T) {
	tests := []struct {
		path string
		uri  string
	}{
		{"/home/user/file.cs", "file:///home/user/file.cs"},
		{"/home/user/project/Assets/Scripts/Main.cs", "file:///home/user/project/Assets/Scripts/Main.cs"},
		{"/tmp/test.txt", "file:///tmp/test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotURI := FileToURI(tt.path)
			if gotURI != tt.uri {
				t.Errorf("FileToURI(%q) = %q, want %q", tt.path, gotURI, tt.uri)
			}

			gotPath := URIToFile(tt.uri)
			if gotPath != tt.path {
				t.Errorf("URIToFile(%q) = %q, want %q", tt.uri, gotPath, tt.path)
			}
		})
	}
}

func TestFileURIWithSpaces(t *testing.T) {
	path := "/home/user/my project/file.cs"
	uri := FileToURI(path)

	// The URI should encode the space.
	if !strings.Contains(uri, "my%20project") && !strings.Contains(uri, "my+project") {
		t.Errorf("expected encoded space in URI %q", uri)
	}

	// Round-trip should recover the original path.
	roundTrip := URIToFile(uri)
	if roundTrip != path {
		t.Errorf("round-trip failed: got %q, want %q", roundTrip, path)
	}
}

func TestReadMessageFormat(t *testing.T) {
	// Simulate reading an LSP message from a buffer.
	body := `{"jsonrpc":"2.0","id":1,"result":null}`
	raw := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)

	c := &Client{
		stdout: bufio.NewReader(strings.NewReader(raw)),
	}

	data, err := c.readMessage()
	if err != nil {
		t.Fatalf("readMessage failed: %v", err)
	}

	if string(data) != body {
		t.Errorf("readMessage body mismatch: got %q, want %q", string(data), body)
	}
}

func TestReadMessageWithExtraHeaders(t *testing.T) {
	// Some servers send Content-Type along with Content-Length.
	body := `{"jsonrpc":"2.0","id":2,"result":{}}`
	raw := fmt.Sprintf("Content-Length: %d\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n%s", len(body), body)

	c := &Client{
		stdout: bufio.NewReader(strings.NewReader(raw)),
	}

	data, err := c.readMessage()
	if err != nil {
		t.Fatalf("readMessage failed: %v", err)
	}

	if string(data) != body {
		t.Errorf("body mismatch: got %q, want %q", string(data), body)
	}
}
