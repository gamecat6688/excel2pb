# 用 openpyxl 读写 excel2pb 配置表

excel2pb 使用单元格批注声明字段约束；读写 `.xlsx` 必须选择保留批注的库。使用 `openpyxl`，不要用 pandas 写回工作簿。

- 表头固定四行：第 1 行字段名、第 2 行类型、第 3 行导出过滤、第 4 行说明；数据从第 5 行开始。
- 只有第 1 行字段名单元格的批注会被视为 tag；多个 tag 用 `;` 分隔。
- openpyxl 的行、列编号从 1 开始，与 Excel 显示的行号一致。
- 修改现有工作簿时保留颜色、边框、批注和冻结窗格；第 1–4 行所有单元格水平、垂直居中，第 4 行说明文字开启自动换行。
- 修改字段名或数值后，按可见内容调整列宽。中日韩全角字符按两个显示单位计算，增加水平留白，长文本列最大宽度建议限制为 48；超过上限的文本自动换行并增加行高，不要把整张表拉得过宽。
- 需要确定性格式化时，优先运行 `../scripts/format_workbooks.py <工作簿或目录>`。它会保留现有工作簿，统一第 1–4 行对齐方式，按 Unicode 显示宽度调整列宽，并处理长文本换行和行高。应在最后一次 excel2pb 导出后运行，因为 i18n 合并可能重新生成 `I18N.xlsx`，覆盖此前的列宽或行高。

## 读取表与批注

```python
import openpyxl

wb = openpyxl.load_workbook("assets/xls/物品.xlsx")
for ws in wb.worksheets:
    if ws.title.startswith("#"):
        continue

    names = [cell.value for cell in ws[1]]
    types = [cell.value for cell in ws[2]]
    filters = [cell.value for cell in ws[3]]

    for col in range(1, ws.max_column + 1):
        cell = ws.cell(row=1, column=col)
        tags = [] if not cell.comment else [
            tag.strip() for tag in cell.comment.text.split(";") if tag.strip()
        ]

    for row in range(5, ws.max_row + 1):
        values = [ws.cell(row=row, column=col).value for col in range(1, ws.max_column + 1)]
        if not any(value is not None and str(value) != "" for value in values):
            continue
```

## 写入 tag

```python
from openpyxl.comments import Comment

cell = ws.cell(row=1, column=3)
cell.comment = Comment("fk:Item.ID;index;", "designer")
wb.save("assets/xls/物品.xlsx")
```

覆盖批注时重新赋值 `cell.comment`；删除批注时赋值 `None`。修改现有表时先 `load_workbook()` 再改，不能新建 Workbook 覆盖原文件，否则会丢掉其他 sheet 与批注。

如果批注正文包含 `[Threaded comment]`、Excel 版本兼容说明或微软链接，这些文字不是 excel2pb tag。确认其中没有必要的 `fk:` 或 `index` 后删除整个批注；若兼容文本后仍带有实际 `Comment: fk:...`，则把批注规范化为纯 tag 文本。

设置四行表头格式时复用现有对齐属性，只覆盖必要字段；第 4 行保持自动换行：

```python
from copy import copy

for row in range(1, 5):
    for cell in ws[row]:
        alignment = copy(cell.alignment)
        alignment.horizontal = "center"
        alignment.vertical = "center"
        if row == 4:
            alignment.wrap_text = True
        cell.alignment = alignment
```
