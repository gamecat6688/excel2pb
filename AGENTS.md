# Repository Guidelines

## Project Structure and Module Organization

`main.go` loads `config.yaml`, parses workbooks, merges i18n data, and exports artifacts. Parsing, validation, cleanup, and code generation live in `parser/`; configuration code lives in `config/`; concurrency helpers live in `works/`. Tests sit beside their packages as `*_test.go`, with fixtures in `parser/testdata/`. Source workbooks are under `assets/xls/`, and language templates are under `assets/template/`. Never commit generated `out_proto`, `out_pb`, `out_data`, or `out_code` directories.

## Build, Test, and Development Commands

- `go run .`: Run the exporter with `config.yaml` and write generated artifacts.
- `go build -o excel2pb.exe .`: Build the Windows CLI locally.
- `build_windows.bat`: Create a static Windows AMD64 binary; `build_all.bat`: build Linux, Windows, and macOS AMD64 binaries.
- `go test ./config ./parser ./works`: Run the maintained tests without compiling generated code.
- `go test ./parser -run TestName -v`: Run one parser test while debugging.
- `go vet ./config ./parser ./works`: Check the maintained packages.

## Coding Style and Naming Conventions

Use tabs and standard Go formatting; run `gofmt -w` on changed `.go` files. Keep package names short and lowercase. Exported identifiers use `PascalCase`, internal identifiers use `camelCase`, and tests use `TestBehaviorOrScenario`. Add target-specific generators as `parser/codegen_<language>.go` and place the corresponding templates under `assets/template/<language>/`. Prefer small, focused helpers and table-driven tests for validation scenarios.

## Testing Guidelines

Use Go's `testing` package. Each behavior change or bug fix needs a focused regression test. Use `t.TempDir()` for generated files and `parser/testdata/` for static fixtures. Run tests and vet commands appropriate to the change before submitting. Avoid `go test ./...` when generated Go output exists under `assets/`, because it may import a consuming project. No coverage threshold is enforced.

## Commit and Pull Request Guidelines

The history currently contains only `Initial commit`; use concise Conventional Commit-style subjects such as `fix: 修复外键校验` or `test: 补充枚举解析用例`. Keep commits focused and write explanations in Simplified Chinese. Open pull requests against the project's Gitea repository. Include the motivation, validation commands, linked issues, and any schema, generated-output, or compatibility impact. Do not commit binaries or secrets.

## Configuration Safety

Treat paths and export filters in `config.yaml` as user-facing behavior. Avoid embedding machine-specific absolute paths. Test cleanup changes carefully: generated-file cleanup must remain scoped to configured output directories and must never remove source workbooks or templates.
