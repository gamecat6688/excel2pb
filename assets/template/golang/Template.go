package tables

import (
	"fmt"
	"io/ioutil"
)

type {{StructName}} struct {
{% for Var in Vars %}
	// {{Var.Name}} {{Var.Desc}}
	{{Var.Name}} {{ToType(Var.Type)}}
{% endfor %}
}

{{ArraryStruct}}

func (m *{{StructName}}) GetKey() {{ToType(KeyType)}} {
	return m.{{KeyName}}
}

// 从流中读取数据
func (m *{{StructName}}) readFromStream(bs *Stream) {
{% for Var in Vars %}	
	m.{{Var.Name}} = {{ToReader(Var.Type)}}
{% endfor %}
}

// 表格管理器
type {{StructName}}Reader struct {
	datas map[{{ToType(KeyType)}}]*{{StructName}}
}

// 加载表格
func (t *{{StructName}}Reader) load(path string) bool {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("err:", err.Error())
		return false
	}

	s := NewStream(f)
	count := s.ReadInt32()

	for i := 0; i < int(count); i++ {
		data := &{{StructName}}{}
		data.readFromStream(s)
		t.datas[data.GetKey()] = data
	}

	return true
}

// 根据key获取数据的结构
func (t *{{StructName}}Reader)Find(key {{ToType(KeyType)}}) *{{StructName}} {
	v, ok := t.datas[key]
	if !ok {
		fmt.Println("not find data", key)
	}
	return v
}

// 获得个数
func (t *{{StructName}}Reader) Count() int {
	return len(t.datas)
}

// 获得数据列表
func (t *{{StructName}}Reader) GetDatas() (rv []*{{StructName}}) {
	for _,v := range t.datas {
		rv = append(rv, v)
	}
	return
}

var inner{{StructName}}Reader = &{{StructName}}Reader{datas:make(map[{{ToType(KeyType)}}]*{{StructName}})}
func Get{{StructName}}Reader() *{{StructName}}Reader {
	return inner{{StructName}}Reader
}

// 初始化时加载表格
func init() {
	Get{{StructName}}Reader().load(getTablePath() + "{{StructName}}_s.tbl")
}

