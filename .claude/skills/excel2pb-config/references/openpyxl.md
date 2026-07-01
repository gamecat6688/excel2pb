# 用 openpyxl 读写 excel2pb 配置表

excel2pb 的批注（cell comment）就是字段约束，所以操作 `.xlsx` 必须用能读写批注的库。Python 的 `openpyxl` 可以，`pandas` 默认丢批注、别用。安装：`pip install openpyxl`。

excel2pb 的关键约定（决定了下面代码为什么这么写）：
- 表头 4 行：行 1 字段名、行 2 类型、行 3 导出过滤、行 4 说明；数据从行 5 起。
- **只有第 1 行（字段名行）的批注会被当作 tag**，多个 tag 用 `;` 分隔。
- openpyxl 行列从 1 开始计数，正好对应 Excel 行号。

## 读取表结构 + 批注 tag

```python
import openpyxl

wb = openpyxl.load_workbook("assets/xls/物品.xlsx")
for ws in wb.worksheets:
    name = ws.title
    if name.startswith("#"):          # 辅助表，跳过
        continue
    is_enum = name.endswith("Enum")

    # 表头四行（行1名、行2类型、行3过滤、行4说明）
    names   = [c.value for c in ws[1]]
    types   = [c.value for c in ws[2]]
    filters = [c.value for c in ws[3]]

    # 第 1 行批注 = 字段 tag
    for col in range(1, ws.max_column + 1):
        cell = ws.cell(row=1, column=col)
        if cell.comment:
            raw = cell.comment.text                       # 原始批注文本
            tags = [t.strip() for t in raw.split(";") if t.strip()]
            # 例如 ["fk:Item.ID", "index"]

    # 数据行从第 5 行开始
    for r in range(5, ws.max_row + 1):
        row = [ws.cell(row=r, column=c).value for c in range(1, ws.max_column + 1)]
        if not any(v is not None and str(v) != "" for v in row):
            continue                                       # 跳过空行
```

## 写入批注（声明约束）

```python
from openpyxl.comments import Comment

# 给第 1 行第 3 列的字段加外键 + 索引约束
cell = ws.cell(row=1, column=3)
cell.comment = Comment("fk:Item.ID;index;", "designer")   # 第二个参数是作者名
```

要点：
- 批注写在**第 1 行**对应字段列上，否则 excel2pb 不识别。
- 内容就是 tag 串，如 `fk:Item.ID`、`fk:Reward.ItemID=Item.ID`、`index`、多个用 `;` 连接。
- 覆盖已有批注直接重新赋值 `cell.comment = Comment(...)`；删除用 `cell.comment = None`。

## 新建一张规范表

```python
import openpyxl
from openpyxl.comments import Comment

wb = openpyxl.Workbook()
ws = wb.active
ws.title = "Item"                     # sheet 名 = message 名

# 四行表头：名 / 类型 / 过滤 / 说明
headers = [
    ("ID",     "pk int32",  "cs", "道具ID"),
    ("Name",   "i18n",      "cs", "道具名(多语言)"),
    ("Type",   "ItemType",  "cs", "道具类型(枚举)"),
    ("Rewards","repeated Reward", "s", "掉落奖励"),
]
for col, (n, t, f, d) in enumerate(headers, start=1):
    ws.cell(row=1, column=col, value=n)
    ws.cell(row=2, column=col, value=t)
    ws.cell(row=3, column=col, value=f)
    ws.cell(row=4, column=col, value=d)

# 给 Rewards 加嵌套外键约束
ws.cell(row=1, column=4).comment = Comment("fk:Reward.ItemID=Item.ID", "designer")

# 数据从第 5 行
ws.cell(row=5, column=1, value=1001)
ws.cell(row=5, column=2, value="回城卷轴")     # i18n 填源文本，不填 key
ws.cell(row=5, column=3, value="Consumable")   # 枚举名或整数皆可
ws.cell(row=5, column=4, value="2001|1;2002|3")# 嵌套数组：ItemID|Count，; 分隔

wb.save("assets/xls/物品.xlsx")
```

## 常见坑

- 用 `pandas.read_excel` 读不到批注，也会在写回时丢掉批注和其它 sheet——不要用它改配置表。
- `ws.max_row` / `max_column` 可能因残留空单元格偏大，遍历数据时按「整行为空则跳过」处理。
- 中文 sheet 名 / 文件名没问题，但字段名和 message 名（sheet 名）要用合法标识符。
- 改动已有表时用 `load_workbook` 打开再改，别新建 Workbook 覆盖（会丢其它 sheet 和批注）。
