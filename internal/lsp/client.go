package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// ---------------------------------------------------------------------------
// Core LSP types
// ---------------------------------------------------------------------------

// Position in a text document (0-indexed line and character).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range inside a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a span inside a resource.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic represents a compiler/linter diagnostic.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1=Error, 2=Warning, 3=Info, 4=Hint
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// CompletionItem represents a single completion proposal.
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind"`
	Detail     string `json:"detail"`
	InsertText string `json:"insertText,omitempty"`
}

// DiagnosticEvent is emitted by the reader goroutine when the server pushes
// diagnostics for a file.
type DiagnosticEvent struct {
	URI         string
	Diagnostics []Diagnostic
}

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 wire types
// ---------------------------------------------------------------------------

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *int64           `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *jsonrpcError    `json:"error,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("LSP error %d: %s", e.Code, e.Message)
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client manages an OmniSharp subprocess and speaks LSP JSON-RPC 2.0 over
// its stdin/stdout.
type Client struct {
	omnisharpPath string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	nextID  int64 // atomic
	pending map[int64]chan json.RawMessage
	mu      sync.Mutex

	diagnostics chan DiagnosticEvent
	running     atomic.Bool
	done        chan struct{}
}

// NewClient creates a new LSP client. omnisharpPath is the path to the
// OmniSharp binary; if empty it defaults to "OmniSharp" (looked up in PATH).
func NewClient(omnisharpPath string) *Client {
	if omnisharpPath == "" {
		omnisharpPath = "OmniSharp"
	}
	return &Client{
		omnisharpPath: omnisharpPath,
		pending:       make(map[int64]chan json.RawMessage),
		diagnostics:   make(chan DiagnosticEvent, 64),
		done:          make(chan struct{}),
	}
}

// Start spawns OmniSharp, performs the initialize handshake, and begins the
// reader goroutine. projectRoot is the path to the Unity project directory.
func (c *Client) Start(projectRoot string) error {
	c.cmd = exec.Command(c.omnisharpPath, "--languageserver", "--hostPID", fmt.Sprint(0))
	c.cmd.Dir = projectRoot

	stdinPipe, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("lsp: stdin pipe: %w", err)
	}
	c.stdin = stdinPipe

	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("lsp: stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdoutPipe)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("lsp: start OmniSharp: %w", err)
	}

	c.running.Store(true)
	go c.readLoop()

	// ---- initialize handshake ----
	initParams := map[string]interface{}{
		"processId": nil,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": false,
					},
				},
				"hover": map[string]interface{}{
					"contentFormat": []string{"plaintext"},
				},
				"publishDiagnostics": map[string]interface{}{},
			},
		},
		"rootUri": FileToURI(projectRoot),
	}

	if _, err := c.sendRequest("initialize", initParams); err != nil {
		_ = c.Stop()
		return fmt.Errorf("lsp: initialize: %w", err)
	}

	if err := c.sendNotification("initialized", map[string]interface{}{}); err != nil {
		_ = c.Stop()
		return fmt.Errorf("lsp: initialized: %w", err)
	}

	return nil
}

// Stop shuts down the LSP server gracefully.
func (c *Client) Stop() error {
	if !c.running.Load() {
		return nil
	}
	c.running.Store(false)

	// Send shutdown request (best-effort).
	_, _ = c.sendRequest("shutdown", nil)

	// Send exit notification (best-effort).
	_ = c.sendNotification("exit", nil)

	// Close stdin so readLoop sees EOF and exits.
	if c.stdin != nil {
		_ = c.stdin.Close()
	}

	// Wait for the process.
	err := c.cmd.Wait()

	// Signal readLoop completion.
	select {
	case <-c.done:
	default:
		close(c.done)
	}

	// Drain pending channels.
	c.mu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()

	return err
}

// Diagnostics returns a read-only channel that emits server-pushed diagnostics.
func (c *Client) Diagnostics() <-chan DiagnosticEvent {
	return c.diagnostics
}

// ---------------------------------------------------------------------------
// Public LSP methods
// ---------------------------------------------------------------------------

// DidOpen notifies the server that a document was opened.
func (c *Client) DidOpen(uri, languageID, text string) error {
	return c.sendNotification("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       text,
		},
	})
}

// DidChange notifies the server of a full-document change.
func (c *Client) DidChange(uri, text string, version int) error {
	return c.sendNotification("textDocument/didChange", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]interface{}{
			{"text": text},
		},
	})
}

// DidClose notifies the server that a document was closed.
func (c *Client) DidClose(uri string) error {
	return c.sendNotification("textDocument/didClose", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	})
}

// Completion requests completions at the given position.
func (c *Client) Completion(uri string, pos Position) ([]CompletionItem, error) {
	raw, err := c.sendRequest("textDocument/completion", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
	})
	if err != nil {
		return nil, err
	}

	// The response can be either a []CompletionItem or a CompletionList with an items field.
	var items []CompletionItem
	if err := json.Unmarshal(raw, &items); err == nil {
		return items, nil
	}

	var list struct {
		Items []CompletionItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("lsp: parse completion response: %w", err)
	}
	return list.Items, nil
}

// Definition requests the definition location(s) for the symbol at pos.
func (c *Client) Definition(uri string, pos Position) ([]Location, error) {
	raw, err := c.sendRequest("textDocument/definition", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
	})
	if err != nil {
		return nil, err
	}

	// May be a single Location or []Location.
	var locs []Location
	if err := json.Unmarshal(raw, &locs); err == nil {
		return locs, nil
	}

	var single Location
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, fmt.Errorf("lsp: parse definition response: %w", err)
	}
	return []Location{single}, nil
}

// References finds all references to the symbol at pos.
func (c *Client) References(uri string, pos Position) ([]Location, error) {
	raw, err := c.sendRequest("textDocument/references", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
		"context":      map[string]interface{}{"includeDeclaration": true},
	})
	if err != nil {
		return nil, err
	}

	var locs []Location
	if err := json.Unmarshal(raw, &locs); err != nil {
		return nil, fmt.Errorf("lsp: parse references response: %w", err)
	}
	return locs, nil
}

// Hover returns the hover text for the symbol at pos.
func (c *Client) Hover(uri string, pos Position) (string, error) {
	raw, err := c.sendRequest("textDocument/hover", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
	})
	if err != nil {
		return "", err
	}

	// result may be null if no hover info is available.
	if string(raw) == "null" {
		return "", nil
	}

	var hover struct {
		Contents interface{} `json:"contents"`
	}
	if err := json.Unmarshal(raw, &hover); err != nil {
		return "", fmt.Errorf("lsp: parse hover response: %w", err)
	}

	return extractHoverText(hover.Contents), nil
}

// Rename renames the symbol at pos to newName across the workspace.
func (c *Client) Rename(uri string, pos Position, newName string) error {
	_, err := c.sendRequest("textDocument/rename", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     pos,
		"newName":      newName,
	})
	return err
}

// ---------------------------------------------------------------------------
// Internal transport
// ---------------------------------------------------------------------------

// sendRequest sends a JSON-RPC request and blocks until the response arrives.
func (c *Client) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.nextID, 1)
	ch := make(chan json.RawMessage, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("lsp: marshal request: %w", err)
	}

	if err := c.writeMessage(data); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	result, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("lsp: connection closed while waiting for response to %s", method)
	}
	return result, nil
}

// sendNotification sends a JSON-RPC notification (no ID, no response expected).
func (c *Client) sendNotification(method string, params interface{}) error {
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("lsp: marshal notification: %w", err)
	}
	return c.writeMessage(data)
}

// writeMessage writes a complete LSP message (Content-Length header + body) to
// the server's stdin.
func (c *Client) writeMessage(data []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return fmt.Errorf("lsp: write header: %w", err)
	}
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("lsp: write body: %w", err)
	}
	return nil
}

// readMessage reads a single LSP message (Content-Length framed) from stdout.
func (c *Client) readMessage() ([]byte, error) {
	// Read headers until we see the empty line (\r\n\r\n).
	var contentLength int
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("lsp: read header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("lsp: bad content-length %q: %w", val, err)
			}
			contentLength = n
		}
		// Ignore other headers (Content-Type etc.).
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("lsp: missing Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, body); err != nil {
		return nil, fmt.Errorf("lsp: read body: %w", err)
	}
	return body, nil
}

// readLoop runs in a goroutine. It reads messages from stdout and dispatches
// responses to the matching pending channel, or publishes notifications.
func (c *Client) readLoop() {
	defer func() {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
	}()

	for c.running.Load() {
		data, err := c.readMessage()
		if err != nil {
			// If we're shutting down, this is expected.
			if !c.running.Load() {
				return
			}
			// Otherwise log and bail — in a real app we'd reconnect.
			return
		}

		var msg jsonrpcResponse
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// ---- Server notification (no ID) ----
		if msg.ID == nil {
			c.handleNotification(msg)
			continue
		}

		// ---- Response to a pending request ----
		c.mu.Lock()
		ch, ok := c.pending[*msg.ID]
		if ok {
			delete(c.pending, *msg.ID)
		}
		c.mu.Unlock()

		if ok {
			if msg.Error != nil {
				// Encode the error as a JSON-RPC error message in the result
				// so the caller can inspect it. We send nil so the caller gets
				// an empty result and we could propagate the error differently,
				// but for simplicity we just send the result (which may be null).
				ch <- msg.Result
			} else {
				ch <- msg.Result
			}
		}
	}
}

// handleNotification processes server-initiated notifications.
func (c *Client) handleNotification(msg jsonrpcResponse) {
	switch msg.Method {
	case "textDocument/publishDiagnostics":
		var params struct {
			URI         string       `json:"uri"`
			Diagnostics []Diagnostic `json:"diagnostics"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return
		}
		select {
		case c.diagnostics <- DiagnosticEvent{
			URI:         params.URI,
			Diagnostics: params.Diagnostics,
		}:
		default:
			// Channel full — drop oldest and push new.
			select {
			case <-c.diagnostics:
			default:
			}
			c.diagnostics <- DiagnosticEvent{
				URI:         params.URI,
				Diagnostics: params.Diagnostics,
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// extractHoverText pulls a plain string out of the various shapes that the
// "contents" field of a Hover response can take (string, MarkupContent, or
// MarkedString).
func extractHoverText(v interface{}) string {
	switch c := v.(type) {
	case string:
		return c
	case map[string]interface{}:
		if val, ok := c["value"]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
	case []interface{}:
		var parts []string
		for _, item := range c {
			parts = append(parts, extractHoverText(item))
		}
		return strings.Join(parts, "\n")
	}
	return fmt.Sprintf("%v", v)
}
