# 仓库规范

## 项目结构与模块组织

`main.go` 负责加载 `config.yaml`、解析工作簿、合并 i18n 数据并导出产物。解析、校验、清理和代码生成逻辑位于 `parser/`；配置代码位于 `config/`；并发辅助代码位于 `works/`。测试文件以 `*_test.go` 的形式与对应包放在一起，测试夹具位于 `parser/testdata/`。源工作簿位于 `assets/xls/`，语言模板位于 `assets/template/`。禁止提交生成的 `out_proto`、`out_pb`、`out_data` 或 `out_code` 目录。

## 构建、测试与开发命令

- `go run .`：使用 `config.yaml` 运行导出器并写入生成产物。
- `go build -o excel2pb.exe .`：在本地构建 Windows 命令行程序。
- `build_windows.bat`：生成静态链接的 Windows AMD64 二进制文件；`build_all.bat`：生成 Linux、Windows 和 macOS AMD64 二进制文件。
- `go test ./config ./parser ./works`：运行维护范围内的测试，不编译生成代码。
- `go test ./parser -run TestName -v`：调试时运行单个解析器测试。
- `go vet ./config ./parser ./works`：检查维护范围内的包。

## 编码风格与命名约定

使用制表符和 Go 标准格式；修改 `.go` 文件后运行 `gofmt -w`。包名保持简短且全小写。导出标识符使用 `PascalCase`，内部标识符使用 `camelCase`，测试名称使用 `TestBehaviorOrScenario`。新增目标语言生成器时，文件命名为 `parser/codegen_<language>.go`，对应模板放在 `assets/template/<language>/`。优先使用职责单一的小型辅助函数，并对校验场景采用表驱动测试。

## 测试规范

使用 Go 的 `testing` 包。每项行为变更或缺陷修复都需要添加针对性的回归测试。生成临时文件时使用 `t.TempDir()`，静态夹具放在 `parser/testdata/`。提交前运行与改动范围相符的测试和 vet 命令。当 `assets/` 下存在生成的 Go 输出时，避免运行 `go test ./...`，因为它可能导入消费端项目。本项目不设覆盖率硬性门槛。

## 提交与合并请求规范

提交历史目前仅有 `Initial commit`；提交主题采用简洁的 Conventional Commits 风格，例如 `fix: 修复外键校验` 或 `test: 补充枚举解析用例`。每个提交应聚焦单一改动，说明使用简体中文。向项目的 Gitea 仓库发起合并请求，并包含修改动机、验证命令、关联 Issue，以及对结构、生成产物或兼容性的影响。禁止提交二进制文件或密钥。

## 配置安全

将 `config.yaml` 中的路径和导出过滤规则视为面向用户的行为。不要嵌入机器相关的绝对路径。清理逻辑必须谨慎测试：生成文件的清理范围必须限制在配置的输出目录内，绝不能删除源工作簿或模板。
