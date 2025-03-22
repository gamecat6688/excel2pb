using System;
using System.Collections.Generic;
using System.Text;
using System.Collections;

namespace Store
{
	public class {{StructName}}
	{
		{% for Var in Vars %}
		// {{Var.Name}} {{Var.Desc}}
		protected {{ToType(Var.Type)}} {{Var.LowerName}};
		{% if Var.Type == "i18n" %}
		public string {{Var.Name}} { get { 
			return i18n.translate({{Var.LowerName}}.Key); 
		} } 
		{% else %}
		public {{ToType(Var.Type)}} {{Var.Name}} { get { 
			return {{Var.LowerName}}; 
		} } 
		{% endif %}
		{% endfor %}

		{{ArraryStruct}}

		public {{ToType(KeyType)}} GetKey()
		{
			return {{KeyName}};
		}

		public void ReadFromStream(BinStream bs)
		{
			{% for Var in Vars %}	
			{{Var.LowerName}} = {{ToReader(Var.Type)}};
			{% endfor %}
		}
	}

	public class {{StructName}}Reader : TableReader<{{StructName}}>
	{
		private static {{StructName}}Reader _Instance;

		public static {{StructName}}Reader Instance()
		{
			if (_Instance == null)
			{
				_Instance = new {{StructName}}Reader();
			}
			return _Instance;
		}

		public {{StructName}}Reader()
		{
			InitializeX();
		}

		public bool InitializeX()
		{
			string path = TablePath.GetTablePath() + "{{StructName}}_c.tbl";
			if (!LoadData(path))
			{
				return true;
			}

			return false;
		}

		public override void onLoadComplete()
		{
			int count = mBinStream.ReadInt32();
			for (int i = 0; i < count; i++)
			{
				{{StructName}} item = new {{StructName}}();
				item.ReadFromStream(mBinStream);
				mDatas[item.GetKey().ToString()] = item;
				mDataList.Add(item);
			}
			mBinStream = null;
		}

	}

}
