package assets

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AssetType classifies a Unity asset.
type AssetType int

const (
	TypeTexture2D  AssetType = iota
	TypeMaterial
	TypeGameObject
	TypeAudioClip
	TypeShader
	TypeTextAsset
	TypeOther
)

// String returns the human-readable name of an AssetType.
func (t AssetType) String() string {
	switch t {
	case TypeTexture2D:
		return "Texture2D"
	case TypeMaterial:
		return "Material"
	case TypeGameObject:
		return "GameObject"
	case TypeAudioClip:
		return "AudioClip"
	case TypeShader:
		return "Shader"
	case TypeTextAsset:
		return "TextAsset"
	default:
		return "Other"
	}
}

// ParseAssetType converts a Unity type name string to an AssetType.
func ParseAssetType(name string) AssetType {
	switch name {
	case "Texture2D":
		return TypeTexture2D
	case "Material":
		return TypeMaterial
	case "GameObject":
		return TypeGameObject
	case "AudioClip":
		return TypeAudioClip
	case "Shader":
		return TypeShader
	case "TextAsset":
		return TypeTextAsset
	default:
		return TypeOther
	}
}

// AssetInfo describes a single asset within a bundle.
type AssetInfo struct {
	Name     string    // display name (or path_id if unnamed)
	Type     AssetType // parsed type enum
	TypeName string    // raw type name from Unity
	Size     int64     // byte size
	Bundle   string    // which bundle file it came from
	PathID   int64     // unique ID within the bundle
}

// BundleInfo describes a Unity bundle/asset file and its contents.
type BundleInfo struct {
	Path   string      // full filesystem path
	Name   string      // basename
	Size   int64       // file size in bytes
	Assets []AssetInfo // contents (populated on demand via IndexBundle)
}

// Engine wraps UnityPy (Python) as a subprocess for reading Unity asset
// bundles. It degrades gracefully when UnityPy is not installed.
type Engine struct {
	gameDataPath string
	bundles      []BundleInfo
	available    bool   // whether python3 + UnityPy are importable
	pythonPath   string // path to working python3 binary
}

// NewEngine creates a new asset engine. Call SetGameDataPath and then Scan
// before using other methods.
func NewEngine() *Engine {
	e := &Engine{}
	e.pythonPath, e.available = e.findPython()
	return e
}

// SetGameDataPath sets the game data directory to scan for bundles.
func (e *Engine) SetGameDataPath(path string) {
	e.gameDataPath = path
	e.bundles = nil
}

// Available reports whether python3 and UnityPy are importable.
func (e *Engine) Available() bool {
	return e.available
}

// Bundles returns the list of discovered bundles from the last Scan.
func (e *Engine) Bundles() []BundleInfo {
	return e.bundles
}

// findPython tries multiple python paths to find one with UnityPy installed.
func (e *Engine) findPython() (string, bool) {
	candidates := []string{"python3", "/usr/bin/python3", "python", "/usr/bin/python"}
	for _, p := range candidates {
		cmd := exec.Command(p, "-c", "import UnityPy; print('ok')")
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "ok" {
			return p, true
		}
	}
	return "python3", false
}

// Scan discovers .assets and .bundle files under the game data path. When
// UnityPy is available it also indexes each bundle's contents; otherwise it
// returns stub entries with no asset list.
func (e *Engine) Scan() error {
	if e.gameDataPath == "" {
		return errors.New("no game data path configured")
	}

	info, err := os.Stat(e.gameDataPath)
	if err != nil {
		return fmt.Errorf("game data path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("game data path is not a directory: %s", e.gameDataPath)
	}

	// Walk the directory for bundle/asset files.
	var paths []string
	err = filepath.Walk(e.gameDataPath, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries
		}
		if fi.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".assets" || ext == ".bundle" || ext == ".resource" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	e.bundles = make([]BundleInfo, 0, len(paths))

	if e.available {
		for _, p := range paths {
			bi, scanErr := e.scanBundle(p)
			if scanErr != nil {
				// If UnityPy can't parse this file, add a stub entry.
				e.bundles = append(e.bundles, BundleInfo{
					Path: p,
					Name: filepath.Base(p),
				})
				continue
			}
			e.bundles = append(e.bundles, bi)
		}
	} else {
		// Fallback: list files without indexing contents.
		for _, p := range paths {
			e.bundles = append(e.bundles, BundleInfo{
				Path: p,
				Name: filepath.Base(p),
			})
		}
	}

	return nil
}

// ListFiles returns the list of .assets/.bundle/.resource files in the data path
// without scanning their contents.
func (e *Engine) ListFiles() ([]string, error) {
	if e.gameDataPath == "" {
		return nil, errors.New("no game data path configured")
	}
	info, err := os.Stat(e.gameDataPath)
	if err != nil {
		return nil, fmt.Errorf("game data path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", e.gameDataPath)
	}

	var paths []string
	err = filepath.Walk(e.gameDataPath, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil || fi.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".assets" || ext == ".bundle" || ext == ".resource" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// ScanAllBundles scans all given files in a single Python process for speed.
// Calls progressFn after each file with the result.
func (e *Engine) ScanAllBundles(files []string, progressFn func(index int, bi BundleInfo)) error {
	e.bundles = nil

	if !e.available || len(files) == 0 {
		for i, f := range files {
			bi := BundleInfo{Path: f, Name: filepath.Base(f)}
			e.bundles = append(e.bundles, bi)
			if progressFn != nil {
				progressFn(i, bi)
			}
		}
		return nil
	}

	// Build a single Python script that scans all files
	script := `
import UnityPy, json, sys

results = []
for path in sys.argv[1:]:
    try:
        env = UnityPy.load(path)
        assets = []
        for obj in env.objects:
            try:
                data = obj.read()
                name = str(getattr(data, 'name', '')) or str(getattr(data, 'm_Name', ''))
                if not name:
                    name = f"{obj.type.name}_{obj.path_id}"
            except:
                name = f"{obj.type.name}_{obj.path_id}"
            assets.append({
                "name": name,
                "type": obj.type.name,
                "size": obj.byte_size,
                "path_id": obj.path_id
            })
        results.append({"path": path, "assets": assets})
    except Exception as ex:
        results.append({"path": path, "assets": [], "error": str(ex)})
    # Flush progress line by line so Go can read incrementally
    print(json.dumps(results[-1]), flush=True)
`
	args := append([]string{"-c", script}, files...)
	cmd := exec.Command(e.pythonPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start python: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024) // 4MB buffer for large outputs
	idx := 0
	for scanner.Scan() {
		line := scanner.Text()
		var result struct {
			Path   string `json:"path"`
			Assets []struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Size   int64  `json:"size"`
				PathID int64  `json:"path_id"`
			} `json:"assets"`
		}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		bi := BundleInfo{
			Path: result.Path,
			Name: filepath.Base(result.Path),
		}
		for _, a := range result.Assets {
			bi.Assets = append(bi.Assets, AssetInfo{
				Name:     a.Name,
				Type:     ParseAssetType(a.Type),
				TypeName: a.Type,
				Size:     a.Size,
				Bundle:   result.Path,
				PathID:   a.PathID,
			})
		}
		e.bundles = append(e.bundles, bi)
		if progressFn != nil {
			progressFn(idx, bi)
		}
		idx++
	}

	return cmd.Wait()
}

// AddBundle appends a bundle to the engine's list.
func (e *Engine) AddBundle(bi BundleInfo) {
	e.bundles = append(e.bundles, bi)
}

// ClearBundles resets the bundle list.
func (e *Engine) ClearBundles() {
	e.bundles = nil
}

// SetBundles sets the bundle list directly.
func (e *Engine) SetBundles(bundles []BundleInfo) {
	e.bundles = bundles
}

// IndexBundle indexes a single bundle file with UnityPy and returns its assets.
// This is the slow operation — only called when the user selects a bundle.
func (e *Engine) IndexBundle(bundlePath string) ([]AssetInfo, error) {
	if !e.available {
		return nil, errors.New("UnityPy is not available")
	}
	bi, err := e.scanBundle(bundlePath)
	if err != nil {
		return nil, err
	}
	return bi.Assets, nil
}

// scanBundle uses UnityPy to index a single bundle file.
func (e *Engine) scanBundle(bundlePath string) (BundleInfo, error) {
	script := `
import UnityPy, json, sys
env = UnityPy.load(sys.argv[1])
assets = []
for obj in env.objects:
    try:
        data = obj.read()
        name = str(getattr(data, 'name', '')) or str(getattr(data, 'm_Name', ''))
        if not name:
            name = f"{obj.type.name}_{obj.path_id}"
    except:
        name = f"{obj.type.name}_{obj.path_id}"
    assets.append({
        "name": name,
        "type": obj.type.name,
        "size": obj.byte_size,
        "path_id": obj.path_id
    })
print(json.dumps(assets))
`
	cmd := exec.Command(e.pythonPath, "-c", script, bundlePath)
	out, err := cmd.Output()
	if err != nil {
		return BundleInfo{}, fmt.Errorf("UnityPy scan %s: %w", bundlePath, err)
	}

	var raw []struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Size   int64  `json:"size"`
		PathID int64  `json:"path_id"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return BundleInfo{}, fmt.Errorf("parsing UnityPy output for %s: %w", bundlePath, err)
	}

	bi := BundleInfo{
		Path:   bundlePath,
		Name:   filepath.Base(bundlePath),
		Assets: make([]AssetInfo, len(raw)),
	}
	for i, r := range raw {
		bi.Assets[i] = AssetInfo{
			Name:     r.Name,
			Type:     ParseAssetType(r.Type),
			TypeName: r.Type,
			Size:     r.Size,
			Bundle:   bundlePath,
			PathID:   r.PathID,
		}
	}
	return bi, nil
}

// ExportTexture exports a texture asset to a PNG file on disk.
func (e *Engine) ExportTexture(bundle string, pathID int64, outputPath string) error {
	if !e.available {
		return errors.New("UnityPy is not available")
	}

	script := fmt.Sprintf(`
import UnityPy, sys
try:
    env = UnityPy.load(sys.argv[1])
    for obj in env.objects:
        if obj.path_id == %d:
            data = obj.read()
            try:
                img = data.image
                img.save(sys.argv[2])
                print('ok')
                sys.exit(0)
            except Exception as e:
                print('export_error: ' + str(e))
                sys.exit(1)
    print('not_found')
    sys.exit(1)
except Exception as e:
    print('load_error: ' + str(e))
    sys.exit(1)
`, pathID)

	cmd := exec.Command(e.pythonPath, "-c", script, bundle, outputPath)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("export texture: %s", outStr)
	}
	if outStr != "ok" {
		return fmt.Errorf("texture export failed: %s", outStr)
	}
	return nil
}

// ReplaceTexture replaces a texture asset in a bundle with an image from disk.
func (e *Engine) ReplaceTexture(bundle string, pathID int64, inputPath string) error {
	if !e.available {
		return errors.New("UnityPy is not available")
	}

	script := fmt.Sprintf(`
import UnityPy, sys
from PIL import Image
env = UnityPy.load(sys.argv[1])
for obj in env.objects:
    if obj.path_id == %d:
        data = obj.read()
        if hasattr(data, 'image'):
            img = Image.open(sys.argv[2])
            data.image = img
            data.save()
            break
with open(sys.argv[1], 'wb') as f:
    f.write(env.file.save())
print('ok')
`, pathID)

	cmd := exec.Command(e.pythonPath, "-c", script, bundle, inputPath)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("replace texture: %w", err)
	}
	if strings.TrimSpace(string(out)) != "ok" {
		return errors.New("replacement failed")
	}
	return nil
}

// GetProperties reads all serialized properties of an asset.
func (e *Engine) GetProperties(bundle string, pathID int64) (map[string]interface{}, error) {
	if !e.available {
		return nil, errors.New("UnityPy is not available")
	}

	script := fmt.Sprintf(`
import UnityPy, json, sys
env = UnityPy.load(sys.argv[1])
for obj in env.objects:
    if obj.path_id == %d:
        data = obj.read()
        tree = obj.read_typetree()
        print(json.dumps(tree, default=str))
        sys.exit(0)
print('{}')
`, pathID)

	cmd := exec.Command(e.pythonPath, "-c", script, bundle)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get properties: %w", err)
	}

	var props map[string]interface{}
	if err := json.Unmarshal(out, &props); err != nil {
		return nil, fmt.Errorf("parsing properties: %w", err)
	}
	return props, nil
}

// SetProperty writes a single property value on an asset.
func (e *Engine) SetProperty(bundle string, pathID int64, key string, value interface{}) error {
	if !e.available {
		return errors.New("UnityPy is not available")
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	script := fmt.Sprintf(`
import UnityPy, json, sys
env = UnityPy.load(sys.argv[1])
for obj in env.objects:
    if obj.path_id == %d:
        tree = obj.read_typetree()
        tree[%q] = json.loads(%q)
        obj.save_typetree(tree)
        break
with open(sys.argv[1], 'wb') as f:
    f.write(env.file.save())
print('ok')
`, pathID, key, string(valueJSON))

	cmd := exec.Command(e.pythonPath, "-c", script, bundle)
	out, cmdErr := cmd.Output()
	if cmdErr != nil {
		return fmt.Errorf("set property: %w", cmdErr)
	}
	if strings.TrimSpace(string(out)) != "ok" {
		return errors.New("set property failed")
	}
	return nil
}
