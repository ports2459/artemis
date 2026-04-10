using BepInEx;
using BepInEx.Logging;
using HarmonyLib;
using UnityEngine;

namespace lethal
{
    [BepInPlugin("com.author.lethal", "lethal", "1.0.0")]
    public class Plugin : BaseUnityPlugin
    {
        private Harmony _harmony;

        private void Awake()
        {
            Logger.LogInfo("lethal loaded!");
            _harmony = new Harmony("com.author.lethal");
            _harmony.PatchAll();
        }
    }
}
