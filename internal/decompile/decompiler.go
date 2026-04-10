package decompile

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Assembly represents a .NET assembly (DLL) found in the game directory.
type Assembly struct {
	Name string
	Path string
	Size int64
}

// TypeEntry represents a type (class/struct/interface/enum) within an assembly.
type TypeEntry struct {
	FullName  string // e.g. "Namespace.ClassName"
	Namespace string
	Name      string
}

// Decompiler uses a bundled .NET helper (backed by ICSharpCode.Decompiler)
// to decompile .NET assemblies. The helper is built automatically on first use.
type Decompiler struct {
	helperDir string // ~/.cache/artemis/decompiler
	helperExe string // path to built DLL
	dotnet    string // path to dotnet CLI
	ready     bool
	buildErr  error
	once      sync.Once
}

// NewDecompiler creates a new Decompiler instance.
func NewDecompiler() *Decompiler {
	cacheDir := filepath.Join(cacheHome(), "artemis", "decompiler")
	dotnet, _ := exec.LookPath("dotnet")
	return &Decompiler{
		helperDir: cacheDir,
		dotnet:    dotnet,
	}
}

// Available returns true if dotnet is installed.
func (d *Decompiler) Available() bool {
	return d.dotnet != ""
}

// EnsureHelper builds the helper tool if it hasn't been built yet.
// Safe to call multiple times; the build only happens once.
func (d *Decompiler) EnsureHelper() error {
	d.once.Do(func() {
		d.buildErr = d.buildHelper()
		if d.buildErr == nil {
			d.ready = true
		}
	})
	return d.buildErr
}

// Ready returns true if the helper has been built successfully.
func (d *Decompiler) Ready() bool {
	return d.ready
}

// ScanAssemblies finds all .dll files in the given directory (typically Managed/).
func ScanAssemblies(dir string) ([]Assembly, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var assemblies []Assembly
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".dll") {
			continue
		}
		info, err := e.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		assemblies = append(assemblies, Assembly{
			Name: e.Name(),
			Path: filepath.Join(dir, e.Name()),
			Size: size,
		})
	}

	sort.Slice(assemblies, func(i, j int) bool {
		return assemblies[i].Name < assemblies[j].Name
	})

	return assemblies, nil
}

// FindManagedDir locates the Managed/ directory inside a Unity game folder.
func FindManagedDir(gamePath string) string {
	direct := filepath.Join(gamePath, "Managed")
	if isDir(direct) {
		return direct
	}

	entries, err := os.ReadDir(gamePath)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), "_Data") {
			managed := filepath.Join(gamePath, e.Name(), "Managed")
			if isDir(managed) {
				return managed
			}
		}
	}

	for _, sub := range []string{"Data/Managed", "GameData/Managed"} {
		p := filepath.Join(gamePath, sub)
		if isDir(p) {
			return p
		}
	}

	return ""
}

// ListTypes returns the types declared in an assembly.
func (d *Decompiler) ListTypes(assemblyPath string) ([]TypeEntry, error) {
	if err := d.EnsureHelper(); err != nil {
		return nil, err
	}

	out, err := d.runHelper("list-types", assemblyPath)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		FullName  string `json:"fullName"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		// Fallback: treat output as newline-delimited type names
		return parseTypeList(out), nil
	}

	types := make([]TypeEntry, len(raw))
	for i, r := range raw {
		types[i] = TypeEntry{FullName: r.FullName, Namespace: r.Namespace, Name: r.Name}
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].FullName < types[j].FullName
	})
	return types, nil
}

// DecompileAssembly decompiles an entire assembly to C# source.
func (d *Decompiler) DecompileAssembly(assemblyPath string) (string, error) {
	if err := d.EnsureHelper(); err != nil {
		return "", err
	}
	return d.runHelper("decompile", assemblyPath)
}

// DecompileType decompiles a specific type from an assembly.
func (d *Decompiler) DecompileType(assemblyPath, typeName string) (string, error) {
	if err := d.EnsureHelper(); err != nil {
		return "", err
	}
	return d.runHelper("decompile-type", assemblyPath, typeName)
}

// ---------------------------------------------------------------------------
// Helper tool management
// ---------------------------------------------------------------------------

func (d *Decompiler) buildHelper() error {
	if d.dotnet == "" {
		return fmt.Errorf("dotnet CLI not found on PATH")
	}

	if err := os.MkdirAll(d.helperDir, 0755); err != nil {
		return fmt.Errorf("create helper dir: %w", err)
	}

	// Check if already built
	dllPath := filepath.Join(d.helperDir, "bin", "Release", "net8.0", "ArtemisDecompiler.dll")
	if _, err := os.Stat(dllPath); err == nil {
		d.helperExe = dllPath
		return nil
	}
	// Also check net9.0 in case that's what's installed
	dllPath9 := filepath.Join(d.helperDir, "bin", "Release", "net9.0", "ArtemisDecompiler.dll")
	if _, err := os.Stat(dllPath9); err == nil {
		d.helperExe = dllPath9
		return nil
	}

	// Write project files
	if err := d.writeHelperProject(); err != nil {
		return err
	}

	// Build it
	cmd := exec.Command(d.dotnet, "build", "-c", "Release", "--nologo", "-v", "quiet")
	cmd.Dir = d.helperDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build helper failed: %w\n%s", err, string(out))
	}

	// Find the built DLL (check both net8.0 and net9.0)
	for _, tfm := range []string{"net9.0", "net8.0"} {
		p := filepath.Join(d.helperDir, "bin", "Release", tfm, "ArtemisDecompiler.dll")
		if _, err := os.Stat(p); err == nil {
			d.helperExe = p
			return nil
		}
	}

	return fmt.Errorf("helper DLL not found after build")
}

func (d *Decompiler) writeHelperProject() error {
	// Detect installed SDK version to pick the right TFM
	tfm := "net8.0"
	cmd := exec.Command(d.dotnet, "--version")
	if out, err := cmd.Output(); err == nil {
		ver := strings.TrimSpace(string(out))
		if strings.HasPrefix(ver, "9.") {
			tfm = "net9.0"
		}
	}

	csproj := fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>%s</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="ICSharpCode.Decompiler" Version="9.1.0.7988" />
  </ItemGroup>
</Project>
`, tfm)

	programCS := `using System;
using System.IO;
using System.Text.Json;
using ICSharpCode.Decompiler;
using ICSharpCode.Decompiler.CSharp;
using ICSharpCode.Decompiler.TypeSystem;

class Program
{
    static int Main(string[] args)
    {
        if (args.Length < 2)
        {
            Console.Error.WriteLine("Usage: ArtemisDecompiler <command> <assembly> [typeName]");
            Console.Error.WriteLine("Commands: list-types, decompile, decompile-type");
            return 1;
        }

        string command = args[0];
        string assemblyPath = args[1];

        if (!File.Exists(assemblyPath))
        {
            Console.Error.WriteLine($"Assembly not found: {assemblyPath}");
            return 1;
        }

        try
        {
            var settings = new DecompilerSettings(ICSharpCode.Decompiler.CSharp.LanguageVersion.Latest);
            var decompiler = new CSharpDecompiler(assemblyPath, settings);

            switch (command)
            {
                case "list-types":
                    ListTypes(decompiler);
                    break;
                case "decompile":
                    Console.Write(decompiler.DecompileWholeModuleAsString());
                    break;
                case "decompile-type":
                    if (args.Length < 3)
                    {
                        Console.Error.WriteLine("decompile-type requires a type name");
                        return 1;
                    }
                    DecompileType(decompiler, args[2]);
                    break;
                default:
                    Console.Error.WriteLine($"Unknown command: {command}");
                    return 1;
            }
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine($"Error: {ex.Message}");
            return 1;
        }

        return 0;
    }

    static void ListTypes(CSharpDecompiler decompiler)
    {
        var types = new System.Collections.Generic.List<object>();
        foreach (var type in decompiler.TypeSystem.MainModule.TypeDefinitions)
        {
            if (type.Name == "<Module>") continue;
            types.Add(new
            {
                fullName = type.FullName,
                @namespace = type.Namespace,
                name = type.Name
            });
        }
        Console.Write(JsonSerializer.Serialize(types));
    }

    static void DecompileType(CSharpDecompiler decompiler, string typeName)
    {
        // Try to find the type by full name
        ITypeDefinition found = null;
        foreach (var type in decompiler.TypeSystem.MainModule.TypeDefinitions)
        {
            if (type.FullName == typeName || type.ReflectionName == typeName)
            {
                found = type;
                break;
            }
        }

        if (found == null)
        {
            // Try partial match
            foreach (var type in decompiler.TypeSystem.MainModule.TypeDefinitions)
            {
                if (type.Name == typeName)
                {
                    found = type;
                    break;
                }
            }
        }

        if (found == null)
        {
            Console.Error.WriteLine($"Type not found: {typeName}");
            Console.Error.WriteLine("Decompiling full assembly instead.");
            Console.Write(decompiler.DecompileWholeModuleAsString());
            return;
        }

        Console.Write(decompiler.DecompileTypeAsString(found.FullTypeName));
    }
}
`

	if err := os.WriteFile(filepath.Join(d.helperDir, "ArtemisDecompiler.csproj"), []byte(csproj), 0644); err != nil {
		return fmt.Errorf("write csproj: %w", err)
	}
	if err := os.WriteFile(filepath.Join(d.helperDir, "Program.cs"), []byte(programCS), 0644); err != nil {
		return fmt.Errorf("write Program.cs: %w", err)
	}

	return nil
}

func (d *Decompiler) runHelper(args ...string) (string, error) {
	if d.helperExe == "" {
		return "", fmt.Errorf("helper not built")
	}

	cmdArgs := append([]string{d.helperExe}, args...)
	cmd := exec.Command(d.dotnet, cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}

	return string(out), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseTypeList(output string) []TypeEntry {
	var types []TypeEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := TypeEntry{FullName: line}
		if idx := strings.LastIndex(line, "."); idx >= 0 {
			entry.Namespace = line[:idx]
			entry.Name = line[idx+1:]
		} else {
			entry.Name = line
		}
		types = append(types, entry)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].FullName < types[j].FullName
	})
	return types
}

func cacheHome() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
