package app

import "github.com/eden/artemis/internal/panel"

type SwitchPanelMsg struct {
	Panel panel.PanelID
}

type OpenProjectMsg struct {
	Path string
}

type StatusMsg struct {
	Text string
}

// FileOpenMsg requests opening a file in the editor.
type FileOpenMsg struct {
	Path string
}

// FileSavedMsg confirms a file was saved successfully.
type FileSavedMsg struct {
	Path string
}

// FileCloseMsg requests closing the active editor tab.
type FileCloseMsg struct{}

// DiagnosticItem represents a single diagnostic from the LSP server.
type DiagnosticItem struct {
	Line     int
	Col      int
	EndLine  int
	EndCol   int
	Severity int // 1=Error, 2=Warning, 3=Info, 4=Hint
	Message  string
}

// DiagnosticsMsg carries LSP diagnostics for a file.
type DiagnosticsMsg struct {
	Path        string
	Diagnostics []DiagnosticItem
}

// CompletionItem represents a single completion suggestion.
type CompletionItem struct {
	Label  string
	Detail string
	Kind   int
}

// CompletionMsg carries completion results from the LSP server.
type CompletionMsg struct {
	Items []CompletionItem
}
