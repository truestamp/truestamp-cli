import { Cell } from "./cell.ts";
import { stripColor } from "./deps.ts";
import { Row } from "./row.ts";
import { consumeWords, longest } from "./utils.ts";
export class TableLayout {
    table;
    options;
    constructor(table, options) {
        this.table = table;
        this.options = options;
    }
    toString() {
        const opts = this.createLayout();
        return opts.rows.length ? this.renderRows(opts) : "";
    }
    createLayout() {
        Object.keys(this.options.chars).forEach((key) => {
            if (typeof this.options.chars[key] !== "string") {
                this.options.chars[key] = "";
            }
        });
        const hasBodyBorder = this.table.getBorder() ||
            this.table.hasBodyBorder();
        const hasHeaderBorder = this.table.hasHeaderBorder();
        const hasBorder = hasHeaderBorder || hasBodyBorder;
        const header = this.table.getHeader();
        const rows = this.spanRows(header ? [header, ...this.table] : this.table.slice());
        const columns = Math.max(...rows.map((row) => row.length));
        for (const row of rows) {
            const length = row.length;
            if (length < columns) {
                const diff = columns - length;
                for (let i = 0; i < diff; i++) {
                    row.push(this.createCell(null, row));
                }
            }
        }
        const padding = [];
        const width = [];
        for (let colIndex = 0; colIndex < columns; colIndex++) {
            const minColWidth = Array.isArray(this.options.minColWidth)
                ? this.options.minColWidth[colIndex]
                : this.options.minColWidth;
            const maxColWidth = Array.isArray(this.options.maxColWidth)
                ? this.options.maxColWidth[colIndex]
                : this.options.maxColWidth;
            const colWidth = longest(colIndex, rows, maxColWidth);
            width[colIndex] = Math.min(maxColWidth, Math.max(minColWidth, colWidth));
            padding[colIndex] = Array.isArray(this.options.padding)
                ? this.options.padding[colIndex]
                : this.options.padding;
        }
        return {
            padding,
            width,
            rows,
            columns,
            hasBorder,
            hasBodyBorder,
            hasHeaderBorder,
        };
    }
    spanRows(_rows, rowIndex = 0, colIndex = 0, rowSpan = [], colSpan = 1) {
        const rows = _rows;
        if (rowIndex >= rows.length && rowSpan.every((span) => span === 1)) {
            return rows;
        }
        else if (rows[rowIndex] && colIndex >= rows[rowIndex].length &&
            colIndex >= rowSpan.length && colSpan === 1) {
            return this.spanRows(rows, ++rowIndex, 0, rowSpan, 1);
        }
        if (colSpan > 1) {
            colSpan--;
            rowSpan[colIndex] = rowSpan[colIndex - 1];
            rows[rowIndex].splice(colIndex - 1, 0, rows[rowIndex][colIndex - 1]);
            return this.spanRows(rows, rowIndex, ++colIndex, rowSpan, colSpan);
        }
        if (colIndex === 0) {
            rows[rowIndex] = this.createRow(rows[rowIndex] || []);
        }
        if (rowSpan[colIndex] > 1) {
            rowSpan[colIndex]--;
            rows[rowIndex].splice(colIndex, 0, rows[rowIndex - 1][colIndex]);
            return this.spanRows(rows, rowIndex, ++colIndex, rowSpan, colSpan);
        }
        rows[rowIndex][colIndex] = this.createCell(rows[rowIndex][colIndex] || null, rows[rowIndex]);
        colSpan = rows[rowIndex][colIndex].getColSpan();
        rowSpan[colIndex] = rows[rowIndex][colIndex].getRowSpan();
        return this.spanRows(rows, rowIndex, ++colIndex, rowSpan, colSpan);
    }
    createRow(row) {
        return Row.from(row).border(this.table.getBorder(), false);
    }
    createCell(cell, row) {
        return Cell.from(cell ?? "").border(row.getBorder(), false);
    }
    renderRows(opts) {
        let result = "";
        const rowSpan = new Array(opts.columns).fill(1);
        for (let rowIndex = 0; rowIndex < opts.rows.length; rowIndex++) {
            result += this.renderRow(rowSpan, rowIndex, opts);
        }
        return result.slice(0, -1);
    }
    renderRow(rowSpan, rowIndex, opts, isMultiline) {
        const row = opts.rows[rowIndex];
        const prevRow = opts.rows[rowIndex - 1];
        const nextRow = opts.rows[rowIndex + 1];
        let result = "";
        let colSpan = 1;
        if (!isMultiline && rowIndex === 0 && row.hasBorder()) {
            result += this.renderBorderRow(undefined, row, rowSpan, opts);
        }
        let isMultilineRow = false;
        result += " ".repeat(this.options.indent || 0);
        for (let colIndex = 0; colIndex < opts.columns; colIndex++) {
            if (colSpan > 1) {
                colSpan--;
                rowSpan[colIndex] = rowSpan[colIndex - 1];
                continue;
            }
            result += this.renderCell(colIndex, row, opts);
            if (rowSpan[colIndex] > 1) {
                if (!isMultiline) {
                    rowSpan[colIndex]--;
                }
            }
            else if (!prevRow || prevRow[colIndex] !== row[colIndex]) {
                rowSpan[colIndex] = row[colIndex].getRowSpan();
            }
            colSpan = row[colIndex].getColSpan();
            if (rowSpan[colIndex] === 1 && row[colIndex].length) {
                isMultilineRow = true;
            }
        }
        if (opts.columns > 0) {
            if (row[opts.columns - 1].getBorder()) {
                result += this.options.chars.right;
            }
            else if (opts.hasBorder) {
                result += " ";
            }
        }
        result += "\n";
        if (isMultilineRow) {
            return result + this.renderRow(rowSpan, rowIndex, opts, isMultilineRow);
        }
        if ((rowIndex === 0 && opts.hasHeaderBorder) ||
            (rowIndex < opts.rows.length - 1 && opts.hasBodyBorder)) {
            result += this.renderBorderRow(row, nextRow, rowSpan, opts);
        }
        if (rowIndex === opts.rows.length - 1 && row.hasBorder()) {
            result += this.renderBorderRow(row, undefined, rowSpan, opts);
        }
        return result;
    }
    renderCell(colIndex, row, opts, noBorder) {
        let result = "";
        const prevCell = row[colIndex - 1];
        const cell = row[colIndex];
        if (!noBorder) {
            if (colIndex === 0) {
                if (cell.getBorder()) {
                    result += this.options.chars.left;
                }
                else if (opts.hasBorder) {
                    result += " ";
                }
            }
            else {
                if (cell.getBorder() || prevCell?.getBorder()) {
                    result += this.options.chars.middle;
                }
                else if (opts.hasBorder) {
                    result += " ";
                }
            }
        }
        let maxLength = opts.width[colIndex];
        const colSpan = cell.getColSpan();
        if (colSpan > 1) {
            for (let o = 1; o < colSpan; o++) {
                maxLength += opts.width[colIndex + o] + opts.padding[colIndex + o];
                if (opts.hasBorder) {
                    maxLength += opts.padding[colIndex + o] + 1;
                }
            }
        }
        const { current, next } = this.renderCellValue(cell, maxLength);
        row[colIndex].setValue(next);
        if (opts.hasBorder) {
            result += " ".repeat(opts.padding[colIndex]);
        }
        result += current;
        if (opts.hasBorder || colIndex < opts.columns - 1) {
            result += " ".repeat(opts.padding[colIndex]);
        }
        return result;
    }
    renderCellValue(cell, maxLength) {
        const length = Math.min(maxLength, stripColor(cell.toString()).length);
        let words = consumeWords(length, cell.toString());
        const breakWord = stripColor(words).length > length;
        if (breakWord) {
            words = words.slice(0, length);
        }
        const next = cell.toString().slice(words.length + (breakWord ? 0 : 1));
        const fillLength = maxLength - stripColor(words).length;
        const current = words + " ".repeat(fillLength);
        return {
            current,
            next: cell.clone(next),
        };
    }
    renderBorderRow(prevRow, nextRow, rowSpan, opts) {
        let result = "";
        let colSpan = 1;
        for (let colIndex = 0; colIndex < opts.columns; colIndex++) {
            if (rowSpan[colIndex] > 1) {
                if (!nextRow) {
                    throw new Error("invalid layout");
                }
                if (colSpan > 1) {
                    colSpan--;
                    continue;
                }
            }
            result += this.renderBorderCell(colIndex, prevRow, nextRow, rowSpan, opts);
            colSpan = nextRow?.[colIndex].getColSpan() ?? 1;
        }
        return result.length ? " ".repeat(this.options.indent) + result + "\n" : "";
    }
    renderBorderCell(colIndex, prevRow, nextRow, rowSpan, opts) {
        const a1 = prevRow?.[colIndex - 1];
        const a2 = nextRow?.[colIndex - 1];
        const b1 = prevRow?.[colIndex];
        const b2 = nextRow?.[colIndex];
        const a1Border = !!a1?.getBorder();
        const a2Border = !!a2?.getBorder();
        const b1Border = !!b1?.getBorder();
        const b2Border = !!b2?.getBorder();
        const hasColSpan = (cell) => (cell?.getColSpan() ?? 1) > 1;
        const hasRowSpan = (cell) => (cell?.getRowSpan() ?? 1) > 1;
        let result = "";
        if (colIndex === 0) {
            if (rowSpan[colIndex] > 1) {
                if (b1Border) {
                    result += this.options.chars.left;
                }
                else {
                    result += " ";
                }
            }
            else if (b1Border && b2Border) {
                result += this.options.chars.leftMid;
            }
            else if (b1Border) {
                result += this.options.chars.bottomLeft;
            }
            else if (b2Border) {
                result += this.options.chars.topLeft;
            }
            else {
                result += " ";
            }
        }
        else if (colIndex < opts.columns) {
            if ((a1Border && b2Border) || (b1Border && a2Border)) {
                const a1ColSpan = hasColSpan(a1);
                const a2ColSpan = hasColSpan(a2);
                const b1ColSpan = hasColSpan(b1);
                const b2ColSpan = hasColSpan(b2);
                const a1RowSpan = hasRowSpan(a1);
                const a2RowSpan = hasRowSpan(a2);
                const b1RowSpan = hasRowSpan(b1);
                const b2RowSpan = hasRowSpan(b2);
                const hasAllBorder = a1Border && b2Border && b1Border && a2Border;
                const hasAllRowSpan = a1RowSpan && b1RowSpan && a2RowSpan && b2RowSpan;
                const hasAllColSpan = a1ColSpan && b1ColSpan && a2ColSpan && b2ColSpan;
                if (hasAllRowSpan && hasAllBorder) {
                    result += this.options.chars.middle;
                }
                else if (hasAllColSpan && hasAllBorder && a1 === b1 && a2 === b2) {
                    result += this.options.chars.mid;
                }
                else if (a1ColSpan && b1ColSpan && a1 === b1) {
                    result += this.options.chars.topMid;
                }
                else if (a2ColSpan && b2ColSpan && a2 === b2) {
                    result += this.options.chars.bottomMid;
                }
                else if (a1RowSpan && a2RowSpan && a1 === a2) {
                    result += this.options.chars.leftMid;
                }
                else if (b1RowSpan && b2RowSpan && b1 === b2) {
                    result += this.options.chars.rightMid;
                }
                else {
                    result += this.options.chars.midMid;
                }
            }
            else if (a1Border && b1Border) {
                if (hasColSpan(a1) && hasColSpan(b1) && a1 === b1) {
                    result += this.options.chars.bottom;
                }
                else {
                    result += this.options.chars.bottomMid;
                }
            }
            else if (b1Border && b2Border) {
                if (rowSpan[colIndex] > 1) {
                    result += this.options.chars.left;
                }
                else {
                    result += this.options.chars.leftMid;
                }
            }
            else if (b2Border && a2Border) {
                if (hasColSpan(a2) && hasColSpan(b2) && a2 === b2) {
                    result += this.options.chars.top;
                }
                else {
                    result += this.options.chars.topMid;
                }
            }
            else if (a1Border && a2Border) {
                if (hasRowSpan(a1) && a1 === a2) {
                    result += this.options.chars.right;
                }
                else {
                    result += this.options.chars.rightMid;
                }
            }
            else if (a1Border) {
                result += this.options.chars.bottomRight;
            }
            else if (b1Border) {
                result += this.options.chars.bottomLeft;
            }
            else if (a2Border) {
                result += this.options.chars.topRight;
            }
            else if (b2Border) {
                result += this.options.chars.topLeft;
            }
            else {
                result += " ";
            }
        }
        const length = opts.padding[colIndex] + opts.width[colIndex] +
            opts.padding[colIndex];
        if (rowSpan[colIndex] > 1 && nextRow) {
            result += this.renderCell(colIndex, nextRow, opts, true);
            if (nextRow[colIndex] === nextRow[nextRow.length - 1]) {
                if (b1Border) {
                    result += this.options.chars.right;
                }
                else {
                    result += " ";
                }
                return result;
            }
        }
        else if (b1Border && b2Border) {
            result += this.options.chars.mid.repeat(length);
        }
        else if (b1Border) {
            result += this.options.chars.bottom.repeat(length);
        }
        else if (b2Border) {
            result += this.options.chars.top.repeat(length);
        }
        else {
            result += " ".repeat(length);
        }
        if (colIndex === opts.columns - 1) {
            if (b1Border && b2Border) {
                result += this.options.chars.rightMid;
            }
            else if (b1Border) {
                result += this.options.chars.bottomRight;
            }
            else if (b2Border) {
                result += this.options.chars.topRight;
            }
            else {
                result += " ";
            }
        }
        return result;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibGF5b3V0LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibGF5b3V0LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxJQUFJLEVBQVMsTUFBTSxXQUFXLENBQUM7QUFDeEMsT0FBTyxFQUFFLFVBQVUsRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUN2QyxPQUFPLEVBQVEsR0FBRyxFQUFFLE1BQU0sVUFBVSxDQUFDO0FBRXJDLE9BQU8sRUFBRSxZQUFZLEVBQUUsT0FBTyxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBY25ELE1BQU0sT0FBTyxXQUFXO0lBT1o7SUFDQTtJQUZWLFlBQ1UsS0FBWSxFQUNaLE9BQXVCO1FBRHZCLFVBQUssR0FBTCxLQUFLLENBQU87UUFDWixZQUFPLEdBQVAsT0FBTyxDQUFnQjtJQUM5QixDQUFDO0lBR0csUUFBUTtRQUNiLE1BQU0sSUFBSSxHQUFvQixJQUFJLENBQUMsWUFBWSxFQUFFLENBQUM7UUFDbEQsT0FBTyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLFVBQVUsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDO0lBQ3ZELENBQUM7SUFPUyxZQUFZO1FBQ3BCLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsQ0FBQyxPQUFPLENBQUMsQ0FBQyxHQUFXLEVBQUUsRUFBRTtZQUN0RCxJQUFJLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBMkIsQ0FBQyxLQUFLLFFBQVEsRUFBRTtnQkFDdkUsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBMkIsQ0FBQyxHQUFHLEVBQUUsQ0FBQzthQUN0RDtRQUNILENBQUMsQ0FBQyxDQUFDO1FBRUgsTUFBTSxhQUFhLEdBQVksSUFBSSxDQUFDLEtBQUssQ0FBQyxTQUFTLEVBQUU7WUFDbkQsSUFBSSxDQUFDLEtBQUssQ0FBQyxh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