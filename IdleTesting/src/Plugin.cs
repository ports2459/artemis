using BepInEx;
using BepInEx.Logging;
using HarmonyLib;
using UnityEngine;

namespace IdleTesting
{
    [BepInPlugin("com.author.IdleTesting", "IdleTesting", "1.0.0")]
    public class Plugin : BaseUnityPlugin
    {
        private Harmony _harmony;

        private void Awake()
        {
            Logger.LogInfo("IdleTesting loaded!");
            _harmony = new Harmony("com.author.IdleTesting");
            _harmony.PatchAll();
        }
    }
}
