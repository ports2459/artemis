using HarmonyLib;

namespace IdleTesting
{
    [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    public class ExamplePatch
    {
        static void Postfix()
        {
        }
    }
}
