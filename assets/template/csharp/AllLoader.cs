namespace Data
{
    public class AllLoader
    {
        public static void LoadAll()
        {
			{{range .Names}}{{.}}Model.inst.Init();
			{{end}}
		}
    }
}