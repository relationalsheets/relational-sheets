# Installing

Binaries can be downloaded here:
- [Linux / amd64](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-linux-amd64)
- [Linux / arm64](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-linux-arm64)
- [Linux / i386](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-linux-386)
- [OS X / amd64](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-darwin-amd64)
- [OS X / arm64](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-darwin-arm64)
- [Windows / amd64](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-windows-amd64.exe)
- [Windows / i386](https://relational-sheets.s3.us-west-2.amazonaws.com/relational-sheets-20231111-windows-386.exe)

After downloading, move the file to your "Applications" folder (on Windows)
or run `chmod 0777` (on Linux) to make it executable.

Alternatively, compile from source by cloning this repository and
running `go build` with `go 1.21` or later.

# Running

All you need to provide is your database credentials, via enviornment variable:
```
export DATABASE_URL=postgresql://username:password@host/database
```
Then run the installed binary.

# Use

<h2>Creating Sheets</h2>
<p>
    Click <code>Open > + New</code> to create a new sheet.
    This will open a modal where you can select the <i>primary table</i> the sheet should use
    and join additional tables. Only tables with foreign keys between them can be joined.
    Table names include the database schema&mdash;in most cases this will be "public".
</p>
<h2>Adding &amp; Editing Data</h2>
<p>
    Click <code>Insert > Row</code> to insert a new row in the primary table.
    A blank row will appear at the top of the table with an "Add" button below it.
    You must enter values for all non-null, non-defaulted columns in the primary table,
    but entering values in joined tables is optional. If values are entered in a column
    on a joined table, "Add" will attempt to insert a row in the joined table as well
    and link it to the primary table via the foreign keys used to join the tables.
</p>
<p>
    Primary key columns are identified by a blue header. Clicking on a primary key cell
    will allow you to add rows to the other tables in the sheet which will be linked to
    the selected table, e.g. adding records to a many-to-one relationship. Values can be
    provided for any tables except the selected table, but the set of tables with values
    provided must have a path via foreign keys to the selected table.
</p>
<p>
    Click on an individual cell to edit it. Editing is only allowed on tables with primary
    keys. The database will be updated when on <code>Enter</code> or when you click outside
    of the cell.
</p>
<h2>Sorting, Hiding &amp; Filtering</h2>
<p>
    Clicking on a database column header will cycle it between being unsorted, sorted
    descending, and sorted ascending. It is possible to sort on multiple columns at once,
    in which case the leftmost column will be the primary sort key and columns to the right
    will only be used to break ties.
</p>
<p>
    <code>Ctrl+Click</code> on any column header to hide the column. If you accidentally hide
    columns you don't want to, you can unhide all columns by clicking <code>Edit > Show All Columns</code>.
    Hiding a column removes any sorting and filtering on that column.
</p>
<p>
    Hovering over the filter icon at the right of a database column header will allow you to
    define a filter on a column. This should be a partial SQL expression, where the column will
    be used as the left operand. For example, ">1" will filter the table to only include rows
    where that column has a value greater than 1. Filters can be removed by clearing out the
    filter input or clicking <code>Edit > Clear All Filters</code>.
</p>
<h2>Using the Spreadsheet</h2>
<p>
    To the right of the database tables you can add <i>spreadsheet columns</i> where you
    can enter whatever data you want. Click <code>Insert > Column</code> to add a column.
    Values entered in the cells will be saved. Spreadsheet rows are not tied to database
    rows directly. If you wish to maintain the alignment of spreadsheet rows and
    database rows as more rows appear in your database, you should put an ascending order
    on a sequential ID or timestamp column so that new rows appear at the bottom.
</p>
<p>
    Like Excel, you can perform computations
    and reference other columns by entering a formula that begins with <code>=</code>,
    e.g. <code>=A1+B2</code> will add the values in first cell in column "A" and the second
    cell in column "B". Ranges are also supported when used with aggregation functions,
    e.g. <code>=SUM(A1:A3)</code> will return the sum of the first 3 cells in column "A"
    and <code>=SUM(A:A)</code> will return the sum of the entire column.
    Use <code>Ctrl+Click</code> on a cell with a formula in it to fill every cell below it
    with the same formula, with rows in cell references intelligently adjusted.
</p>
<p>
    You can also reference cells from your database tables using the same syntax.
    E.g. if there is column named "foo" in one of your tables, the first cell shown in that table can be referenced
    as "foo1". If there are multiple columns with the same name, you can distinguish them by using a fully-qualified
    name, e.g. "public.mytable.mycolumn". Aggregation functions invoked with the whole column as the range will
    run against every row in the table, even if the sheet is not able to display all rows due to row limits.
</p>
<p>
    Spreadsheet columns support the following functions:
    <ul>
        <li><code>IF(condition, value_when_true, value_when_false)</code></li>
        <li><code>MAX(values...)</code></li>
        <li><code>MIN(values...)</code></li>
        <li><code>SUM(values...)</code></li>
        <li><code>PRODUCT(values...)</code></li>
        <li><code>AVERAGE(values...)</code></li>
        <li><code>COUNTIF(condition_range, condition)</code></li>
        <li><code>SUMIF(condition_range, condition[, sum_range])</code></li>
        <li><code>AVERAGEIF(condition_range, condition[, sum_range])</code></li>
        <li><code>REGEXMATCH(search_string, pattern)</code></li>
    </ul>
</p>
<h2>Mouse &amp; Keybindings</h2>
<p>
    For database columns:
    <ul>
        <li><code>Click</code> the header to sort by a column or change sorting direction</li>
        <li><code>Shift+Click</code> the header to hide a column</li>
        <li><code>Click</code> a cell in a <i>primary key</i> column (identified by a blue header) to add a child row</li>
        <li><code>Click</code> any other cell to edit the value in that cell (only possible for tables with primary keys)</li>
    </ul>
    For spreadsheet columns:
    <ul>
        <li><code>Click</code> the header to rename the column</li>
        <li><code>Shift+Click</code> the header to delete the column</li>
        <li><code>Click</code> a cell to see and edit its formula</li>
        <li><code>Ctrl+Click</code> a cell to intelligently fill the cells below it with the formula</li>
    </ul>
</p>

# Contributing

This project is not accepting outside PRs at this time.
However, feature requests are welcome and bug reports are always appreciated!
