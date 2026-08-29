# Editing excel2pb Workbooks with openpyxl

excel2pb uses cell comments to declare field constraints. When reading or writing `.xlsx` files, use a library that preserves comments. Use `openpyxl`; never write the workbook back with pandas.

- The header has four fixed rows: row 1 contains field names, row 2 types, row 3 export filters, and row 4 descriptions. Data starts on row 5.
- Only comments on field-name cells in row 1 are treated as tags. Separate multiple tags with `;`.
- openpyxl row and column indexes start at 1 and therefore match the row and column numbers displayed by Excel.
- Preserve existing styles when editing a workbook. Keep row 4 descriptions horizontally and vertically centered, and enable wrapping where needed.

## Read Sheets and Comments

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

## Write Tags

```python
from openpyxl.comments import Comment

cell = ws.cell(row=1, column=3)
cell.comment = Comment("fk:Item.ID;index;", "designer")
wb.save("assets/xls/物品.xlsx")
```

Replace a comment by assigning a new value to `cell.comment`; delete it by assigning `None`. When editing an existing workbook, call `load_workbook()` first. Do not create a new `Workbook` and overwrite the file, because doing so loses other sheets and comments.

Text such as `[Threaded comment]`, Excel compatibility notices, or Microsoft links is not an excel2pb tag. If the comment contains no required `fk:` or `index` tag, remove the entire comment. If compatibility text wraps an actual value such as `Comment: fk:...`, normalize it to the plain tag text.

When centering row 4, copy the existing alignment and change only the required properties:

```python
from copy import copy

for cell in ws[4]:
    alignment = copy(cell.alignment)
    alignment.horizontal = "center"
    alignment.vertical = "center"
    alignment.wrap_text = True
    cell.alignment = alignment
```
