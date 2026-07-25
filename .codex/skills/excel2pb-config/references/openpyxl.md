# 用 openpyxl 读写 excel2pb 配置表

excel2pb 使用单元格批注声明字段约束；读写 `.xlsx` 必须选择保留批注的库。使用 `openpyxl`，不要用 pandas 写回工作簿。

- 表头固定四行：第 1 行字段名、第 2 行类型、第 3 行导出过滤、第 4 行说明；数据从第 5 行开始。
- 只有第 1 行字段名单元格的批注会被视为 tag；多个 tag 用 `;` 分隔。
- openpyxl 的行、列编号从 1 开始，与 Excel 显示的行号一致。

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
