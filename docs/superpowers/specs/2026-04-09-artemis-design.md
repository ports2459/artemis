# Artemis: TUI IDE for Unity Modding

## Overview

Artemis is a terminal-based IDE for creating mods for Unity games. It's a single Go binary built with Bubbletea/Lipgloss that provides a full modding workbench: code editor with LSP, asset browser with terminal image support, build/deploy pipeline, and a Lua plugin system.

**Target audience:** Technical modders comfortable with terminals.

## Tech Stack

- **Language:** Go
- **TUI framework:** Bubbletea + Lipgloss
- **LSP server:** OmniSharp (C# intellisense, spawned as subprocess)
- **Asset parsing:** Shell out to UnityPy (Python) as primary tool, UABE as optional fallback
- **Plugin runtime:** GopherLua (pure-Go Lua 5.1 VM)
- **Config format:** TOML
- **Image display:** Kitty graphics protocol, sixel fallback, ASCII block character fallback

## Architecture

Single binary, monolith with modular internal packages. No IPC, no multi-process architecture.

```
artemis
├── App Shell
│   ├── Layout manager (panel switching, focus)
│   ├── Keybinding system (VS Code-style, no modal editing)
│   ├── Status bar
│   ├── Command palette (Ctrl+P)
│   └── Terminal capability detection
├── Panels
│   ├── Project Manager
│   ├── Code Editor
│   ├── Asset Browser
│   └── Build & Deploy
├── Core Services
│   ├── LSP Client (OmniSharp management)
│   ├── Asset Engine (Unity asset parsing/serialization)
│   ├── Mod Framework Abstraction (BepInEx, MelonLoader)
│   ├── Config Store (global + per-project TOML)
│   └── Lua Plugin Runtime
```

### App Shell

The app shell owns the screen. It manages panel switching (Ctrl+1-4), routes keyboard input to the focused panel, renders the shared top bar and status bar, and hosts the command palette overlay.

Panels communicate through Bubbletea's built-in Cmd/Msg system — a panel emits a Cmd, the app shell routes the resulting Msg to the appropriate handler.

## Panels

### Project Manager (Ctrl+1)

Displays a list of known mod projects and a "New Project" option. Projects are discovered from a configurable workspace directory.

**New project flow:**
1. User selects "New Project"
2. Choose: blank project or framework template (BepInEx / MelonLoader)
3. Enter project name
4. Artemis generates the project structure and opens it in the editor

**Project list shows:** Name, framework, last modified date. Navigate with arrow keys, Enter to open.

### Code Editor (Ctrl+2)

Full-featured code editor. No modal editing — VS Code-style keybindings throughout.

**Layout:**
- Left: file explorer tree (collapsible)
- Center: tabbed editor area with line numbers and syntax highlighting
- Right: diagnostics panel (errors/warnings) and symbol info

**Editor features:**
- Syntax highlighting for C# (primary), JSON, TOML, Lua
- Multi-tab editing
- Find/replace (Ctrl+F / Ctrl+H)
- Go to line (Ctrl+G)
- Undo/redo (Ctrl+Z / Ctrl+Y)
- Copy/cut/paste (Ctrl+C / Ctrl+X / Ctrl+V)
- Select next occurrence (Ctrl+D)
- Toggle comment (Ctrl+/)
- Indent/outdent (Tab / Shift+Tab)
- Word jump (Ctrl+Left/Right)
- Selection (Shift+Arrow, Ctrl+Shift+Arrow)

**LSP integration (via OmniSharp):**
- Real-time diagnostics (errors/warnings as you type)
- Autocomplete with type info (Ctrl+Space)
- Go to definition (F12)
- Find references (Shift+F12)
- Hover info (Ctrl+K Ctrl+I)
- Rename symbol (F2)
- Quick actions (Ctrl+.)

**File explorer:**
- Tree view of project files
- Arrow keys + Enter to navigate
- Keyboard shortcuts: `a` (new file), `d` (delete), `r` (rename)

### Asset Browser (Ctrl+3)

Browse, inspect, and edit Unity asset bundles from the game's data directory.

**Layout:**
- Left: asset bundle tree (list of .assets files and their contents by type)
- Center: asset grid showing previews
- Right: inspector panel with metadata and actions

**Asset types supported:**
- Texture2D — display real image previews via Kitty/sixel/ASCII
- Materials — show shader name and property values
- GameObjects/Prefabs — show component tree, allow add/remove components
- AudioClips — show duration, sample rate, allow playback info
- Shaders — show name, allow swapping
- Text assets — show content

**Actions per asset:**
- Export (e.g., texture to PNG, audio to WAV)
- Replace (swap in a new file)
- Edit properties (modify serialized values)
- View raw bytes

**Deep editing capabilities:**
- Modify prefabs: add/remove components, change component values
- Swap shaders on materials
- Replace textures
- Tweak numeric/string/enum values on any serialized asset
- No raw mesh manipulation (use Blender for that)

**Image display:**
Terminal capability detection at startup determines the image protocol:
1. Kitty graphics protocol — highest quality, inline images
2. Sixel — fallback for sixel-capable terminals (foot, xterm, WezTerm)
3. ASCII block characters (░▒▓) — fallback for basic terminals

### Build & Deploy (Ctrl+4)

Compile the mod and optionally copy it to the game directory.

**Configuration display:**
- Framework and version
- Game path
- Output path (where the DLL goes)
- Deploy directory (game's plugin folder)

**Actions:**
- Build (Ctrl+B) — runs `dotnet build` on the .csproj
- Build & Deploy (Ctrl+Shift+B) — builds then copies DLL to deploy directory
- Clean — removes build artifacts

**Build output:**
- Real-time streaming of stdout/stderr from the build process
- Errors parsed and cross-referenced with editor diagnostics
- Color-coded output (green for success, red for errors, blue for info)

## Core Services

### LSP Client

Manages the OmniSharp subprocess. Communicates over stdin/stdout using LSP JSON-RPC protocol.

**Lifecycle:**
1. Spawned when a project is opened
2. Initialized with the project root and .csproj path
3. Sends/receives LSP messages (didOpen, didChange, completion, definition, references, diagnostics, rename)
4. Debounces didChange notifications on keystrokes
5. Killed when project is closed or Artemis exits

### Asset Engine

Handles Unity asset bundle reading and writing. Shells out to UnityPy (Python) as the primary subprocess tool since no native Go Unity asset libraries exist. UABE can serve as an optional fallback.

**Flow:**
1. User configures game data path in artemis.toml
2. Asset engine scans for .assets and .bundle files
3. Indexes all assets by type (Texture2D, GameObject, Material, etc.)
4. On inspect: extracts asset data, decodes as needed (e.g., texture to PNG for display)
5. On replace/edit: patches the asset bundle via the subprocess tool

### Mod Framework Abstraction

Common interface over BepInEx and MelonLoader. Handles:
- Project scaffolding (correct folder structure, boilerplate code, manifest/metadata)
- Knowing where to deploy built DLLs based on framework conventions
- Resolving framework-specific assembly references for the .csproj
- Future framework support can be added by implementing the same interface

### Config Store

Two-tier TOML configuration:

**Global** (`~/.config/artemis/config.toml`):
- Default game paths
- Preferred framework
- Editor preferences (tab size, theme)
- Keybinding overrides
- Terminal image protocol override (auto/kitty/sixel/ascii)

**Per-project** (`artemis.toml` in project root):
- Project name, version, author
- Framework type and version
- Game path and deploy directory
- Build configuration

### Terminal Capability Detection

Runs at startup. Queries the terminal for:
- Image protocol support (Kitty, sixel, or none)
- True color support
- Unicode width support

Results inform the asset browser's image rendering and the UI's color/character choices.

## Lua Plugin System

Embeds GopherLua (pure-Go Lua 5.1 VM). No CGo, no external dependencies — compiles into the single binary.

### Plugin Discovery & Loading

**Plugin locations:**
- Global: `~/.config/artemis/plugins/<plugin-name>/init.lua`
- Per-project: `.artemis/plugins/<plugin-name>/init.lua`
- Init file: `~/.config/artemis/init.lua` (runs at startup, like nvim's init.lua)

**Loading order:**
1. Execute `~/.config/artemis/init.lua`
2. Scan and load global plugins
3. Scan and load project-local plugins
4. Fire `artemis_ready` event

Module resolution: `require("foo.bar")` searches plugin directories for `lua/foo/bar.lua` or `lua/foo/bar/init.lua`.

### API Surface

All exposed under a global `artemis` table:

| Namespace | Purpose |
|---|---|
| `artemis.editor` | Open/close files, get/set buffer content, cursor manipulation |
| `artemis.keymap` | Register/remove keybindings |
| `artemis.ui` | Add command palette entries, show notifications, register status bar items |
| `artemis.project` | Read project config, list files, get framework info |
| `artemis.build` | Trigger builds, read build output |
| `artemis.assets` | List/inspect/export assets programmatically |
| `artemis.hooks` | Register callbacks on events |
| `artemis.fs` | File system utilities |
| `artemis.config` | Read/write config values |

### Event System

Plugins register callbacks on named events with optional groups (prevents duplicate registration on reload):

```lua
artemis.hooks.on("on_save", {
  group = "my-plugin",
  callback = function(ev)
    -- ev.file, ev.buffer, etc.
  end,
})
```

**Available events:**
- `on_save`, `on_open`, `on_close` — file events
- `on_build_start`, `on_build_success`, `on_build_error` — build events
- `on_asset_change` — asset modification events
- `on_project_open`, `on_project_close` — project events
- `artemis_ready` — after all plugins loaded
- `on_command` — command palette execution

### Plugin Convention

Plugins expose a `setup(opts)` function:

```lua
-- ~/.config/artemis/plugins/my-plugin/init.lua
local M = {}

function M.setup(opts)
  artemis.keymap.set("ctrl+shift+h", function()
    -- custom action
  end, { desc = "My Plugin Action" })
end

return M
```

Users configure in their init.lua:

```lua
require("my-plugin").setup({
  some_option = true,
})
```

### No Sandboxing

Plugins have full access to the Lua stdlib and the artemis API. The target audience is technical modders who are trusted to install plugins they understand.

### No Plugin Manager (v1)

Users manually clone/copy plugins into the plugin directory. A plugin manager can be built as a plugin itself later.

## Keybindings

VS Code-style, no modal editing.

### Global
| Key | Action |
|---|---|
| Alt+1/2/3/4 | Switch panels |
| Ctrl+P | Command palette |
| Ctrl+S | Save |
| Ctrl+W | Close tab |
| Ctrl+Tab | Next tab |
| Ctrl+Q | Quit |

### Editor
| Key | Action |
|---|---|
| Arrow keys | Move cursor |
| Ctrl+Left/Right | Jump word |
| Shift+Arrow | Select |
| Ctrl+Shift+Arrow | Select by word |
| Home/End | Line start/end |
| Ctrl+Home/End | File start/end |
| Ctrl+C/X/V | Copy/cut/paste |
| Ctrl+Z/Y | Undo/redo |
| Ctrl+D | Select next occurrence |
| Ctrl+F | Find |
| Ctrl+H | Find & replace |
| Ctrl+G | Go to line |
| Ctrl+/ | Toggle comment |
| Tab/Shift+Tab | Indent/outdent |
| Ctrl+Space | Autocomplete |
| F2 | Rename symbol |
| F12 | Go to definition |
| Shift+F12 | Find references |
| Ctrl+. | Quick actions / hover info |

### File Explorer
| Key | Action |
|---|---|
| Arrow keys | Navigate tree |
| Enter | Open file / expand folder |
| a | New file |
| d | Delete |
| r | Rename |

### Asset Browser
| Key | Action |
|---|---|
| Arrow keys | Navigate grid |
| Enter | Inspect asset |
| e | Export |
| r | Replace |
| p | Edit properties |
| / | Search |

### Build Panel
| Key | Action |
|---|---|
| b | Build |
| d | Build & deploy |
| c | Clean |

All keybindings overridable via `artemis.keymap.set()` in Lua.

## Project Structure

### Blank Project
```
MyMod/
  src/
    Plugin.cs
  artemis.toml
  MyMod.csproj
```

### BepInEx Template
```
MyMod/
  src/
    Plugin.cs          -- BepInPlugin attribute, Harmony setup
    Patches.cs         -- example HarmonyPatch stub
  libs/
  assets/
  artemis.toml
  MyMod.csproj         -- references BepInEx, Harmony, UnityEngine
```

### MelonLoader Template
```
MyMod/
  src/
    Mod.cs             -- MelonMod entry point
    Patches.cs         -- HarmonyPatch stub
  libs/
  assets/
  artemis.toml
  MyMod.csproj         -- references MelonLoader, Harmony, Il2Cpp
```

## Data Flow

### Startup
1. Detect terminal capabilities (Kitty/sixel/truecolor)
2. Load global config from `~/.config/artemis/config.toml`
3. Load and execute `~/.config/artemis/init.lua`
4. Load global and project plugins
5. If launched with a project path: load `artemis.toml`, open editor panel, spawn OmniSharp
6. If launched bare: show project manager

### Editing
1. User opens file from tree → new editor tab
2. LSP client sends `textDocument/didOpen` to OmniSharp
3. On keystrokes, debounced `textDocument/didChange` sent
4. LSP responds with diagnostics → rendered in diagnostics panel
5. Autocomplete/definition/references on demand via keybindings

### Asset Editing
1. User configures game data path (one-time per project)
2. Asset engine subprocess scans bundles, indexes by type
3. Grid view shows assets; selecting a texture extracts to temp PNG and displays via Kitty/sixel
4. Replace/edit actions patch the bundle via subprocess

### Build & Deploy
1. User triggers build → Artemis runs `dotnet build` on the .csproj
2. Stdout/stderr streamed to output panel in real time
3. On success + deploy → DLL copied to configured game plugin folder
4. Build errors parsed and shown in editor diagnostics

## Non-Goals

- Raw mesh manipulation (use Blender)
- GUI/web frontend
- Plugin manager in v1
- Plugin sandboxing
- Game launching or log tailing
- Supporting non-Unity games
