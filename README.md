# excel2pb

将 Excel 配置表导出为 protobuf 二进制数据，并生成对应的读取代码。

按客户端 / 服务器两个目标分别导出：`.proto` 结构文件、protobuf 二进制数据、编译后的 pb 代码、以及读取加载代码。

## 依赖

- 安装 [protoc](https://github.com/protocolbuffers/protobuf) 并加入 PATH。
- 各语言的 codegen 插件需自行安装，例如 golang 导出需要 `protoc-gen-go`。

## 构建与运行

```bash
go build -o excel2pb.exe main.go     # 或运行 build_windows.bat / build_all.bat（交叉编译）
./excel2pb.exe                        # 读取 config.yaml 和 assets/xls/，输出到 assets/out_*
```

主要配置项在 `config.yaml`（缺省时回退到内置默认值）：`ExcelDir` 输入目录、`TimeZone` 时区、`EnableI18n` 是否导出多语言、`LogLevel` 日志级别、`MaxProcess` 并发数，以及 `Outs` 中客户端 / 服务器各自的输出路径、包名和目标语言。

## 工作簿与 Sheet 命名

- **工作簿文件名** = 导出的二进制 / proto 名称。
- 一个工作簿可包含多个 Sheet，也可以只有一个。
- `#` 开头的 Sheet 用于辅助计算或备注，**不会导出**。
- `~` 开头的临时文件会被跳过。
- 以 `Enum` 结尾的 Sheet 会被当作枚举表导出。

## 目录结构

支持一层或多层文件夹嵌套：

```
excel
├── 物品
│   ├── 货币.xlsx
│   ├── 装备.xlsx
│   └── 材料.xlsx
├── 商店.xlsx
└── 升级.xlsx
```

## 表头格式

前 4 行为表头，第 5 行起为数据：

| 行 | 含义 |
| --- | --- |
| 第 1 行 | 字段名称。可插入单元格批注，批注表示字段的特殊功能，类似 gorm 的 tag |
| 第 2 行 | 字段类型，支持修饰符 |
| 第 3 行 | 导出过滤：`c` 客户端、`s` 服务器、`cs` 两者都导出 |
| 第 4 行 | 字段中文说明 |
| 第 5 行 | 数据开始行 |

### 字段类型（第 2 行）

支持的基础类型：`int32`、`int64`、`float`、`double`、`bool`、`string`、`timestamp`（时间戳，转 int64）、`i18n`（多语言，转 string key）、枚举、以及嵌套结构（类型名为另一个 Sheet 名）。

类型修饰符：

- `pk`：主键，具有唯一性，例如 `pk int32`
- `unique`：唯一性，例如 `unique int32`
- `repeated`：数组，例如 `repeated int32`

### 单元格批注 Tag（第 1 行）

批注打在字段名所在行，多个 tag 用 `;` 分隔，例如 `fk:Item.ID;index;`：

- `fk`：引用其他表的字段，类似 MySQL 外键（foreign key）。`fk:Item.ID` 表示引用 `Item` 表的 `ID` 字段，导出时会校验引用值是否存在。
- `index`：索引。

## 数据值编码

- 数组用 `;` 分隔元素，例如 `1;2;3`。
- 嵌套结构用 `|` 分隔单个元素内的各字段，`;` 分隔多个元素，例如 `100|1;101|2;103|5`。

## 多语言（i18n）

开启 `EnableI18n` 后，所有 `i18n` 类型字段的值会被汇总到单个 `I18N.xlsx`（key = `表名_字段名_主键值`），与已有文件合并；导出的二进制中写入的是 key 而非文本。

## TODO List

- [x] 支持多个主键
- [x] 支持生成读取代码
- [x] 支持多语言导出
- [x] 支持时间戳转换
