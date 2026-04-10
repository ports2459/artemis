package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerNew(t *testing.T) {
	r := NewRunner()
	if r == nil {
		t.Fatal("NewRunner returned nil")
	}
	if r.Status() != StatusIdle {
		t.Errorf("expected StatusIdle, got %v", r.Status())
	}
	if r.Result() != nil {
		t.Error("expected nil result for new runner")
	}
	if len(r.Output()) != 0 {
		t.Error("expected empty output for new runner")
	}
}

func TestRunnerSetProject(t *testing.T) {
	r := NewRunner()
	r.SetProject("/some/path", "Release")

	if r.projectPath != "/some/path" {
		t.Errorf("expected projectPath '/some/path', got '%s'", r.projectPath)
	}
	if r.config != "Release" {
		t.Errorf("expected config 'Release', got '%s'", r.config)
	}

	// Empty configuration should not overwrite.
	r.SetProject("/other/path", "")
	if r.config != "Release" {
		t.Errorf("expected config to remain 'Release', got '%s'", r.config)
	}
	if r.projectPath != "/other/path" {
		t.Errorf("expected projectPath '/other/path', got '%s'", r.projectPath)
	}
}

func TestRunnerStatusIdle(t *testing.T) {
	r := NewRunner()
	if r.Status() != StatusIdle {
		t.Errorf("expected StatusIdle, got %v", r.Status())
	}

	// Verify the string representation.
	if r.Status().String() != "idle" {
		t.Errorf("expected 'idle', got '%s'", r.Status().String())
	}

	// Verify other status strings.
	if StatusBuilding.String() != "building..." {
		t.Errorf("expected 'building...', got '%s'", StatusBuilding.String())
	}
	if StatusSuccess.String() != "succeeded" {
		t.Errorf("expected 'succeeded', got '%s'", StatusSuccess.String())
	}
	if StatusFailed.String() != "failed" {
		t.Errorf("expected 'failed', got '%s'", StatusFailed.String())
	}
}

func TestDeployCreatesDirectory(t *testing.T) {
	// Create a temp directory for the source DLL.
	srcDir := t.TempDir()
	dllContent := []byte("fake DLL content for testing")
	dllPath := filepath.Join(srcDir, "TestMod.dll")
	if err := os.WriteFile(dllPath, dllContent, 0644); err != nil {
		t.Fatalf("failed to create temp DLL: %v", err)
	}

	// Deploy to a non-existent subdirectory.
	deployDir := filepath.Join(t.TempDir(), "plugins", "nested")

	r := NewRunner()
	if err := r.Deploy(dllPath, deployDir); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify directory was created.
	info, err := os.Stat(deployDir)
	if err != nil {
		t.Fatalf("deploy dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("deploy path is not a directory")
	}

	// Verify file was copied.
	destPath := filepath.Join(deployDir, "TestMod.dll")
	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read deployed DLL: %v", err)
	}
	if string(destContent) != string(dllContent) {
		t.Errorf("deployed DLL content mismatch: got %q, want %q", destContent, dllContent)
	}
}

func TestClassifyLine(t *testing.T) {
	tests := []struct {
		text      string
		fromErr   bool
		wantError bool
		wantInfo  bool
	}{
		{"Program.cs(10,5): error CS1001: ...", false, true, false},
		{"Build succeeded.", false, false, true},
		{"Restore complete", false, false, true},
		{"  Compiling...", false, false, false},
		{"something on stderr", true, true, false},
		{"Program.cs(10,5): warning CS0168: ...", false, false, true},
	}

	for _, tt := range tests {
		line := classifyLine(tt.text, tt.fromErr)
		if line.IsError != tt.wantError {
			t.Errorf("classifyLine(%q, %v).IsError = %v, want %v",
				tt.text, tt.fromErr, line.IsError, tt.wantError)
		}
		if line.IsInfo != tt.wantInfo {
			t.Errorf("classifyLine(%q, %v).IsInfo = %v, want %v",
				tt.text, tt.fromErr, line.IsInfo, tt.wantInfo)
		}
	}
}

func TestRunnerReset(t *testing.T) {
	r := NewRunner()
	r.mu.Lock()
	r.status = StatusSuccess
	r.output = []OutputLine{{Text: "test"}}
	res := Result{Status: StatusSuccess}
	r.result = &res
	r.mu.Unlock()

	r.Reset()

	if r.Status() != StatusIdle {
		t.Errorf("after Reset, expected StatusIdle, got %v", r.Status())
	}
	if len(r.Output()) != 0 {
		t.Error("after Reset, expected empty output")
	}
	if r.Result() != nil {
		t.Error("after Reset, expected nil result")
	}
}
