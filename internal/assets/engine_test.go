package assets

import (
	"testing"
)

func TestEngineNew(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	// available may be true or false depending on system; just verify it runs
	_ = e.Available()
}

func TestEngineScanWithoutPath(t *testing.T) {
	e := NewEngine()
	err := e.Scan()
	if err == nil {
		t.Fatal("expected error when scanning without a configured path")
	}
	if err.Error() != "no game data path configured" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestEngineScanNonexistentPath(t *testing.T) {
	e := NewEngine()
	e.SetGameDataPath("/tmp/artemis_nonexistent_path_for_test")
	err := e.Scan()
	if err == nil {
		t.Fatal("expected error when scanning a nonexistent path")
	}
}

func TestEngineScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	e := NewEngine()
	e.SetGameDataPath(dir)
	err := e.Scan()
	if err != nil {
		t.Fatalf("unexpected error scanning empty dir: %v", err)
	}
	if len(e.Bundles()) != 0 {
		t.Fatalf("expected 0 bundles, got %d", len(e.Bundles()))
	}
}

func TestAssetTypeString(t *testing.T) {
	tests := []struct {
		at   AssetType
		want string
	}{
		{TypeTexture2D, "Texture2D"},
		{TypeMaterial, "Material"},
		{TypeGameObject, "GameObject"},
		{TypeAudioClip, "AudioClip"},
		{TypeShader, "Shader"},
		{TypeTextAsset, "TextAsset"},
		{TypeOther, "Other"},
		{AssetType(99), "Other"},
	}
	for _, tc := range tests {
		got := tc.at.String()
		if got != tc.want {
			t.Errorf("AssetType(%d).String() = %q, want %q", int(tc.at), got, tc.want)
		}
	}
}

func TestParseAssetType(t *testing.T) {
	tests := []struct {
		input string
		want  AssetType
	}{
		{"Texture2D", TypeTexture2D},
		{"Material", TypeMaterial},
		{"GameObject", TypeGameObject},
		{"AudioClip", TypeAudioClip},
		{"Shader", TypeShader},
		{"TextAsset", TypeTextAsset},
		{"MonoBehaviour", TypeOther},
		{"", TypeOther},
	}
	for _, tc := range tests {
		got := ParseAssetType(tc.input)
		if got != tc.want {
			t.Errorf("ParseAssetType(%q) = %d, want %d", tc.input, int(got), int(tc.want))
		}
	}
}

func TestSetGameDataPathResetsBundles(t *testing.T) {
	e := NewEngine()
	e.bundles = []BundleInfo{{Name: "test"}}
	e.SetGameDataPath("/some/path")
	if len(e.Bundles()) != 0 {
		t.Fatal("expected bundles to be cleared after SetGameDataPath")
	}
}

func TestEngineMethodsWithoutUnityPy(t *testing.T) {
	e := &Engine{available: false}

	if err := e.ExportTexture("b", 1, "out.png"); err == nil {
		t.Error("ExportTexture should fail without UnityPy")
	}
	if err := e.ReplaceTexture("b", 1, "in.png"); err == nil {
		t.Error("ReplaceTexture should fail without UnityPy")
	}
	if _, err := e.GetProperties("b", 1); err == nil {
		t.Error("GetProperties should fail without UnityPy")
	}
	if err := e.SetProperty("b", 1, "key", "val"); err == nil {
		t.Error("SetProperty should fail without UnityPy")
	}
}
