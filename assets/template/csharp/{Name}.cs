using System;
using System.Collections.Generic;
using pb;

namespace Data
{
    public class {{.Name}}Model : Singleton<{{.Name}}Model>
    {
        private Dictionary<{{.KeyType}}, {{.Name}}> rows = new ();

        public void Init()
        {
            var cfg = Loader.Load(new {{.Name}}Config()) as {{.Name}}Config;
            foreach (var r in cfg.Records)
            {
                rows[r.ID] = r;
            }

            onInit();
        }

        private void onInit()
        {
            
        }
    
        public bool Has({{.KeyType}} id)
        {
            return rows.ContainsKey(id);
        }
        
        public {{.Name}} Get({{.KeyType}} id)
        {
            return rows[id];
        }
        
        public Dictionary<{{.KeyType}}, {{.Name}}> GetRows()
        {
            return rows;
        }
    }
}