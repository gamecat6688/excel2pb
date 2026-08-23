# AGENTS.md

本文件为在此仓库中工作的 Codex 提供项目约定。

## 项目概览

这是一个 Go 命令行工具：读取游戏配置 Excel 工作簿，并分别导出客户端（`client`）和服务器（`server`）所需的 `.proto`、protobuf 二进制、经 `protoc` 编译的代码和模板生成的加载代码。

## 构建、测试与验证

```powershell
go build -o excel2pb.exe main.go
.\excel2pb.exe
go test ./parser/
```

- `excel2pb.exe` 读取 `config.yaml` 与 `assets/xls/`，结果写至 `assets/out_*`。
- 不要执行 `go test ./...` 或 `go build ./...`：`assets/template/` 中的 `.go` 文件是 `text/template`，不是可编译的 Go 源文件。
- 运行完整导出前，确认 PATH 中有 `protoc` 以及所需语言的插件（例如 Go 的 `protoc-gen-go`）。`exportPb` 只记录 `protoc` 错误；缺插件可能留下空的 `out_pb/`，因此必须检查实际输出。
- 主键重复、未知类型、错误时间戳等配置问题会触发 `panic`，应修复数据而非吞掉错误。

## 配置与执行流程

- `config.yaml` 缺失时回退到 `config/config.go` 的内嵌默认配置。
- `Outs` 定义 `Client` / `Server` 目标以及路径、`PackageName`、`CodeLanguage`；表头的 `c`、`s`、`cs` 决定导出端。
- 其他关键项：`ExcelDir`、`TimeZone`、`EnableI18n`、`LogLevel`、`MaxProcess`、`TplCodePaths`、`CodeOutPaths`。
- 主流程为 `ParseExcels() -> MergeI18n() -> Export()`；`Export()` 顺序为 `checks -> exportProto -> exportData -> exportCode -> exportPb`。
- 导出的 proto 会被 `ExportData` 通过 `bufbuild/protocompile` 和 `dynamicpb` 再次解析并序列化，因此 proto 输出目录既是中间产物也是交付产物，不能随意删除。
- 并发 worker 的 panic 会被 `works/works.go` 记录而非向上传播；`checks` 主路径上的 panic 是例外。

## Excel 约定

- 工作簿名是导出的 message/proto 名；支持 `ExcelDir` 下一级目录；以 `~` 开头的临时文件跳过。
- `#` 开头的 sheet 跳过；以 `Enum` 结尾的 sheet 导出为枚举，其余导出为 message。
- 前四行固定为：字段名、类型、导出过滤（`c`/`s`/`cs`）、中文说明；数据从第 5 行开始。
- 类型修饰符：`pk`、`unique`、`repeated`。特殊类型：`timestamp`、`i18n`、枚举名和其他 sheet 名所表示的嵌套 message。
- 字段名行的单元格批注是 tag；支持 `fk:Sheet.Field` 和 `index`。仅第一行批注会被解析。
- 数组使用 `;` 分隔；嵌套值单项使用 `|` 分隔字段、多项使用 `;` 分隔。
- 每个数据表应有主键；多列 `pk` 表示复合主键，唯一性按组合检查。主键字段要对所需目标端导出，通常设为 `cs`。

## i18n 与代码生成

- 启用 `EnableI18n` 时，`i18n` 字段的源文本会合并到 `I18N.xlsx`；二进制中写入 `Sheet_Field_主键值` 的 key，而不是原文。
- i18n 表必须有稳定主键；多个主键以 `_` 拼接。不要把 i18n 字段自身设为主键。
- `CodeLanguage` 支持 `golang`、`csharp`、`godot`（Godot 4 / GDScript）。模板位于 `assets/template/<lang>/`，均使用 Go 的 `text/template`。
- 不带 `{`/`}` 的模板是每种语言执行一次的 Loader；带 `{name}` 或 `{Name}` 的模板按 sheet 执行的 Module。Module 需要主键。
- 单主键保持旧版 map 输出；多主键 Go 使用 `<表名>Key` 结构体，C# 使用 `ValueTuple`，Godot 使用 `Array` 键并生成 `make_key()`。

## 配置表修改

修改 `assets/xls` 下的 `.xlsx`、表头、字段约束、i18n、道具/商店/技能等游戏数值，或导出前校验配置时，加载 `.codex/skills/excel2pb-config/SKILL.md` 并遵从其中的设计与检查流程。读写带批注的工作簿使用 `openpyxl`，不要用会丢失批注的 pandas 写回。
