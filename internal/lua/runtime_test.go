package lua

import (
	"os"
	"path/filepath"
	"testing"

	glua "github.com/yuin/gopher-lua"
)

func TestNewRuntime(t *testing.T) {
	r := NewRuntime()
	if r == nil {
		t.Fatal("NewRuntime returned nil")
	}
	if r.L == nil {
		t.Fatal("Lua state is nil")
	}
	r.Close()
	if r.L != nil {
		t.Fatal("Lua state should be nil after Close")
	}
}

func TestLoadLuaString(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`x = 1 + 1`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}

	val := r.L.GetGlobal("x")
	if val.Type() != glua.LTNumber {
		t.Fatalf("expected number, got %s", val.Type())
	}
	if glua.LVAsNumber(val) != 2 {
		t.Fatalf("expected 2, got %v", val)
	}
}

func TestHooksRegistration(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`
		artemis.hooks.on("on_save", {
			group = "test-plugin",
			callback = function(ev) end,
		})
	`)
	if err != nil {
		t.Fatalf("hook registration failed: %v", err)
	}

	hooks, ok := r.hooks["on_save"]
	if !ok {
		t.Fatal("no hooks registered for on_save")
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
	if hooks[0].Group != "test-plugin" {
		t.Fatalf("expected group 'test-plugin', got '%s'", hooks[0].Group)
	}
}

func TestHooksFireEvent(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`
		hook_called = false
		hook_event_name = ""
		hook_file_path = ""
		artemis.hooks.on("on_save", {
			group = "test",
			callback = function(ev)
				hook_called = true
				hook_event_name = ev.name
				hook_file_path = ev.path
			end,
		})
	`)
	if err != nil {
		t.Fatalf("hook setup failed: %v", err)
	}

	r.FireEvent("on_save", map[string]string{"path": "/tmp/test.cs"})

	called := r.L.GetGlobal("hook_called")
	if called != glua.LTrue {
		t.Fatal("hook callback was not called")
	}

	evName := r.L.GetGlobal("hook_event_name")
	if glua.LVAsString(evName) != "on_save" {
		t.Fatalf("expected event name 'on_save', got '%s'", glua.LVAsString(evName))
	}

	filePath := r.L.GetGlobal("hook_file_path")
	if glua.LVAsString(filePath) != "/tmp/test.cs" {
		t.Fatalf("expected path '/tmp/test.cs', got '%s'", glua.LVAsString(filePath))
	}
}

func TestHooksGroupReplacement(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	// Register first hook
	err := r.DoString(`
		call_count = 0
		artemis.hooks.on("on_save", {
			group = "my-plugin",
			callback = function(ev)
				call_count = call_count + 10
			end,
		})
	`)
	if err != nil {
		t.Fatalf("first hook registration failed: %v", err)
	}

	// Register second hook with same group -- should replace
	err = r.DoString(`
		artemis.hooks.on("on_save", {
			group = "my-plugin",
			callback = function(ev)
				call_count = call_count + 1
			end,
		})
	`)
	if err != nil {
		t.Fatalf("second hook registration failed: %v", err)
	}

	hooks := r.hooks["on_save"]
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook after group replacement, got %d", len(hooks))
	}

	// Fire and verify only the new callback runs
	r.FireEvent("on_save", nil)

	count := r.L.GetGlobal("call_count")
	if glua.LVAsNumber(count) != 1 {
		t.Fatalf("expected call_count=1 (new callback), got %v", count)
	}
}

func TestKeymapRegistration(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`
		artemis.keymap.set("ctrl+shift+h", function()
			-- action
		end, { desc = "Show help" })
	`)
	if err != nil {
		t.Fatalf("keymap set failed: %v", err)
	}

	bindings := r.Keybindings()
	if len(bindings) != 1 {
		t.Fatalf("expected 1 keybinding, got %d", len(bindings))
	}
	if bindings[0].Key != "ctrl+shift+h" {
		t.Fatalf("expected key 'ctrl+shift+h', got '%s'", bindings[0].Key)
	}
	if bindings[0].Desc != "Show help" {
		t.Fatalf("expected desc 'Show help', got '%s'", bindings[0].Desc)
	}
}

func TestConfigGetSet(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`
		artemis.config.set("editor.tab_size", 4)
		tab_size = artemis.config.get("editor.tab_size")
		missing = artemis.config.get("nonexistent.key")
	`)
	if err != nil {
		t.Fatalf("config get/set failed: %v", err)
	}

	tabSize := r.L.GetGlobal("tab_size")
	if glua.LVAsNumber(tabSize) != 4 {
		t.Fatalf("expected tab_size=4, got %v", tabSize)
	}

	missing := r.L.GetGlobal("missing")
	if missing != glua.LNil {
		t.Fatalf("expected nil for missing key, got %v", missing)
	}

	// Verify through Go side too
	val, ok := r.Config()["editor.tab_size"]
	if !ok {
		t.Fatal("config key not found in Go map")
	}
	if glua.LVAsNumber(val) != 4 {
		t.Fatalf("expected 4 in Go config map, got %v", val)
	}
}

func TestLogMessages(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`
		artemis.log.info("Plugin loaded!")
		artemis.log.warn("Something odd")
		artemis.log.error("Failed!")
	`)
	if err != nil {
		t.Fatalf("log calls failed: %v", err)
	}

	logs := r.Logs()
	if len(logs) != 3 {
		t.Fatalf("expected 3 log entries, got %d", len(logs))
	}

	// Check levels and messages
	checks := []struct {
		level LogLevel
		msg   string
	}{
		{LogInfo, "Plugin loaded!"},
		{LogWarn, "Something odd"},
		{LogError, "Failed!"},
	}
	for i, c := range checks {
		if logs[i].Level != c.level {
			t.Errorf("log[%d]: expected level %d, got %d", i, c.level, logs[i].Level)
		}
		if logs[i].Message != c.msg {
			t.Errorf("log[%d]: expected message %q, got %q", i, c.msg, logs[i].Message)
		}
		if logs[i].Time.IsZero() {
			t.Errorf("log[%d]: time should not be zero", i)
		}
	}
}

func TestLoadPluginFromDir(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	// Create a temp directory structure: plugins/test-plugin/init.lua
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	initLua := filepath.Join(pluginDir, "init.lua")
	code := `
		plugin_loaded = true
		artemis.hooks.on("on_open", {
			group = "test-plugin",
			callback = function(ev) end,
		})
		artemis.log.info("test-plugin loaded")
	`
	if err := os.WriteFile(initLua, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	// Also create a helper module to test require() support
	helperLua := filepath.Join(pluginDir, "helper.lua")
	helperCode := `
		local M = {}
		function M.greet() return "hello" end
		return M
	`
	if err := os.WriteFile(helperLua, []byte(helperCode), 0644); err != nil {
		t.Fatalf("failed to write helper.lua: %v", err)
	}

	err := r.LoadPlugins(pluginsDir)
	if err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	loaded := r.L.GetGlobal("plugin_loaded")
	if loaded != glua.LTrue {
		t.Fatal("plugin_loaded should be true")
	}

	hooks, ok := r.hooks["on_open"]
	if !ok || len(hooks) != 1 {
		t.Fatal("expected 1 hook registered by plugin")
	}

	logs := r.Logs()
	if len(logs) != 1 || logs[0].Message != "test-plugin loaded" {
		t.Fatalf("expected log from plugin, got %v", logs)
	}
}

func TestLoadInitFile(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	// Non-existent file should not error
	err := r.LoadInitFile("/tmp/nonexistent-artemis-init-xyz.lua")
	if err != nil {
		t.Fatalf("LoadInitFile should return nil for missing file, got: %v", err)
	}

	// Create a temp init.lua
	tmpDir := t.TempDir()
	initPath := filepath.Join(tmpDir, "init.lua")
	code := `init_loaded = true`
	if err := os.WriteFile(initPath, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	err = r.LoadInitFile(initPath)
	if err != nil {
		t.Fatalf("LoadInitFile failed: %v", err)
	}

	val := r.L.GetGlobal("init_loaded")
	if val != glua.LTrue {
		t.Fatal("init_loaded should be true")
	}
}

func TestEditorStubs(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	err := r.DoString(`
		artemis.editor.open("/tmp/test.cs")
		artemis.editor.save()
		local content = artemis.editor.get_content()
	`)
	if err != nil {
		t.Fatalf("editor stubs failed: %v", err)
	}

	cmds := r.EditorCommands()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 editor commands, got %d", len(cmds))
	}
	if cmds[0].Action != "open" || cmds[0].Args[0] != "/tmp/test.cs" {
		t.Errorf("unexpected open command: %+v", cmds[0])
	}
	if cmds[1].Action != "save" {
		t.Errorf("unexpected save command: %+v", cmds[1])
	}
	if cmds[2].Action != "get_content" {
		t.Errorf("unexpected get_content command: %+v", cmds[2])
	}

	r.ClearEditorCommands()
	if len(r.EditorCommands()) != 0 {
		t.Fatal("commands should be cleared")
	}
}

func TestKeymapWithoutDesc(t *testing.T) {
	r := NewRuntime()
	defer r.Close()

	// Keymap set without the optional desc table
	err := r.DoString(`
		artemis.keymap.set("ctrl+s", function() end)
	`)
	if err != nil {
		t.Fatalf("keymap without desc failed: %v", err)
	}

	bindings := r.Keybindings()
	if len(bindings) != 1 {
		t.Fatalf("expected 1 keybinding, got %d", len(bindings))
	}
	if bindings[0].Key != "ctrl+s" {
		t.Fatalf("expected key 'ctrl+s', got '%s'", bindings[0].Key)
	}
	if bindings[0].Desc != "" {
		t.Fatalf("expected empty desc, got '%s'", bindings[0].Desc)
	}
}
