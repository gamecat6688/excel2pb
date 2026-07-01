using System;
using System.Collections.Generic;
using pb;

namespace Data
{
    public partial class {{.Name}}Model : Singleton<{{.Name}}Model>
    {
        private Dictionary<{{.KeyType}}, {{.Name}}> rows = new ();

        public void Init()
        {
            var cfg = Loader.Load(new {{.Name}}Config()) as {{.Name}}Config;
            foreach (var r in cfg.Records)
            {
                rows[{{if .MultiKey}}({{range $i, $k := .Keys}}{{if $i}}, {{end}}r.{{$k.Name}}{{end}}){{else}}r.{{.KeyName}}{{end}}] = r;
            }

            onInit();
        }

        private void onInit()
        {
            
        }
    
        public bool Has({{.KeyType}} key)
        {
            return rows.ContainsKey(key);
        }
        
        public {{.Name}} Get({{.KeyType}} key)
        {
            return rows[key];
        }
        
        public Dictionary<{{.KeyType}}, {{.Name}}> GetRows()
        {
            return rows;
        }
    }
}