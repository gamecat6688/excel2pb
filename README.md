# excel2pb

excel导出protobuf二进制，并导出读取代码

# 依赖说明
```
需要安装protoc并加入path路径
golang的导出需要安装protoc-gen-go插件
```

# excel工作簿说明
```
表格工作簿命名说明，工作簿的名称就是导出的二进制或proto的名称，#开头的工作簿用于辅助计算或备注，不会导出。
一个表格可以包含多个sheet，也可以只有一个sheet。
```

# 表头说明
```
 第1行 字段名称，这行可以插入批注，批注用来表示字段的特殊功能，类似grom的tag
 第2行 字段类型，支持修饰符：
    1. pk：表示主键， 比如：pk int32
    2. unique：表示唯一性， 比如：unique int32
    3. repeated：表示数组
 第3行 导出过滤,c,cs,s c表示客户端，s表示服务器
 第4行 字段的中文说明
 第5行 数据的开始行
```

# Tag批注的功能说明
``` fk:Item.ID;index;
举例：
 1. fk 引用其他表的字段，类似mysql的外键(foreign key)
    fk:Item.ID;index;
    表示引用Item表的ID字段，index表示索引

 2. 字段类型
    目前支持的类型有：
    1. 基础类型：int, float, string, bool, timestamp, enum, struct
    2. 数组类型：repeated <typename>
```
# excel文件夹结构说明
## 支持一层文件夹结构
```
excel
├── 物品.xlsx
├── 商店.xlsx
├── 升级.xlsx
```

## 支持多层文件夹结构
```
excel
├── 物品
│   ├── 货币.xlsx
│   ├── 装备.xlsx
│   ├── 材料.xlsx
├── 商店.xlsx
├── 升级.xlsx
```



# TODO List
- [ ] 支持多个主键
- [ ] 支持生成读取代码
- [x] 支持多语言导出
- [x] 支持时间戳转换
