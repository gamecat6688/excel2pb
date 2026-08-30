# excel2pb-config 技能

用于设计、编辑、整理和校验 `assets/xls/` 下的 excel2pb 游戏配置工作簿，包括表头、字段类型、主键、外键、索引、枚举、国际化和客户端/服务端导出规则。

## 如何触发

- 显式触发：在请求中写 `$excel2pb-config`。
- 自动触发：请求涉及 `assets/xls/` 下的 `.xlsx` 配置表、excel2pb、protobuf 表结构、`pk`、`unique`、`fk`、`index`、`i18n`、枚举、时间戳或 `c`/`s`/`cs` 导出过滤时，Codex 会自动选择本技能。

## 触发示例

```text
使用 $excel2pb-config 检查 assets/xls 下所有配置表，列出会导致导出失败的问题。
```

```text
给道具配置表增加一个可本地化的描述字段，并保留现有表头样式和批注。
```

```text
检查奖励表里的 fk、复合主键和 repeated 嵌套值是否合法，然后整理列宽和行高。
```

技能的完整执行规则见 [SKILL.md](SKILL.md)。
