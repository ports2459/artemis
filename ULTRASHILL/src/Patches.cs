using HarmonyLib;

namespace ULTRASHILL
{
    [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    public class ExamplePatch
    {
        static void Postfix()
        {
        }
    }
}
