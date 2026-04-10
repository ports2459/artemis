package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eden/artemis/internal/config"
)

type TemplateType int

const (
	TemplateBlank TemplateType = iota
	TemplateBepInEx
	TemplateMelonLoader
)

func (t TemplateType) String() string {
	switch t {
	case TemplateBepInEx:
		return "BepInEx"
	case TemplateMelonLoader:
		return "MelonLoader"
	default:
		return "Blank"
	}
}

func Scaffold(dir, name string, tmpl TemplateType) error {
	dirs := []string{"src"}
	if tmpl != TemplateBlank {
		dirs = append(dirs, "libs", "assets")
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			return err
		}
	}

	switch tmpl {
	case TemplateBlank:
		if err := writeFile(filepath.Join(dir, "src/Plugin.cs"), blankPlugin(name)); err != nil {
			return err
		}
	case TemplateBepInEx:
		if err := writeFile(filepath.Join(dir, "src/Plugin.cs"), bepinexPlugin(name)); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "src/Patches.cs"), bepinexPatches(name)); err != nil {
			return err
		}
	case TemplateMelonLoader:
		if err := writeFile(filepath.Join(dir, "src/Mod.cs"), melonMod(name)); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "src/Patches.cs"), melonPatches(name)); err != nil {
			return err
		}
	}

	if err := writeFile(filepath.Join(dir, name+".csproj"), csproj(name, tmpl)); err != nil {
		return err
	}

	cfg := config.ProjectConfig{
		Project: config.ProjectInfo{
			Name:    name,
			Version: "1.0.0",
		},
		Framework: frameworkConfig(tmpl),
		Build: config.BuildConfig{
			Configuration: "Debug",
		},
	}
	return cfg.Save(filepath.Join(dir, "artemis.toml"))
}

func frameworkConfig(tmpl TemplateType) config.FrameworkConfig {
	switch tmpl {
	case TemplateBepInEx:
		return config.FrameworkConfig{Type: "bepinex", Version: "5.4.22"}
	case TemplateMelonLoader:
		return config.FrameworkConfig{Type: "melonloader", Version: "0.6.1"}
	default:
		return config.FrameworkConfig{}
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func blankPlugin(name string) string {
	return fmt.Sprintf(`using UnityEngine;

namespace %s
{
    public class Plugin
    {
    }
}
`, name)
}

func bepinexPlugin(name string) string {
	return fmt.Sprintf(`using BepInEx;
using BepInEx.Logging;
using HarmonyLib;
using UnityEngine;

namespace %s
{
    [BepInPlugin("com.author.%s", "%s", "1.0.0")]
    public class Plugin : BaseUnityPlugin
    {
        private Harmony _harmony;

        private void Awake()
        {
            Logger.LogInfo("%s loaded!");
            _harmony = new Harmony("com.author.%s");
            _harmony.PatchAll();
        }
    }
}
`, name, name, name, name, name)
}

func bepinexPatches(name string) string {
	return fmt.Sprintf(`using HarmonyLib;

namespace %s
{
    [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    public class ExamplePatch
    {
        static void Postfix()
        {
        }
    }
}
`, name)
}

func melonMod(name string) string {
	return fmt.Sprintf(`using MelonLoader;
using HarmonyLib;
using UnityEngine;

[assembly: MelonInfo(typeof(%s.Mod), "%s", "1.0.0", "Author")]
[assembly: MelonGame(null, null)]

namespace %s
{
    public class Mod : MelonMod
    {
        public override void OnInitializeMelon()
        {
            LoggerInstance.Msg("%s loaded!");
        }
    }
}
`, name, name, name, name)
}

func melonPatches(name string) string {
	return fmt.Sprintf(`using HarmonyLib;

namespace %s
{
    [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    public class ExamplePatch
    {
        static void Postfix()
        {
        }
    }
}
`, name)
}

func csproj(name string, tmpl TemplateType) string {
	refs := ""
	switch tmpl {
	case TemplateBepInEx:
		refs = `
    <!-- Update these paths to point to your game's managed DLLs -->
    <Reference Include="BepInEx">
      <HintPath>libs/BepInEx.dll</HintPath>
    </Reference>
    <Reference Include="0Harmony">
      <HintPath>libs/0Harmony.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine">
      <HintPath>libs/UnityEngine.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine.CoreModule">
      <HintPath>libs/UnityEngine.CoreModule.dll</HintPath>
    </Reference>`
	case TemplateMelonLoader:
		refs = `
    <!-- Update these paths to point to your game's managed DLLs -->
    <Reference Include="MelonLoader">
      <HintPath>libs/MelonLoader.dll</HintPath>
    </Reference>
    <Reference Include="0Harmony">
      <HintPath>libs/0Harmony.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine">
      <HintPath>libs/UnityEngine.dll</HintPath>
    </Reference>
    <Reference Include="UnityEngine.CoreModule">
      <HintPath>libs/UnityEngine.CoreModule.dll</HintPath>
    </Reference>`
	}

	return fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <TargetFramework>net472</TargetFramework>
    <AssemblyName>%s</AssemblyName>
    <RootNamespace>%s</RootNamespace>
    <LangVersion>latest</LangVersion>
  </PropertyGroup>

  <ItemGroup>%s
  </ItemGroup>

</Project>
`, name, name, refs)
}
