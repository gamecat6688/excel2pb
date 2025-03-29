# excel2pb

excel导出protobuf二进制，并导出读取代码


# 表格工作簿命名说明，工作簿的名称就是导出的二进制或proto的名称。开头的工作簿用于辅助计算或备注，不会导出。

# 表头说明
```
 第1行 字段名称，这行可以插入批注，批注用来表示字段的特殊功能，类似grom的tag
 第2行 字段类型
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

# TODO List
- [ ] 支持多个主键
- [ ] 支持生成读取代码
- [ ] 支持多语言导出
- [x] 支持时间戳转换
