using System;
using System.Collections.Generic;
using pb;

namespace Data
{
    public class {{.Name}}Model : Singleton<{{.Name}}Model>
    {
        internal Dictionary<{{.KeyType}}, pb.{{.Name}}> rows = new Dictionary<{{.KeyType}}, pb.{{.Name}}>();

        public void Init()
        {
            var cfg = Loader.Load(new {{.Name}}Config()) as {{.Name}}onfig;
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
        
        public pb.{{.Name}} Get({{.KeyType}} id)
        {
            return rows[id];
        }
        
        public Dictionary<{{.KeyType}}, pb.{{.Name}}> GetRows()
        {
            return rows;
        }
    }
}