# excel2pb

将 Excel 游戏配置导出为 `.proto`、protobuf 二进制数据、目标语言类型代码和配置加载代码。

支持 Go、C# 和 Godot 4（GDScript），字段可按客户端与服务器分别导出。

## 快速开始

安装 [protoc](https://github.com/protocolbuffers/protobuf) 及目标语言插件，并确保命令位于 `PATH`。Go 需要 `protoc-gen-go`；Godot 需要 `protoc-gen-gdscript` 和对应的 `addons/protobuf`。

```powershell
go test ./config ./parser ./works
go build -o excel2pb.exe .
.\excel2pb.exe
```

程序读取 `config.yaml` 和 `assets/xls/`，默认将结果写入 `assets/out_*`。

> 不要运行 `go test ./...` 或 `go build ./...`，因为 `assets/template/` 中的 `.go` 文件是模板，不是可直接编译的源码。

## 配置

主要配置项：

| 配置 | 说明 |
| --- | --- |
| `ExcelDir` | Excel 工作簿目录，支持递归扫描子目录 |
| `EnableI18n` | 是否合并并导出多语言配置 |
| `TimeZone` | `timestamp` 字段使用的时区，如 `+08:00` |
| `MaxProcess` | 最大并发任务数，`0` 使用运行时默认值 |
| `LogLevel` | `DEBUG`、`INFO`、`WARN` 或 `ERROR` |
| `Outs.Client/Server` | 要启用的导出端；至少配置一个，未配置的端不会生成任何产物 |
| `TplCodePaths` | 各语言模板目录 |
| `CodeOutPaths` | 各语言加载代码输出目录 |

Go 目标必须同时设置：

- `PackageName`：protobuf 包名，例如 `pbs`；
- `GoPackagePath`：完整 Go import，例如 `server/pbs`；
- `GoModulePath`：包含该 import 的 module，例如 `server`。

`ProtoImportPath` 当前必须留空。可以只配置 `Outs.Client`（单机或纯客户端项目）、只配置 `Outs.Server`，或同时配置两端。同时配置客户端与服务器时不能使用相同的 `CodeLanguage`，否则加载代码会写入同一目录。

## Excel 规范

工作簿只是 Sheet 容器。以 `~` 开头的临时工作簿和以 `#` 开头的 Sheet 不导出；枚举 Sheet 使用 `<Name>_Enum` 命名。

普通 Sheet、枚举名和字段名必须使用无下划线的 PascalCase，例如 `ItemType`、`ID`。

### 表头

普通数据 Sheet 的前四行为固定表头，数据从第 5 行开始：

| 行 | 内容 |
| --- | --- |
| 1 | 字段名；字段 tag 写在这一行单元格的普通批注中 |
| 2 | 类型及修饰符 |
| 3 | 导出端：`c`、`s` 或 `cs` |
| 4 | 字段说明 |

基础类型包括 `int32`、`int64`、`float`、`double`、`bool`、`string`、`timestamp` 和 `i18n`，也可以引用枚举名或其他 Sheet 作为嵌套 message。

常用修饰符：

- `pk int32`：主键；多个 `pk` 组成复合主键；
- `unique string`：字段值唯一；
- `repeated int32`：数组字段。

有数据的 Sheet 必须定义主键。主键不能为空、组合必须唯一，并且必须覆盖所有已配置的导出端；例如只配置 Client 时可以使用 `c` 或 `cs`，同时配置两端时必须使用 `cs`。

### 字段 tag

多个 tag 使用 `;` 分隔：

- `fk:Item.ID`：当前字段引用 `Item.ID`；
- `fk:Reward.ItemID=Item.ID`：嵌套结构中的外键；
- `index`：保留索引标记，当前尚未生成二级索引代码。

外键目标必须是 `unique` 字段或唯一的单主键，字段类型必须一致。数组和嵌套外键会逐项校验。

### 数据编码

- `timestamp`：`YYYY-MM-DD HH:MM:SS`，按 `TimeZone` 转为 Unix 秒；
- 数组：使用 `;`，例如 `1;2;3`；
- 嵌套结构：字段使用 `|`，多项使用 `;`，例如 `1001|2;1002|5`；
- 枚举：填写枚举名称或合法的 int32 值。

嵌套值必须按目标 Sheet 的完整字段顺序填写；导出时再按字段的 `c`、`s`、`cs` 过滤。

### 枚举与 i18n

枚举 Sheet 第一行为表头，之后每行依次填写名称、int32 值和可选说明。枚举不能为空，第一项必须为 `0`，名称和值均不可重复。

`i18n` 单元格填写源文本。开启多语言后，内容会合并到 `assets/xls/I18N.xlsx`，二进制数据写入 `Sheet_Field_主键值` 格式的 key。含 i18n 字段的 Sheet 必须有稳定主键，i18n 字段不能作为主键。

## 输出与模板

一次成功导出会生成：

- `ProtoPath`：`.proto` 文件；
- `DataPath`：protobuf 二进制数据；
- `PbPath`：`protoc` 生成的目标语言类型；
- `CodeOutPaths`：基于 `assets/template/<language>/` 生成的加载代码。

模板文件名不含 `{}` 时作为 Loader 执行一次；包含 `{name}` 或 `{Name}` 时按数据 Sheet 生成 Module。

输出目录必须由 excel2pb 专用。全部阶段成功后，程序会清理其中已失效的受管文件；不要在这些目录混放同扩展名的手写文件。

## 校验与排错

导出前会检查配置路径、命名、主键、唯一键、外键、标量类型、时间戳、枚举、嵌套字段数量和 i18n key 冲突。任一解析、模板、worker 或 `protoc` 任务失败，程序都会以失败状态退出。

排查失败时优先关注日志中的工作簿、Sheet、字段和 Excel 行号，并确认：

1. `protoc` 与目标语言插件可执行；
2. Loader 和 Module 模板同时存在；
3. 输出目录互不重叠，也不覆盖 Excel 或模板目录；
4. 生成文件存在且大小不为零。
