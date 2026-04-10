using HarmonyLib;

namespace lethal
{
    [HarmonyPatch(typeof(TargetType), "TargetMethod")]
    public class ExamplePatch
    {
        static void Postfix()
        {
        }
    }
}
