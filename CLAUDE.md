# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 这是什么

一个 Go 命令行工具，读取游戏配置的 Excel 工作簿，按输出目标（客户端 client / 服务器 server）分别导出：
1. `.proto` 结构文件
2. protobuf 二进制数据文件（序列化后的 Excel 行数据）
3. 编译后的 protobuf 代码（调用外部 `protoc`）
4. 加载/读取代码（基于用户模板生成）

## 构建与运行

```bash
go build -o excel2pb.exe main.go     # 或运行 build_windows.bat / build_all.bat（交叉编译）
./excel2pb.exe                        # 读取 config.yaml + assets/xls/，输出到 assets/out_*
```

测试：`go test ./parser/`（注意不要用 `go test ./...` 或 `go build ./...`——`assets/template/` 下的 `.go` 是 text/template 模板，不是合法 Go，会编译失败）。目前仅 `parser/multikey_render_test.go` 一个渲染测试。

**外部运行时依赖**（必须在 PATH 中）：`protoc`，以及每种配置语言对应的 codegen 插件（例如 golang 需要 `protoc-gen-go`）。`exportPb` 阶段会 shell 调用 `protoc`，且只记录错误——插件缺失时会生成空的 `out_pb/` 而不会硬报错。

程序遇到数据错误会 `panic`（主键重复、未知类型、时间戳格式错误等）；这些是刻意抛给配置作者的硬中断。

## 配置

`config.yaml`（若不存在则回退到 `config/config.go` 里内嵌的默认配置）驱动整个流程：
- `Outs` 映射 目标名 → 路径 + `PackageName` + `CodeLanguage`。两个目标键名为 `Client` 和 `Server`，对应 `c`/`s` 导出过滤。
- `ExcelDir`、`TimeZone`（用于时间戳转换）、`EnableI18n`、`LogLevel`、`MaxProcess`（GOMAXPROCS）。
- `TplCodePaths` / `CodeOutPaths` 映射 语言 → 模板目录 / 输出目录。

## 流程（main.go → parser.Parser）

`ParseExcels() → MergeI18n() → Export()`，其中 `Export()` 按顺序执行 `checks → exportProto → exportData → exportCode → exportPb`。大部分阶段通过 `works.Go` / `works.Wait` 在 sheet×filter 维度上并发展开（`works/works.go` 是对全局 WaitGroup 的封装，并对每个 goroutine 做 panic recover——注意 worker 中的 panic 是被记录而非向上传播的，除了 checks 在主路径上刻意 panic 的情况）。

单个数据表的数据流：原始 `[][]string` 行 → `SheetParser`（表头 + dataRows）→ `SplitByFilter(c|s)` 生成过滤后的 `SheetParser` → `ExportProto` 用 text/template 写出 `.proto` → `ExportData` 在运行时用 `bufbuild/protocompile` + `dynamicpb` 反射**重新解析**该 `.proto` 来构造并序列化消息。所以**proto 文件是先生成、再被读回来**用于序列化二进制的——proto 输出目录是中间产物，不只是最终产物。

## Excel 约定（编码在 parser/ 中，另见 README.md）

- **工作簿文件名** = 导出的 message/proto 名。支持 `ExcelDir` 下一层嵌套文件夹。以 `~` 开头的（临时文件）会被跳过。
- **Sheet**：`#` 开头的 sheet 被跳过（草稿/备注）。以 `Enum` 结尾的 → `EnumParser`（proto 枚举）。其余 → `SheetParser`（proto message）。
- **表头行**是前 `HeadCount`（=4）行，定义在 `parser/consts.go`：第 0 行字段名，第 1 行类型，第 2 行导出过滤（`c`/`s`/`cs`），第 3 行中文说明。数据从第 4 行开始。修改 `HeadCount` 会整体位移。
- **类型修饰符**写在类型单元格里：`pk`（主键，唯一）、`unique`、`repeated`。特殊类型：`timestamp`（→int64 unix，使用 `TimeZone`）、`i18n`（→string key，抽取到 I18N 工作簿）、枚举名、以及 struct/message 类型（字段的基础类型若匹配另一个 sheet 名则成为嵌套 message）。
- **单元格批注 tag**打在字段名那一行，作用类似 ORM tag，在 `ParseHeadTags` / `parser/head_tag.go` 中解析。目前支持 `fk:Sheet.Field`（外键，在 `checks` 中校验）和 `index`。
- **值编码**（`parser/utils.go`）：`;` 分隔数组元素（`SplitBaseValue`）；嵌套结构中 `|` 分隔单个元素内的字段、`;` 分隔元素（`SplitCustomValue`）。

## i18n

当 `EnableI18n` 时，`MergeI18n` 把每个 `i18n` 类型字段的值收集进单个 `I18N.xlsx` 工作簿（key = `Sheet_Field_主键值`，见 `MakeI18nKey`），与已有文件合并，数据导出时写入二进制的是 key（而非文本）。见 `parser/i18n.go`。

## 新增 codegen 语言

`ExportCode` / `exportLoadCode` 根据 `CodeLanguage` 分支（支持 `golang`、`csharp`、`godot`；见 `parser/codegen_golang.go`、`codegen_csharp.go`、`codegen_godot.go`）。两类生成器：
- **Loader**（`GenCode(root)`）：每种语言运行一次，遍历文件名**不含** `{`/`}` 的模板文件，拿到排序后的 sheet 名列表。
- **Module**（`GenCode(root, sheet)`）：每个 sheet 运行一次，遍历文件名**含** `{name}`/`{Name}` 的模板文件（小写/原样），替换进输出文件名。模板目录为 `assets/template/<lang>/`。

即使目标不是 Go，模板也是 Go 的 `text/template`（`.cs`、`.h` 等）。module 生成器要求每个数据表都有主键。

## 多主键

一个表可声明多个 `pk` 列，语义为复合主键：`checkPrimaryKey` 按主键**组合**去重（单列可重复，组合唯一）。codegen 通过 `CodeModuleModel.Keys`（`[]KeyField`）+ `MultiKey` 传给模板：
- **单主键**：输出与旧版逐字节一致（`map[T]` / `Dictionary<T,>`，直接用字段名作 key）。
- **多主键**：golang 生成组合 key 结构体 `<表名>Key{...}` 并以其为 map key；csharp 用 ValueTuple `(t1, t2)` 作 `Dictionary` key；Godot 使用 `Array` 作为 `Dictionary` key 并生成 `make_key()`。
`GetI18nPrimaryKey` 用 `_` 拼接所有主键值作为 i18n key 的一部分。
