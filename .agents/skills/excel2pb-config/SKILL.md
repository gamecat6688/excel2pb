---
name: excel2pb-config
description: Design, edit, clean, and validate excel2pb game-configuration workbooks. Use for .xlsx files under assets/xls, headers and styles, protobuf structures, game values, localized text, or pk, unique, fk, index, i18n, timestamps, enums, and client/server export filtering.
---

# excel2pb Configuration Workbooks

excel2pb exports workbooks under `assets/xls/` to protobuf binaries and reader code. Understand the table structure and constraints before changing data so that issues that would cause a panic or corrupt exported data are found before export.

Official excel2pb builds embed the code templates for every supported language. When distributing only the executable, configure `TplCodePaths.<language>` as `embedded://<language>`; for example, use `embedded://godot` for Godot. There is no need to copy the source templates from `assets/template/`.

## Load Guidance by Task

- Before reading, editing, or creating `.xlsx` files in code—especially when reading or writing field comments—read [references/openpyxl.md](references/openpyxl.md). Use a library that preserves comments; never write a workbook back with pandas.
- When text is clipped or a broad schema/content change alters text lengths, run `scripts/format_workbooks.py <workbook-or-directory>` and then verify the resulting widths and wrapped row heights. Run it after the final excel2pb export because i18n merging can regenerate `I18N.xlsx` and reset its widths or row heights.
- When adding a table or field, apply the rules in “Table Structure,” “Types and Primary Keys,” and “Comment Tags.”
- When entering localized, repeated, or nested values, apply the corresponding rules below.
- During a pre-export review, execute every item in “Pre-export Validation Checklist” and report the workbook, sheet, field, and Excel row number. A data row's Excel row number is its zero-based data index plus 5.

## Table Structure

- A workbook contains sheets and may live at any depth under `assets/xls/`. Skip temporary workbooks whose names start with `~`.
- Name gameplay/configuration workbooks in Chinese and omit implementation suffixes such as `Data` (for example, `登录.xlsx`, `玩家资料.xlsx`, and `本地化设置.xlsx`). `I18N.xlsx` is the required exception: excel2pb identifies and regenerates that localization workbook by this exact filename.
- Keep exported sheet names as concise valid technical identifiers even when their workbook filenames are Chinese. Prefer domain or feature names over presentation-layer or implementation suffixes: use `Login`, `World`, `AcquisitionGuide`, or `MaterialSource` rather than `LoginViewData`, `WorldView`, `GuideView`, or `SourceModal` when the records describe feature configuration rather than distinct UI variants. excel2pb derives message, proto, data, and reader names from the sheet name, so a rename must migrate generated files, model lookups, i18n keys, and code references together. Chinese display names belong on workbooks rather than exported sheet tabs.
- A non-enum sheet name determines its exported message, proto, and data filenames. It must be a valid identifier: start with a letter and contain only letters, digits, or underscores. Sheets whose names start with `#` are drafts or notes and are not exported.
- A sheet named `<EnumName>_Enum` defines an enum; every other sheet defines a data message.
- The first four rows of a data sheet have fixed meanings and must not be shifted: row 1 contains field names, row 2 types, row 3 export filters (`c`, `s`, or `cs`), and row 4 descriptions. Data starts on row 5.
- When editing an existing sheet, preserve its colors, borders, and frozen panes. Preserve deliberate column widths when they still display the content; otherwise use content-aware widths that count CJK characters as wide, add padding, and cap very long columns so they wrap instead of making the sheet impractically wide. Keep every cell in rows 1–4 horizontally and vertically centered. Keep row 4 descriptions wrapped so longer explanations remain readable, and increase row heights when wrapped text needs more lines.

## Types and Primary Keys

Base types are `int32`, `int64`, `float`, `double`, `bool`, `string`, `timestamp`, and `i18n`, as well as defined enum names or other sheet names used as nested messages. Type cells may include modifiers:

- `pk int32`: a primary key. Every data sheet that generates reader code should have a primary key.
- `unique string`: values in this column must be unique.
- `repeated Reward`: an array.

Multiple `pk` fields form a composite primary key. Individual components may repeat, but the complete tuple must be unique. Every primary-key component must be non-empty. Row 3 must include every export target actually configured in `Outs`. When only Client is configured, use `c` or `cs`; when both Client and Server are configured, primary-key fields must use `cs`.

Enter `timestamp` values as `YYYY-MM-DD HH:MM:SS`; they are exported as Unix seconds using the configured `TimeZone`. Integers, floating-point numbers, and booleans must parse strictly and must not rely on default zero values. Enum fields may contain an enum name or a 32-bit integer. Enum sheets must not be empty; the first value must be `0`, and names and values must both be unique, for example `InvalidItemType = 0`.

## Comment Tags

Only comments on **field-name cells in row 1** are read. Separate multiple tags with `;`:

- `fk:Item.ID`: every value in this field must exist in field `ID` of the `Item` sheet. Validate each item in a `repeated` value.
- `fk:Reward.ItemID=Item.ID`: validate `ItemID` inside the nested `Reward` message.
- `index`: declare an index. It may be combined with a foreign key, for example `fk:Item.ID;index;`.

Keep only actual tags in comments. Compatibility text generated by Excel or WPS for legacy threaded comments is not a tag. Remove that wrapper text while preserving required `fk:...`, `index`, and similar tags.

## i18n and Complex Values

- Put `i18n` in row 2 for translatable fields, and enter source text rather than a key in the data rows. Export merges the source text into `assets/xls/I18N.xlsx` and writes `Sheet_Field_PrimaryKeyValue` into the binary.
- A sheet with an i18n field must have a stable primary key. Do not use the i18n field itself as a primary key. Join composite-key components with `_`.
- Separate `repeated` values with `;`, for example `1;2;3`.
- Within a nested value, separate fields with `|` in the target sheet's column order, and separate multiple nested items with `;`, for example `1001|2;1002|5`.
- Always enter nested values in the complete target-sheet column order. Export filters target fields according to the nested fields' own `c`, `s`, or `cs` settings; do not change the value order between client and server exports.

## Pre-export Validation Checklist

Report hard errors that stop export before issues that would distort data:

1. Every data sheet has a primary key; all primary-key components are non-empty; primary-key or composite-key tuples do not repeat; and `unique` columns contain no duplicates.
2. Every value parses strictly: integers do not overflow; floats, booleans, and timestamps have valid formats; enum names or integers are valid; and referenced enum and nested types exist.
3. Every sheet containing i18n has a primary key, and the target export does not filter that key out.
4. Every `fk` points to an existing sheet and field, and every value—including repeated and nested items—exists in the target column.
5. The number of fields produced by splitting a nested value on `|` matches the target sheet.
6. Every data sheet has the correct four header rows; every cell in rows 1–4 is horizontally and vertically centered; row 4 descriptions are wrapped; columns are wide enough for ordinary values and headers; capped long text is wrapped with sufficient row height; field and sheet names are valid; and data starts on row 5.
7. Every enum sheet follows `<Name>_Enum`, is non-empty, starts with value `0`, and has no duplicate names or values.

For each issue, identify the workbook, sheet, field, Excel row number, violated rule, and an actionable fix. After editing, run checks appropriate to the change. Before a full export, confirm that the Loader and Module templates for the target language exist, and verify `protoc`, required plugins, and the actual non-empty outputs. Treat any worker, template, or `protoc` failure as an export failure. A successful export removes obsolete managed files from generated directories; do not place handwritten files with managed extensions in `ProtoPath`, `PbPath`, `DataPath`, or `CodeOutPaths`.
