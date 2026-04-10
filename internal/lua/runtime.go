package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarn
	LogError
)

// LogEntry holds a single log message from a Lua plugin.
type LogEntry struct {
	Level   LogLevel
	Message string
	Time    time.Time
}

// EditorCommand represents a deferred editor action requested by Lua.
type EditorCommand struct {
	Action string // "open", "save", "get_content"
	Args   []string
}

// Hook is a registered Lua callback for a named event.
type Hook struct {
	Group    string
	Callback *lua.LFunction
}

// LuaKeybinding is a keybinding registered from Lua.
type LuaKeybinding struct {
	Key      string
	Callback *lua.LFunction
	Desc     string
}

// Runtime manages the Lua VM and plugin lifecycle.
type Runtime struct {
	L              *lua.LState
	pluginDirs     []string
	hooks          map[string][]Hook
	keybindings    []LuaKeybinding
	config         map[string]lua.LValue
	logs           []LogEntry
	editorCommands []EditorCommand
}

// NewRuntime creates a new Lua runtime with the artemis API registered.
func NewRuntime() *Runtime {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: false,
	})

	r := &Runtime{
		L:      L,
		hooks:  make(map[string][]Hook),
		config: make(map[string]lua.LValue),
	}

	r.registerAPI()
	return r
}

// Close shuts down the Lua VM and releases resources.
func (r *Runtime) Close() {
	if r.L != nil {
		r.L.Close()
		r.L = nil
	}
}

// LoadInitFile loads and executes a Lua file (e.g. ~/.config/artemis/init.lua).
// Returns nil if the file does not exist.
func (r *Runtime) LoadInitFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return r.L.DoFile(path)
}

// LoadPlugins scans directories for plugin/init.lua files and loads them.
// Each plugin's directory is added to Lua's package.path so require() works.
func (r *Runtime) LoadPlugins(dirs ...string) error {
	var errs []string
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			errs = append(errs, fmt.Sprintf("reading %s: %v", dir, err))
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginDir := filepath.Join(dir, entry.Name())
			initFile := filepath.Join(pluginDir, "init.lua")

			if _, err := os.Stat(initFile); os.IsNotExist(err) {
				continue
			}

			// Add plugin directory to package.path
			r.addToPackagePath(pluginDir)
			r.pluginDirs = append(r.pluginDirs, pluginDir)

			if err := r.L.DoFile(initFile); err != nil {
				errs = append(errs, fmt.Sprintf("loading %s: %v", initFile, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("plugin errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// addToPackagePath adds a directory to Lua's package.path.
func (r *Runtime) addToPackagePath(dir string) {
	pkg := r.L.GetGlobal("package")
	if tbl, ok := pkg.(*lua.LTable); ok {
		currentPath := lua.LVAsString(tbl.RawGetString("path"))
		newEntry := filepath.Join(dir, "?.lua")
		newEntryInit := filepath.Join(dir, "?", "init.lua")
		tbl.RawSetString("path", lua.LString(currentPath+";"+newEntry+";"+newEntryInit))
	}
}

// FireEvent calls all registered hooks for the named event.
func (r *Runtime) FireEvent(name string, data map[string]string) {
	hooks, ok := r.hooks[name]
	if !ok {
		return
	}

	// Build event table from data
	evTable := r.L.NewTable()
	evTable.RawSetString("name", lua.LString(name))
	for k, v := range data {
		evTable.RawSetString(k, lua.LString(v))
	}

	for _, hook := range hooks {
		if err := r.L.CallByParam(lua.P{
			Fn:      hook.Callback,
			NRet:    0,
			Protect: true,
		}, evTable); err != nil {
			r.logs = append(r.logs, LogEntry{
				Level:   LogError,
				Message: fmt.Sprintf("hook %s/%s error: %v", name, hook.Group, err),
				Time:    time.Now(),
			})
		}
	}
}

// Keybindings returns all keybindings registered from Lua.
func (r *Runtime) Keybindings() []LuaKeybinding {
	return r.keybindings
}

// Logs returns all log entries captured from Lua plugins.
func (r *Runtime) Logs() []LogEntry {
	return r.logs
}

// EditorCommands returns all pending editor commands from Lua plugins.
func (r *Runtime) EditorCommands() []EditorCommand {
	return r.editorCommands
}

// ClearEditorCommands clears the pending editor command queue.
func (r *Runtime) ClearEditorCommands() {
	r.editorCommands = nil
}

// removeHooksByGroup removes all hooks for a given event that belong to a group.
func (r *Runtime) removeHooksByGroup(event, group string) {
	if group == "" {
		return
	}
	hooks, ok := r.hooks[event]
	if !ok {
		return
	}
	filtered := make([]Hook, 0, len(hooks))
	for _, h := range hooks {
		if h.Group != group {
			filtered = append(filtered, h)
		}
	}
	r.hooks[event] = filtered
}

// DoString executes a Lua string in the runtime. Useful for testing.
func (r *Runtime) DoString(code string) error {
	return r.L.DoString(code)
}

// Config returns the current configuration map (for inspection/testing).
func (r *Runtime) Config() map[string]lua.LValue {
	return r.config
}
