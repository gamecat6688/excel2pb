# excel2pb

`excel2pb` 是一个游戏配置导出工具：读取 Excel 工作簿，按客户端（`client`）和服务器（`server`）的字段过滤规则，生成 `.proto`、protobuf 二进制数据、经 `protoc` 编译的类型代码，以及基于模板生成的配置加载代码。

当前支持的加载代码语言为 `golang`、`csharp` 和 Godot 4 / `godot`（GDScript）。

## 依赖

- 安装 [protoc](https://github.com/protocolbuffers/protobuf) 并加入 PATH。
- 安装目标语言对应的 protoc 插件。例如 Go 导出需要 `protoc-gen-go`。
- Godot / GDScript 导出需要安装 [protoc-gen-gdscript-simple](https://github.com/lixi1983/protoc-gen-gdscript-simple)，确保可执行文件名为 `protoc-gen-gdscript` 并已加入 PATH；同时将该项目的 `addons/protobuf` 复制到 Godot 4 项目的 `addons/protobuf`。

## 构建、测试与运行

```powershell
go build -o excel2pb.exe main.go
go test ./parser/ ./works/ ./config/
.\excel2pb.exe
```

- `excel2pb.exe` 读取 `config.yaml` 和 `assets/xls/`，默认输出到 `assets/out_*`。
- 不要运行 `go test ./...` 或 `go build ./...`：`assets/template/` 下的 `.go` 文件是 Go `text/template` 模板，不是可直接编译的源码。
- `build_windows.bat` 构建 Windows 可执行文件；`build_all.bat` 分别使用 `GOOS=linux`、`windows`、`darwin` 交叉构建 `excel2pb`、`excel2pb.exe` 和 `excel2pb_mac`。
- 运行完整导出前，请确认 `protoc` 和目标语言插件均可用。导出失败会返回非零状态；成功后仍应检查实际输出文件不为空。

## 配置

主要配置位于 `config.yaml`；文件缺失时回退到 `config/config.go` 中的内置默认配置。

- `ExcelDir`：Excel 输入目录。
- `ProtoImportPath`：当前必须留空；生成的 proto 之间使用同目录相对导入。
- `TimeZone`：解析时间戳使用的时区。
- `EnableI18n`：是否生成多语言表。
- `LogLevel`：日志级别。
- `MaxProcess`：并发 worker 数量。
- `Outs.Client` / `Outs.Server`：两端各自的 `ProtoPath`、`PbPath`、`DataPath`、`DataExt`、`PackageName` 和 `CodeLanguage`。Go 目标还必须配置 `GoPackagePath`（例如 `server/pbs`）与包含它的 `GoModulePath`（例如 `server`）；前者同时用于 proto 的 `go_package` 和加载代码的 import，后者用于 `protoc-gen-go` 计算相对输出目录。
- `TplCodePaths` / `CodeOutPaths`：各语言的模板目录和加载代码输出目录。

客户端和服务器的 `CodeLanguage` 不能相同：`CodeOutPaths` 按语言配置，同语言会写入同一目录并造成两端代码互相覆盖，程序会在导出前拒绝这种配置。

表头第 3 行的 `c`、`s`、`cs` 决定字段导出到客户端、服务器或两端。

## 导出流程与输出清理

主流程为 `ParseExcels() -> MergeI18n() -> Export()`。其中 `Export()` 依次执行：

1. 配置校验；
2. 生成 proto；
3. 生成 protobuf 二进制数据；
4. 根据模板生成加载代码；
5. 调用 `protoc` 生成目标语言类型代码；
6. 全部成功后清理已失效的生成物。

输出目录应作为 excel2pb 的专用生成目录。清理阶段只在导出成功后执行，并清理由工具管理、但本次不再生成的 `.proto`、数据文件、pb 代码和模板代码；不要在这些目录中混放同扩展名的手写文件。

工作簿解析、模板生成、并发 worker 或 `protoc` 任一阶段失败时，程序都会失败退出，不会把不完整导出视为成功。

## Godot 4 / GDScript

将目标的 `CodeLanguage` 设为 `godot` 即可生成：

- `PbPath`：`protoc-gen-gdscript` 生成的 `*.proto.gd` protobuf 类型代码。
- `DataPath`：Excel 数据对应的 protobuf 二进制文件。
- `CodeOutPaths.godot`：每张表的 `*Model.gd` 与汇总入口 `game_data.gd`。

示例：

```yaml
Outs:
  Client:
    ProtoPath: game/protobuf/proto/
    PbPath: game/protobuf/generated/
    DataPath: game/config/data/
    DataExt: ".bytes"
    PackageName: "pb"
    CodeLanguage: "godot"

TplCodePaths:
  godot: assets/template/godot/

CodeOutPaths:
  godot: game/config/generated/
```

`PbPath`、`DataPath` 和 `CodeOutPaths.godot` 应位于同一个 Godot 项目中；生成器会在脚本中写入它们之间的相对路径。Godot 导出项目时，还需在非资源导出过滤器中包含 `*.bytes`（或配置的 `DataExt`），确保二进制配置被打入 PCK。

使用方式：

```gdscript
var game_data := GameData.new()
if game_data.load_all():
    var item_model = game_data.get_model("Item")
    var item = item_model.get_row(1001)
```

复合主键使用模型生成的 `make_key()`：

```gdscript
attr_model.get_row(AttrModel.make_key(race_id, type))
```

## 工作簿与 Sheet 命名

- 工作簿只是 Sheet 的容器；实际导出的 message、proto 和数据文件名称由 Sheet 名决定。
- 一个工作簿可以包含一个或多个可导出的 Sheet。
- `ExcelDir` 下支持多级目录。
- 以 `~` 开头的 Excel 临时文件会被跳过。
- 以 `#` 开头的 Sheet 用于辅助计算或备注，不会导出。
- 枚举 Sheet 必须以 `_Enum` 结尾，例如 `ItemType_Enum` 导出为 `ItemType` 枚举。
- Sheet 名和字段名应是合法且稳定的 protobuf 标识符。

示例目录：

```text
assets/xls
├── 物品
│   ├── 货币.xlsx
│   ├── 装备.xlsx
│   └── 材料.xlsx
├── 商店.xlsx
└── 升级.xlsx
```

## 表头格式

普通数据 Sheet 的前 4 行为固定表头，第 5 行起为数据：

| 行 | 含义 |
| --- | --- |
| 第 1 行 | 无下划线的 PascalCase 字段名（例如 `ID`、`ItemType`）；该单元格的普通批注可声明字段 tag |
| 第 2 行 | 字段类型及修饰符 |
| 第 3 行 | 导出过滤：`c` 客户端、`s` 服务器、`cs` 两端 |
| 第 4 行 | 字段中文说明；建议水平、垂直居中并启用自动换行 |
| 第 5 行 | 数据开始行 |

只解析第 1 行字段名单元格的普通批注。新版 Excel 自动添加的 `[Threaded comment]` 兼容性说明不是业务 tag，应删除。

普通 Sheet 名和 `_Enum` 之前的枚举名也必须使用无下划线的 PascalCase；这是为了保证 protobuf、Go、C# 和 Godot 生成代码中的类型与字段名保持一致。

### 字段类型（第 2 行）

支持的基础类型：

- `int32`、`int64`、`float`、`double`、`bool`、`string`；
- `timestamp`：按 `TimeZone` 解析并导出为 int64；
- `i18n`：导出为多语言 key；
- 枚举名；
- 其他普通 Sheet 名，表示嵌套 message。

数值、布尔值和时间戳会严格解析，空值或非法文本会导致导出失败。

类型修饰符：

- `pk`：主键，例如 `pk int32`；主键不能为空且必须唯一。
- `unique`：字段值唯一，例如 `unique string`。
- `repeated`：数组，例如 `repeated int32`。

多个 `pk` 字段组成复合主键，唯一性按字段组合检查。主键应导出到所有需要该表的目标端，通常使用 `cs`。

### 枚举 Sheet

枚举 Sheet 使用名称和值两列，并遵循 protobuf 枚举约束：

- 名称不能为空且不能重复；
- 值必须是 int32，且不能重复；
- 第一项的值必须为 `0`。

### 单元格批注 Tag（第 1 行）

多个 tag 使用 `;` 分隔，例如 `fk:Item.ID;index;`：

- `fk:Item.ID`：引用 `Item` Sheet 的 `ID` 字段，导出前校验引用值存在。
- 嵌套字段可使用 `fk:Sheet.Field` 声明引用；`repeated` 外键的每个元素都会分别校验。
- `index`：声明索引标记；当前仅解析并保留该标记，尚未生成二级索引加载代码。

## 数据值编码

- 数组使用 `;` 分隔，例如 `1;2;3`。
- 嵌套结构的单项使用 `|` 按被引用 Sheet 的完整字段顺序编码，多项再使用 `;` 分隔，例如 `100|1;101|2;103|5`。
- 嵌套值先按完整字段顺序解析，然后在客户端或服务器序列化时按各字段的 `c`、`s`、`cs` 过滤。

## 多语言（i18n）

开启 `EnableI18n` 后，所有 `i18n` 字段的源文本会合并到 `I18N.xlsx`，导出的二进制中写入 key 而非原文。

key 格式为 `Sheet_字段名_主键值`；复合主键的各值使用 `_` 连接。包含 i18n 字段的表必须有稳定主键，不要把 i18n 字段自身设为主键。

## 加载代码模板

模板位于 `assets/template/<language>/`，使用 Go `text/template`：

- 文件名不含 `{` 或 `}` 的模板是 Loader，每种语言执行一次。
- 文件名含 `{name}` 或 `{Name}` 的模板是 Module，按普通数据 Sheet 执行。
- 所选语言必须同时存在 Loader 和 Module 模板；Module 生成要求表具有主键。
- 单主键保持 map 查询接口；复合主键在 Go 中使用 `<Sheet>Key` 结构体，在 C# 中使用 `ValueTuple`，在 Godot 中使用 `Array` 键和 `make_key()`。

## 校验与失败行为

导出前会校验主键、`unique`、外键、类型、时间戳、枚举以及嵌套结构字段数量。空主键、重复键、非法标量、未知类型、错误时间戳、无效枚举或缺失模板都会导致导出失败；应修复源配置或工具链，而不是忽略错误。
