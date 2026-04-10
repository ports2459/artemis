package build

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Status represents the current state of the build runner.
type Status int

const (
	StatusIdle     Status = iota
	StatusBuilding
	StatusSuccess
	StatusFailed
)

func (s Status) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusBuilding:
		return "building..."
	case StatusSuccess:
		return "succeeded"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// OutputLine represents a single line of build output.
type OutputLine struct {
	Text    string
	IsError bool
	IsInfo  bool
}

// Result holds the outcome of a build.
type Result struct {
	Status   Status
	Duration time.Duration
	Errors   int
	Warnings int
}

// BuildOutputMsg is sent for each line of build output.
type BuildOutputMsg struct {
	Line OutputLine
}

// BuildCompleteMsg is sent when the build finishes.
type BuildCompleteMsg struct {
	Result Result
}

// Runner manages dotnet build as a subprocess with real-time output streaming.
type Runner struct {
	projectPath string
	config      string // "Debug" or "Release"
	status      Status
	output      []OutputLine
	result      *Result
	cmd         *exec.Cmd
	mu          sync.Mutex
}

// NewRunner creates a new idle build runner.
func NewRunner() *Runner {
	return &Runner{
		status: StatusIdle,
		config: "Debug",
	}
}

// SetProject configures the project path and build configuration.
func (r *Runner) SetProject(path, configuration string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projectPath = path
	if configuration != "" {
		r.config = configuration
	}
}

// Build returns a tea.Cmd that runs `dotnet build` asynchronously.
// It sends BuildOutputMsg for each line of output, and BuildCompleteMsg when done.
func (r *Runner) Build() tea.Cmd {
	r.mu.Lock()
	if r.status == StatusBuilding {
		r.mu.Unlock()
		return nil
	}
	r.status = StatusBuilding
	r.output = nil
	r.result = nil
	projectPath := r.projectPath
	configuration := r.config
	r.mu.Unlock()

	return func() tea.Msg {
		start := time.Now()

		// Find .csproj file in the project directory.
		csprojPath, err := findCsproj(projectPath)
		if err != nil {
			r.mu.Lock()
			r.status = StatusFailed
			res := Result{
				Status:   StatusFailed,
				Duration: time.Since(start),
				Errors:   1,
			}
			r.result = &res
			r.mu.Unlock()
			return BuildCompleteMsg{Result: res}
		}

		cmd := exec.Command("dotnet", "build", csprojPath, "-c", configuration)
		cmd.Dir = projectPath

		r.mu.Lock()
		r.cmd = cmd
		r.mu.Unlock()

		// Capture stdout and stderr via pipes.
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return r.failBuild(start, 1)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return r.failBuild(start, 1)
		}

		if err := cmd.Start(); err != nil {
			return r.failBuild(start, 1)
		}

		// Read both pipes concurrently, collecting lines.
		var wg sync.WaitGroup
		var linesMu sync.Mutex
		var lines []OutputLine

		readPipe := func(pipe io.Reader, isErr bool) {
			defer wg.Done()
			scanner := bufio.NewScanner(pipe)
			for scanner.Scan() {
				text := scanner.Text()
				line := classifyLine(text, isErr)
				linesMu.Lock()
				lines = append(lines, line)
				linesMu.Unlock()
			}
		}

		wg.Add(2)
		go readPipe(stdout, false)
		go readPipe(stderr, true)
		wg.Wait()

		exitErr := cmd.Wait()
		duration := time.Since(start)

		// Count errors and warnings from collected output.
		errors := 0
		warnings := 0
		for _, l := range lines {
			lower := strings.ToLower(l.Text)
			if strings.Contains(lower, "error") && !strings.Contains(lower, "0 error") {
				errors++
			}
			if strings.Contains(lower, "warning") && !strings.Contains(lower, "0 warning") {
				warnings++
			}
		}

		status := StatusSuccess
		if exitErr != nil || errors > 0 {
			status = StatusFailed
		}

		res := Result{
			Status:   status,
			Duration: duration,
			Errors:   errors,
			Warnings: warnings,
		}

		r.mu.Lock()
		r.status = status
		r.output = lines
		r.result = &res
		r.cmd = nil
		r.mu.Unlock()

		return BuildCompleteMsg{Result: res}
	}
}

// Cancel aborts the running build process.
func (r *Runner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		r.cmd = nil
	}
	r.status = StatusIdle
}

// Status returns the current build status.
func (r *Runner) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// Output returns the collected build output lines.
func (r *Runner) Output() []OutputLine {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]OutputLine, len(r.output))
	copy(out, r.output)
	return out
}

// Result returns the result of the last build, or nil if no build has completed.
func (r *Runner) Result() *Result {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.result
}

// Deploy copies a built DLL to the game's deploy directory.
func (r *Runner) Deploy(dllPath, deployDir string) error {
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return fmt.Errorf("create deploy dir: %w", err)
	}

	src, err := os.Open(dllPath)
	if err != nil {
		return fmt.Errorf("open DLL: %w", err)
	}
	defer src.Close()

	dstPath := filepath.Join(deployDir, filepath.Base(dllPath))
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy DLL: %w", err)
	}

	return nil
}

// Reset returns the runner to idle with no output.
func (r *Runner) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = StatusIdle
	r.output = nil
	r.result = nil
}

// failBuild is a helper that records a failed build and returns a BuildCompleteMsg.
func (r *Runner) failBuild(start time.Time, errors int) BuildCompleteMsg {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = StatusFailed
	res := Result{
		Status:   StatusFailed,
		Duration: time.Since(start),
		Errors:   errors,
	}
	r.result = &res
	return BuildCompleteMsg{Result: res}
}

// findCsproj locates a .csproj file in the given directory.
func findCsproj(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read project dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".csproj") {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("no .csproj file found in %s", dir)
}

// classifyLine determines whether a line is an error, info, or regular output.
func classifyLine(text string, fromStderr bool) OutputLine {
	lower := strings.ToLower(text)
	line := OutputLine{Text: text}

	if fromStderr || strings.Contains(lower, ": error") || strings.Contains(lower, "failed") {
		line.IsError = true
	} else if strings.Contains(lower, "build succeeded") ||
		strings.Contains(lower, "restore complete") ||
		strings.Contains(lower, ": warning") ||
		strings.Contains(lower, "microsoft") ||
		strings.Contains(lower, "determining projects") {
		line.IsInfo = true
	}

	return line
}
