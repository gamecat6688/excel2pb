# Repository Guidelines

## Project Structure & Module Organization

`main.go` loads `config.yaml`, parses workbooks, merges i18n data, and exports artifacts. Parsing, validation, cleanup, and code generation live in `parser/`; configuration is in `config/`, and concurrency helpers in `works/`. Tests sit beside packages as `*_test.go`, with fixtures in `parser/testdata/`. Source workbooks are under `assets/xls/`; language templates are in `assets/template/`. Never commit generated `out_proto`, `out_pb`, `out_data`, or `out_code` directories.

## Build, Test, and Development Commands

- `go run .` runs the exporter with `config.yaml` and writes generated assets.
- `go build -o excel2pb.exe .` builds the Windows CLI locally.
- `build_windows.bat` creates a static Windows AMD64 binary; `build_all.bat` builds Linux, Windows, and macOS AMD64 binaries.
- `go test ./config ./parser ./works` runs maintained tests without compiling generated code.
- `go test ./parser -run TestName -v` runs one parser test while debugging.
- `go vet ./config ./parser ./works` checks maintained packages.

## Coding Style & Naming Conventions

Use tabs and standard Go formatting; run `gofmt -w` on changed `.go` files. Keep package names short and lowercase. Exported identifiers use `PascalCase`, internal identifiers use `camelCase`, and tests use `TestBehaviorOrScenario`. Add target-specific generators as `parser/codegen_<language>.go` and corresponding templates under `assets/template/<language>/`. Prefer small helpers and table-driven tests for validation cases.

## Testing Guidelines

Use Go's `testing` package. Each behavior change or bug fix needs a focused regression test. Use `t.TempDir()` for generated files and `parser/testdata/` for static fixtures. Run the scoped test and vet commands before submitting. Avoid `go test ./...` when generated Go output exists under `assets/`; it may import a consuming project. No coverage threshold is enforced.

## Commit & Pull Request Guidelines

History is limited to `Initial commit`; use concise Conventional Commit-style subjects such as `fix: 修复外键校验` or `test: 补充枚举解析用例`. Keep commits focused and write explanations in Simplified Chinese. Open pull requests against the project's Gitea repository. Include motivation, validation commands, linked issues, and any schema, generated-output, or compatibility impact. Do not commit binaries or secrets.

## Configuration Safety

Treat paths and export filters in `config.yaml` as user-facing behavior. Avoid embedding machine-specific absolute paths. Test cleanup changes carefully: generated-file cleanup must remain scoped to configured output directories and must never remove source workbooks or templates.
