package lua

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

// registerAPI sets up the "artemis" global table with all sub-modules.
func (r *Runtime) registerAPI() {
	artemis := r.L.NewTable()

	// artemis.hooks
	hooks := r.L.NewTable()
	hooks.RawSetString("on", r.L.NewFunction(r.apiHooksOn))
	artemis.RawSetString("hooks", hooks)

	// artemis.keymap
	keymap := r.L.NewTable()
	keymap.RawSetString("set", r.L.NewFunction(r.apiKeymapSet))
	artemis.RawSetString("keymap", keymap)

	// artemis.config
	config := r.L.NewTable()
	config.RawSetString("get", r.L.NewFunction(r.apiConfigGet))
	config.RawSetString("set", r.L.NewFunction(r.apiConfigSet))
	artemis.RawSetString("config", config)

	// artemis.log
	log := r.L.NewTable()
	log.RawSetString("info", r.L.NewFunction(r.apiLogInfo))
	log.RawSetString("warn", r.L.NewFunction(r.apiLogWarn))
	log.RawSetString("error", r.L.NewFunction(r.apiLogError))
	artemis.RawSetString("log", log)

	// artemis.editor (stubs)
	editor := r.L.NewTable()
	editor.RawSetString("open", r.L.NewFunction(r.apiEditorOpen))
	editor.RawSetString("save", r.L.NewFunction(r.apiEditorSave))
	editor.RawSetString("get_content", r.L.NewFunction(r.apiEditorGetContent))
	artemis.RawSetString("editor", editor)

	r.L.SetGlobal("artemis", artemis)
}

// --- artemis.hooks ---

// apiHooksOn registers a hook for a named event.
// Lua: artemis.hooks.on("event_name", { group = "group-name", callback = function(ev) ... end })
func (r *Runtime) apiHooksOn(L *lua.LState) int {
	event := L.CheckString(1)
	opts := L.CheckTable(2)

	group := ""
	if g := opts.RawGetString("group"); g != lua.LNil {
		group = g.String()
	}

	cb := opts.RawGetString("callback")
	fn, ok := cb.(*lua.LFunction)
	if !ok {
		L.ArgError(2, "callback must be a function")
		return 0
	}

	// Remove existing hooks with same group+event (group replacement)
	r.removeHooksByGroup(event, group)

	r.hooks[event] = append(r.hooks[event], Hook{
		Group:    group,
		Callback: fn,
	})

	return 0
}

// --- artemis.keymap ---

// apiKeymapSet registers a keybinding.
// Lua: artemis.keymap.set("ctrl+shift+h", function() ... end, { desc = "My action" })
func (r *Runtime) apiKeymapSet(L *lua.LState) int {
	key := L.CheckString(1)
	fn := L.CheckFunction(2)

	desc := ""
	if L.GetTop() >= 3 {
		opts := L.CheckTable(3)
		if d := opts.RawGetString("desc"); d != lua.LNil {
			desc = d.String()
		}
	}

	r.keybindings = append(r.keybindings, LuaKeybinding{
		Key:      key,
		Callback: fn,
		Desc:     desc,
	})

	return 0
}

// --- artemis.config ---

// apiConfigGet retrieves a configuration value.
// Lua: local val = artemis.config.get("editor.tab_size")
func (r *Runtime) apiConfigGet(L *lua.LState) int {
	key := L.CheckString(1)
	val, ok := r.config[key]
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(val)
	return 1
}

// apiConfigSet sets a configuration value.
// Lua: artemis.config.set("editor.tab_size", 2)
func (r *Runtime) apiConfigSet(L *lua.LState) int {
	key := L.CheckString(1)
	val := L.Get(2)
	r.config[key] = val
	return 0
}

// --- artemis.log ---

func (r *Runtime) apiLogInfo(L *lua.LState) int {
	msg := L.CheckString(1)
	r.logs = append(r.logs, LogEntry{
		Level:   LogInfo,
		Message: msg,
		Time:    time.Now(),
	})
	return 0
}

func (r *Runtime) apiLogWarn(L *lua.LState) int {
	msg := L.CheckString(1)
	r.logs = append(r.logs, LogEntry{
		Level:   LogWarn,
		Message: msg,
		Time:    time.Now(),
	})
	return 0
}

func (r *Runtime) apiLogError(L *lua.LState) int {
	msg := L.CheckString(1)
	r.logs = append(r.logs, LogEntry{
		Level:   LogError,
		Message: msg,
		Time:    time.Now(),
	})
	return 0
}

// --- artemis.editor (stubs) ---

// apiEditorOpen queues a command to open a file.
// Lua: artemis.editor.open("/path/to/file")
func (r *Runtime) apiEditorOpen(L *lua.LState) int {
	path := L.CheckString(1)
	r.editorCommands = append(r.editorCommands, EditorCommand{
		Action: "open",
		Args:   []string{path},
	})
	return 0
}

// apiEditorSave queues a command to save the active file.
// Lua: artemis.editor.save()
func (r *Runtime) apiEditorSave(L *lua.LState) int {
	r.editorCommands = append(r.editorCommands, EditorCommand{
		Action: "save",
	})
	return 0
}

// apiEditorGetContent queues a command to get the active buffer content.
// Lua: artemis.editor.get_content()
func (r *Runtime) apiEditorGetContent(L *lua.LState) int {
	r.editorCommands = append(r.editorCommands, EditorCommand{
		Action: "get_content",
	})
	// Return empty string as a stub; actual content would be injected by the app shell.
	L.Push(lua.LString(""))
	return 1
}
