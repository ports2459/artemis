using BepInEx;
using BepInEx.Logging;
using HarmonyLib;
using UnityEngine;

namespace ULTRASHILL
{
    [BepInPlugin("com.author.ULTRASHILL", "ULTRASHILL", "1.0.0")]
    public class Plugin : BaseUnityPlugin
    {
        private Harmony _harmony;

        private void Awake()
        {
            Logger.LogInfo("ULTRASHILL loaded!");
            _harmony = new Harmony("com.author.ULTRASHILL");
            _harmony.PatchAll();
        }
    }
}
