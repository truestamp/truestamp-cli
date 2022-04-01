import { Cell } from "./cell.ts";
import { Row } from "./row.ts";
import { consumeWords, longest, strLength } from "./utils.ts";
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
        const rows = this.#getRows();
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
    #getRows() {
        const header = this.table.getHeader();
        const rows = header ? [header, ...this.table] : this.table.slice();
        const hasSpan = rows.some((row) => row.some((cell) => cell instanceof Cell && (cell.getColSpan() > 1 || cell.getRowSpan() > 1)));
        if (hasSpan) {
            return this.spanRows(rows);
        }
        return rows.map((row) => {
            const newRow = this.createRow(row);
            return newRow.map((cell) => this.createCell(cell, newRow));
        });
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
        return Row.from(row)
            .border(this.table.getBorder(), false)
            .align(this.table.getAlign(), false);
    }
    createCell(cell, row) {
        return Cell.from(cell ?? "")
            .border(row.getBorder(), false)
            .align(row.getAlign(), false);
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
        const length = Math.min(maxLength, strLength(cell.toString()));
        let words = consumeWords(length, cell.toString());
        const breakWord = strLength(words) > length;
        if (breakWord) {
            words = words.slice(0, length);
        }
        const next = cell.toString().slice(words.length + (breakWord ? 0 : 1));
        const fillLength = maxLength - strLength(words);
        const align = cell.getAlign();
        let current;
        if (fillLength === 0) {
            current = words;
        }
        else if (align === "left") {
            current = words + " ".repeat(fillLength);
        }
        else if (align === "center") {
            current = " ".repeat(Math.floor(fillLength / 2)) + words +
                " ".repeat(Math.ceil(fillLength / 2));
        }
        else if (align === "right") {
            current = " ".repeat(fillLength) + words;
        }
        else {
            throw new Error("Unknown direction: " + align);
        }
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibGF5b3V0LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibGF5b3V0LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxJQUFJLEVBQW9CLE1BQU0sV0FBVyxDQUFDO0FBQ25ELE9BQU8sRUFBUSxHQUFHLEVBQUUsTUFBTSxVQUFVLENBQUM7QUFFckMsT0FBTyxFQUFFLFlBQVksRUFBRSxPQUFPLEVBQUUsU0FBUyxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBYzlELE1BQU0sT0FBTyxXQUFXO0lBT1o7SUFDQTtJQUZWLFlBQ1UsS0FBWSxFQUNaLE9BQXVCO1FBRHZCLFVBQUssR0FBTCxLQUFLLENBQU87UUFDWixZQUFPLEdBQVAsT0FBTyxDQUFnQjtJQUM5QixDQUFDO0lBR0csUUFBUTtRQUNiLE1BQU0sSUFBSSxHQUFvQixJQUFJLENBQUMsWUFBWSxFQUFFLENBQUM7UUFDbEQsT0FBTyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLFVBQVUsQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUMsRUFBRSxDQUFDO0lBQ3ZELENBQUM7SUFPUyxZQUFZO1FBQ3BCLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsQ0FBQyxPQUFPLENBQUMsQ0FBQyxHQUFXLEVBQUUsRUFBRTtZQUN0RCxJQUFJLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBMkIsQ0FBQyxLQUFLLFFBQVEsRUFBRTtnQkFDdkUsSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBMkIsQ0FBQyxHQUFHLEVBQUUsQ0FBQzthQUN0RDtRQUNILENBQUMsQ0FBQyxDQUFDO1FBRUgsTUFBTSxhQUFhLEdBQVksSUFBSSxDQUFDLEtBQUssQ0FBQyxTQUFTLEVBQUU7WUFDbkQsSUFBSSxDQUFDLEtBQUssQ0FBQyxhQUFhLEVBQUUsQ0FBQztRQUM3QixNQUFNLGVBQWUsR0FBWSxJQUFJLENBQUMsS0FBSyxDQUFDLGVBQWUsRUFBRSxDQUFDO1FBQzlELE1BQU0sU0FBUyxHQUFZLGVBQWUsSUFBSSxhQUFhLENBQUM7UUFFNUQsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLFFBQVEsRUFBRSxDQUFDO1FBRTdCLE1BQU0sT0FBTyxHQUFXLElBQUksQ0FBQyxHQUFHLENBQUMsR0FBRyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FBQyxHQUFHLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUNuRSxLQUFLLE1BQU0sR0FBRyxJQUFJLElBQUksRUFBRTtZQUN0QixNQUFNLE1BQU0sR0FBVyxHQUFHLENBQUMsTUFBTSxDQUFDO1lBQ2xDLElBQUksTUFBTSxHQUFHLE9BQU8sRUFBRTtnQkFDcEIsTUFBTSxJQUFJLEdBQUcsT0FBTyxHQUFHLE1BQU0sQ0FBQztnQkFDOUIsS0FBSyxJQUFJLENBQUMsR0FBRyxDQUFDLEVBQUUsQ0FBQyxHQUFHLElBQUksRUFBRSxDQUFDLEVBQUUsRUFBRTtvQkFDN0IsR0FBRyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLElBQUksRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDO2lCQUN0QzthQUNGO1NBQ0Y7UUFFRCxNQUFNLE9BQU8sR0FBYSxFQUFFLENBQUM7UUFDN0IsTUFBTSxLQUFLLEdBQWEsRUFBRSxDQUFDO1FBQzNCLEtBQUssSUFBSSxRQUFRLEdBQUcsQ0FBQyxFQUFFLFFBQVEsR0FBRyxPQUFPLEVBQUUsUUFBUSxFQUFFLEVBQUU7WUFDckQsTUFBTSxXQUFXLEdBQVcsS0FBSyxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLFdBQVcsQ0FBQztnQkFDakUsQ0FBQyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsV0FBVyxDQUFDLFFBQVEsQ0FBQztnQkFDcEMsQ0FBQyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsV0FBVyxDQUFDO1lBQzdCLE1BQU0sV0FBVyxHQUFXLEtBQUssQ0FBQyxPQUFPLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxXQUFXLENBQUM7Z0JBQ2pFLENBQUMsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLFdBQVcsQ0FBQyxRQUFRLENBQUM7Z0JBQ3BDLENBQUMsQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLFdBQVcsQ0FBQztZQUM3QixNQUFNLFFBQVEsR0FBVyxPQUFPLENBQUMsUUFBUSxFQUFFLElBQUksRUFBRSxXQUFXLENBQUMsQ0FBQztZQUM5RCxLQUFLLENBQUMsUUFBUSxDQUFDLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FBQyxXQUFXLEVBQUUsSUFBSSxDQUFDLEdBQUcsQ0FBQyxXQUFXLEVBQUUsUUFBUSxDQUFDLENBQUMsQ0FBQztZQUN6RSxPQUFPLENBQUMsUUFBUSxDQUFDLEdBQUcsS0FBSyxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLE9BQU8sQ0FBQztnQkFDckQsQ0FBQyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDLFFBQVEsQ0FBQztnQkFDaEMsQ0FBQyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDO1NBQzFCO1FBRUQsT0FBTztZQUNMLE9BQU87WUFDUCxLQUFLO1lBQ0wsSUFBSTtZQUNKLE9BQU87WUFDUCxTQUFTO1lBQ1QsYUFBYTtZQUNiLGVBQWU7U0FDaEIsQ0FBQztJQUNKLENBQUM7SUFFRCxRQUFRO1FBQ04sTUFBTSxNQUFNLEdBQW9CLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxFQUFFLENBQUM7UUFDdkQsTUFBTSxJQUFJLEdBQUcsTUFBTSxDQUFDLENBQUMsQ0FBQyxDQUFDLE1BQU0sRUFBRSxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxLQUFLLEVBQUUsQ0FBQztRQUNuRSxNQUFNLE9BQU8sR0FBRyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FDaEMsR0FBRyxDQUFDLElBQUksQ0FBQyxDQUFDLElBQUksRUFBRSxFQUFFLENBQ2hCLElBQUksWUFBWSxJQUFJLElBQUksQ0FBQyxJQUFJLENBQUMsVUFBVSxFQUFFLEdBQUcsQ0FBQyxJQUFJLElBQUksQ0FBQyxVQUFVLEVBQUUsR0FBRyxDQUFDLENBQUMsQ0FDekUsQ0FDRixDQUFDO1FBRUYsSUFBSSxPQUFPLEVBQUU7WUFDWCxPQUFPLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLENBQUM7U0FDNUI7UUFFRCxPQUFPLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLEVBQUUsRUFBRTtZQUN0QixNQUFNLE1BQU0sR0FBRyxJQUFJLENBQUMsU0FBUyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1lBQ25DLE9BQU8sTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDLElBQUksRUFBRSxFQUFFLENBQUMsSUFBSSxDQUFDLFVBQVUsQ0FBQyxJQUFJLEVBQUUsTUFBTSxDQUFDLENBQUMsQ0FBQztRQUM3RCxDQUFDLENBQXFCLENBQUM7SUFDekIsQ0FBQztJQVlTLFFBQVEsQ0FDaEIsS0FBYSxFQUNiLFFBQVEsR0FBRyxDQUFDLEVBQ1osUUFBUSxHQUFHLENBQUMsRUFDWixVQUFvQixFQUFFLEVBQ3RCLE9BQU8sR0FBRyxDQUFDO1FBRVgsTUFBTSxJQUFJLEdBQWdCLEtBQW9CLENBQUM7UUFFL0MsSUFBSSxRQUFRLElBQUksSUFBSSxDQUFDLE1BQU0sSUFBSSxPQUFPLENBQUMsS0FBSyxDQUFDLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxJQUFJLEtBQUssQ0FBQyxDQUFDLEVBQUU7WUFDbEUsT0FBTyxJQUFJLENBQUM7U0FDYjthQUFNLElBQ0wsSUFBSSxDQUFDLFFBQVEsQ0FBQyxJQUFJLFFBQVEsSUFBSSxJQUFJLENBQUMsUUFBUSxDQUFDLENBQUMsTUFBTTtZQUNuRCxRQUFRLElBQUksT0FBTyxDQUFDLE1BQU0sSUFBSSxPQUFPLEtBQUssQ0FBQyxFQUMzQztZQUNBLE9BQU8sSUFBSSxDQUFDLFFBQVEsQ0FBQyxJQUFJLEVBQUUsRUFBRSxRQUFRLEVBQUUsQ0FBQyxFQUFFLE9BQU8sRUFBRSxDQUFDLENBQUMsQ0FBQztTQUN2RDtRQUVELElBQUksT0FBTyxHQUFHLENBQUMsRUFBRTtZQUNmLE9BQU8sRUFBRSxDQUFDO1lBQ1YsT0FBTyxDQUFDLFFBQVEsQ0FBQyxHQUFHLE9BQU8sQ0FBQyxRQUFRLEdBQUcsQ0FBQyxDQUFDLENBQUM7WUFDMUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxDQUFDLE1BQU0sQ0FBQyxRQUFRLEdBQUcsQ0FBQyxFQUFFLENBQUMsRUFBRSxJQUFJLENBQUMsUUFBUSxDQUFDLENBQUMsUUFBUSxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUM7WUFDckUsT0FBTyxJQUFJLENBQUMsUUFBUSxDQUFDLElBQUksRUFBRSxRQUFRLEVBQUUsRUFBRSxRQUFRLEVBQUUsT0FBTyxFQUFFLE9BQU8sQ0FBQyxDQUFDO1NBQ3BFO1FBRUQsSUFBSSxRQUFRLEtBQUssQ0FBQyxFQUFFO1lBQ2xCLElBQUksQ0FBQyxRQUFRLENBQUMsR0FBRyxJQUFJLENBQUMsU0FBUyxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxFQUFFLENBQUMsQ0FBQztTQUN2RDtRQUVELElBQUksT0FBTyxDQUFDLFFBQVEsQ0FBQyxHQUFHLENBQUMsRUFBRTtZQUN6QixPQUFPLENBQUMsUUFBUSxDQUFDLEVBQUUsQ0FBQztZQUNwQixJQUFJLENBQUMsUUFBUSxDQUFDLENBQUMsTUFBTSxDQUFDLFFBQVEsRUFBRSxDQUFDLEVBQUUsSUFBSSxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsQ0FBQyxRQUFRLENBQUMsQ0FBQyxDQUFDO1lBQ2pFLE9BQU8sSUFBSSxDQUFDLFFBQVEsQ0FBQyxJQUFJLEVBQUUsUUFBUSxFQUFFLEVBQUUsUUFBUSxFQUFFLE9BQU8sRUFBRSxPQUFPLENBQUMsQ0FBQztTQUNwRTtRQUVELElBQUksQ0FBQyxRQUFRLENBQUMsQ0FBQyxRQUFRLENBQUMsR0FBRyxJQUFJLENBQUMsVUFBVSxDQUN4QyxJQUFJLENBQUMsUUFBUSxDQUFDLENBQUMsUUFBUSxDQUFDLElBQUksSUFBSSxFQUNoQyxJQUFJLENBQUMsUUFBUSxDQUFDLENBQ2YsQ0FBQztRQUVGLE9BQU8sR0FBRyxJQUFJLENBQUMsUUFBUSxDQUFDLENBQUMsUUFBUSxDQUFDLENBQUMsVUFBVSxFQUFFLENBQUM7UUFDaEQsT0FBTyxDQUFDLFFBQVEsQ0FBQyxHQUFHLElBQUksQ0FBQyxRQUFRLENBQUMsQ0FBQyxRQUFRLENBQUMsQ0FBQyxVQUFVLEVBQUUsQ0FBQztRQUUxRCxPQUFPLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxFQUFFLFFBQVEsRUFBRSxFQUFFLFFBQVEsRUFBRSxPQUFPLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFDckUsQ0FBQztJQU1TLFNBQVMsQ0FBQyxHQUFTO1FBQzNCLE9BQU8sR0FBRyxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUM7YUFDakIsTUFBTSxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxFQUFFLEVBQUUsS0FBSyxDQUFDO2FBQ3JDLEtBQUssQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLFFBQVEsRUFBRSxFQUFFLEtBQUssQ0FBYyxDQUFDO0lBQ3RELENBQUM7SUFPUyxVQUFVLENBQUMsSUFBa0IsRUFBRSxHQUFRO1FBQy9DLE9BQU8sSUFBSSxDQUFDLElBQUksQ0FBQyxJQUFJLElBQUksRUFBRSxDQUFDO2FBQ3pCLE1BQU0sQ0FBQyxHQUFHLENBQUMsU0FBUyxFQUFFLEVBQUUsS0FBSyxDQUFDO2FBQzlCLEtBQUssQ0FBQyxHQUFHLENBQUMsUUFBUSxFQUFFLEVBQUUsS0FBSyxDQUFDLENBQUM7SUFDbEMsQ0FBQztJQU1TLFVBQVUsQ0FBQyxJQUFxQjtRQUN4QyxJQUFJLE1BQU0sR0FBRyxFQUFFLENBQUM7UUFDaEIsTUFBTSxPQUFPLEdBQWEsSUFBSSxLQUFLLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQztRQUUxRCxLQUFLLElBQUksUUFBUSxHQUFHLENBQUMsRUFBRSxRQUFRLEdBQUcsSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLEVBQUUsUUFBUSxFQUFFLEVBQUU7WUFDOUQsTUFBTSxJQUFJLElBQUksQ0FBQyxTQUFTLENBQUMsT0FBTyxFQUFFLFFBQVEsRUFBRSxJQUFJLENBQUMsQ0FBQztTQUNuRDtRQUVELE9BQU8sTUFBTSxDQUFDLEtBQUssQ0FBQyxDQUFDLEVBQUUsQ0FBQyxDQUFDLENBQUMsQ0FBQztJQUM3QixDQUFDO0lBU1MsU0FBUyxDQUNqQixPQUFpQixFQUNqQixRQUFnQixFQUNoQixJQUFxQixFQUNyQixXQUFxQjtRQUVyQixNQUFNLEdBQUcsR0FBYyxJQUFJLENBQUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxDQUFDO1FBQzNDLE1BQU0sT0FBTyxHQUEwQixJQUFJLENBQUMsSUFBSSxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsQ0FBQztRQUMvRCxNQUFNLE9BQU8sR0FBMEIsSUFBSSxDQUFDLElBQUksQ0FBQyxRQUFRLEdBQUcsQ0FBQyxDQUFDLENBQUM7UUFDL0QsSUFBSSxNQUFNLEdBQUcsRUFBRSxDQUFDO1FBRWhCLElBQUksT0FBTyxHQUFHLENBQUMsQ0FBQztRQUdoQixJQUFJLENBQUMsV0FBVyxJQUFJLFFBQVEsS0FBSyxDQUFDLElBQUksR0FBRyxDQUFDLFNBQVMsRUFBRSxFQUFFO1lBQ3JELE1BQU0sSUFBSSxJQUFJLENBQUMsZUFBZSxDQUFDLFNBQVMsRUFBRSxHQUFHLEVBQUUsT0FBTyxFQUFFLElBQUksQ0FBQyxDQUFDO1NBQy9EO1FBRUQsSUFBSSxjQUFjLEdBQUcsS0FBSyxDQUFDO1FBRTNCLE1BQU0sSUFBSSxHQUFHLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsTUFBTSxJQUFJLENBQUMsQ0FBQyxDQUFDO1FBRS9DLEtBQUssSUFBSSxRQUFRLEdBQUcsQ0FBQyxFQUFFLFFBQVEsR0FBRyxJQUFJLENBQUMsT0FBTyxFQUFFLFFBQVEsRUFBRSxFQUFFO1lBQzFELElBQUksT0FBTyxHQUFHLENBQUMsRUFBRTtnQkFDZixPQUFPLEVBQUUsQ0FBQztnQkFDVixPQUFPLENBQUMsUUFBUSxDQUFDLEdBQUcsT0FBTyxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsQ0FBQztnQkFDMUMsU0FBUzthQUNWO1lBRUQsTUFBTSxJQUFJLElBQUksQ0FBQyxVQUFVLENBQUMsUUFBUSxFQUFFLEdBQUcsRUFBRSxJQUFJLENBQUMsQ0FBQztZQUUvQyxJQUFJLE9BQU8sQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLEVBQUU7Z0JBQ3pCLElBQUksQ0FBQyxXQUFXLEVBQUU7b0JBQ2hCLE9BQU8sQ0FBQyxRQUFRLENBQUMsRUFBRSxDQUFDO2lCQUNyQjthQUNGO2lCQUFNLElBQUksQ0FBQyxPQUFPLElBQUksT0FBTyxDQUFDLFFBQVEsQ0FBQyxLQUFLLEdBQUcsQ0FBQyxRQUFRLENBQUMsRUFBRTtnQkFDMUQsT0FBTyxDQUFDLFFBQVEsQ0FBQyxHQUFHLEdBQUcsQ0FBQyxRQUFRLENBQUMsQ0FBQyxVQUFVLEVBQUUsQ0FBQzthQUNoRDtZQUVELE9BQU8sR0FBRyxHQUFHLENBQUMsUUFBUSxDQUFDLENBQUMsVUFBVSxFQUFFLENBQUM7WUFFckMsSUFBSSxPQUFPLENBQUMsUUFBUSxDQUFDLEtBQUssQ0FBQyxJQUFJLEdBQUcsQ0FBQyxRQUFRLENBQUMsQ0FBQyxNQUFNLEVBQUU7Z0JBQ25ELGNBQWMsR0FBRyxJQUFJLENBQUM7YUFDdkI7U0FDRjtRQUVELElBQUksSUFBSSxDQUFDLE9BQU8sR0FBRyxDQUFDLEVBQUU7WUFDcEIsSUFBSSxHQUFHLENBQUMsSUFBSSxDQUFDLE9BQU8sR0FBRyxDQUFDLENBQUMsQ0FBQyxTQUFTLEVBQUUsRUFBRTtnQkFDckMsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEtBQUssQ0FBQzthQUNwQztpQkFBTSxJQUFJLElBQUksQ0FBQyxTQUFTLEVBQUU7Z0JBQ3pCLE1BQU0sSUFBSSxHQUFHLENBQUM7YUFDZjtTQUNGO1FBRUQsTUFBTSxJQUFJLElBQUksQ0FBQztRQUVmLElBQUksY0FBYyxFQUFFO1lBQ2xCLE9BQU8sTUFBTSxHQUFHLElBQUksQ0FBQyxTQUFTLENBQUMsT0FBTyxFQUFFLFFBQVEsRUFBRSxJQUFJLEVBQUUsY0FBYyxDQUFDLENBQUM7U0FDekU7UUFHRCxJQUNFLENBQUMsUUFBUSxLQUFLLENBQUMsSUFBSSxJQUFJLENBQUMsZUFBZSxDQUFDO1lBQ3hDLENBQUMsUUFBUSxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUMsTUFBTSxHQUFHLENBQUMsSUFBSSxJQUFJLENBQUMsYUFBYSxDQUFDLEVBQ3ZEO1lBQ0EsTUFBTSxJQUFJLElBQUksQ0FBQyxlQUFlLENBQUMsR0FBRyxFQUFFLE9BQU8sRUFBRSxPQUFPLEVBQUUsSUFBSSxDQUFDLENBQUM7U0FDN0Q7UUFHRCxJQUFJLFFBQVEsS0FBSyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sR0FBRyxDQUFDLElBQUksR0FBRyxDQUFDLFNBQVMsRUFBRSxFQUFFO1lBQ3hELE1BQU0sSUFBSSxJQUFJLENBQUMsZUFBZSxDQUFDLEdBQUcsRUFBRSxTQUFTLEVBQUUsT0FBTyxFQUFFLElBQUksQ0FBQyxDQUFDO1NBQy9EO1FBRUQsT0FBTyxNQUFNLENBQUM7SUFDaEIsQ0FBQztJQVNTLFVBQVUsQ0FDbEIsUUFBZ0IsRUFDaEIsR0FBYyxFQUNkLElBQXFCLEVBQ3JCLFFBQWtCO1FBRWxCLElBQUksTUFBTSxHQUFHLEVBQUUsQ0FBQztRQUNoQixNQUFNLFFBQVEsR0FBcUIsR0FBRyxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsQ0FBQztRQUVyRCxNQUFNLElBQUksR0FBUyxHQUFHLENBQUMsUUFBUSxDQUFDLENBQUM7UUFFakMsSUFBSSxDQUFDLFFBQVEsRUFBRTtZQUNiLElBQUksUUFBUSxLQUFLLENBQUMsRUFBRTtnQkFDbEIsSUFBSSxJQUFJLENBQUMsU0FBUyxFQUFFLEVBQUU7b0JBQ3BCLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxJQUFJLENBQUM7aUJBQ25DO3FCQUFNLElBQUksSUFBSSxDQUFDLFNBQVMsRUFBRTtvQkFDekIsTUFBTSxJQUFJLEdBQUcsQ0FBQztpQkFDZjthQUNGO2lCQUFNO2dCQUNMLElBQUksSUFBSSxDQUFDLFNBQVMsRUFBRSxJQUFJLFFBQVEsRUFBRSxTQUFTLEVBQUUsRUFBRTtvQkFDN0MsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE1BQU0sQ0FBQztpQkFDckM7cUJBQU0sSUFBSSxJQUFJLENBQUMsU0FBUyxFQUFFO29CQUN6QixNQUFNLElBQUksR0FBRyxDQUFDO2lCQUNmO2FBQ0Y7U0FDRjtRQUVELElBQUksU0FBUyxHQUFXLElBQUksQ0FBQyxLQUFLLENBQUMsUUFBUSxDQUFDLENBQUM7UUFFN0MsTUFBTSxPQUFPLEdBQVcsSUFBSSxDQUFDLFVBQVUsRUFBRSxDQUFDO1FBQzFDLElBQUksT0FBTyxHQUFHLENBQUMsRUFBRTtZQUNmLEtBQUssSUFBSSxDQUFDLEdBQUcsQ0FBQyxFQUFFLENBQUMsR0FBRyxPQUFPLEVBQUUsQ0FBQyxFQUFFLEVBQUU7Z0JBRWhDLFNBQVMsSUFBSSxJQUFJLENBQUMsS0FBSyxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsR0FBRyxJQUFJLENBQUMsT0FBTyxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsQ0FBQztnQkFDbkUsSUFBSSxJQUFJLENBQUMsU0FBUyxFQUFFO29CQUVsQixTQUFTLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxRQUFRLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDO2lCQUM3QzthQUNGO1NBQ0Y7UUFFRCxNQUFNLEVBQUUsT0FBTyxFQUFFLElBQUksRUFBRSxHQUFHLElBQUksQ0FBQyxlQUFlLENBQUMsSUFBSSxFQUFFLFNBQVMsQ0FBQyxDQUFDO1FBRWhFLEdBQUcsQ0FBQyxRQUFRLENBQUMsQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLENBQUM7UUFFN0IsSUFBSSxJQUFJLENBQUMsU0FBUyxFQUFFO1lBQ2xCLE1BQU0sSUFBSSxHQUFHLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsUUFBUSxDQUFDLENBQUMsQ0FBQztTQUM5QztRQUVELE1BQU0sSUFBSSxPQUFPLENBQUM7UUFFbEIsSUFBSSxJQUFJLENBQUMsU0FBUyxJQUFJLFFBQVEsR0FBRyxJQUFJLENBQUMsT0FBTyxHQUFHLENBQUMsRUFBRTtZQUNqRCxNQUFNLElBQUksR0FBRyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLFFBQVEsQ0FBQyxDQUFDLENBQUM7U0FDOUM7UUFFRCxPQUFPLE1BQU0sQ0FBQztJQUNoQixDQUFDO0lBUVMsZUFBZSxDQUN2QixJQUFVLEVBQ1YsU0FBaUI7UUFFakIsTUFBTSxNQUFNLEdBQVcsSUFBSSxDQUFDLEdBQUcsQ0FDN0IsU0FBUyxFQUNULFNBQVMsQ0FBQyxJQUFJLENBQUMsUUFBUSxFQUFFLENBQUMsQ0FDM0IsQ0FBQztRQUNGLElBQUksS0FBSyxHQUFXLFlBQVksQ0FBQyxNQUFNLEVBQUUsSUFBSSxDQUFDLFFBQVEsRUFBRSxDQUFDLENBQUM7UUFHMUQsTUFBTSxTQUFTLEdBQUcsU0FBUyxDQUFDLEtBQUssQ0FBQyxHQUFHLE1BQU0sQ0FBQztRQUM1QyxJQUFJLFNBQVMsRUFBRTtZQUNiLEtBQUssR0FBRyxLQUFLLENBQUMsS0FBSyxDQUFDLENBQUMsRUFBRSxNQUFNLENBQUMsQ0FBQztTQUNoQztRQUdELE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxRQUFRLEVBQUUsQ0FBQyxLQUFLLENBQUMsS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDLFNBQVMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDO1FBQ3ZFLE1BQU0sVUFBVSxHQUFHLFNBQVMsR0FBRyxTQUFTLENBQUMsS0FBSyxDQUFDLENBQUM7UUFHaEQsTUFBTSxLQUFLLEdBQWMsSUFBSSxDQUFDLFFBQVEsRUFBRSxDQUFDO1FBQ3pDLElBQUksT0FBZSxDQUFDO1FBQ3BCLElBQUksVUFBVSxLQUFLLENBQUMsRUFBRTtZQUNwQixPQUFPLEdBQUcsS0FBSyxDQUFDO1NBQ2pCO2FBQU0sSUFBSSxLQUFLLEtBQUssTUFBTSxFQUFFO1lBQzNCLE9BQU8sR0FBRyxLQUFLLEdBQUcsR0FBRyxDQUFDLE1BQU0sQ0FBQyxVQUFVLENBQUMsQ0FBQztTQUMxQzthQUFNLElBQUksS0FBSyxLQUFLLFFBQVEsRUFBRTtZQUM3QixPQUFPLEdBQUcsR0FBRyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLFVBQVUsR0FBRyxDQUFDLENBQUMsQ0FBQyxHQUFHLEtBQUs7Z0JBQ3RELEdBQUcsQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQztTQUN6QzthQUFNLElBQUksS0FBSyxLQUFLLE9BQU8sRUFBRTtZQUM1QixPQUFPLEdBQUcsR0FBRyxDQUFDLE1BQU0sQ0FBQyxVQUFVLENBQUMsR0FBRyxLQUFLLENBQUM7U0FDMUM7YUFBTTtZQUNMLE1BQU0sSUFBSSxLQUFLLENBQUMscUJBQXFCLEdBQUcsS0FBSyxDQUFDLENBQUM7U0FDaEQ7UUFFRCxPQUFPO1lBQ0wsT0FBTztZQUNQLElBQUksRUFBRSxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQztTQUN2QixDQUFDO0lBQ0osQ0FBQztJQVNTLGVBQWUsQ0FDdkIsT0FBOEIsRUFDOUIsT0FBOEIsRUFDOUIsT0FBaUIsRUFDakIsSUFBcUI7UUFFckIsSUFBSSxNQUFNLEdBQUcsRUFBRSxDQUFDO1FBRWhCLElBQUksT0FBTyxHQUFHLENBQUMsQ0FBQztRQUNoQixLQUFLLElBQUksUUFBUSxHQUFHLENBQUMsRUFBRSxRQUFRLEdBQUcsSUFBSSxDQUFDLE9BQU8sRUFBRSxRQUFRLEVBQUUsRUFBRTtZQUMxRCxJQUFJLE9BQU8sQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLEVBQUU7Z0JBQ3pCLElBQUksQ0FBQyxPQUFPLEVBQUU7b0JBQ1osTUFBTSxJQUFJLEtBQUssQ0FBQyxnQkFBZ0IsQ0FBQyxDQUFDO2lCQUNuQztnQkFDRCxJQUFJLE9BQU8sR0FBRyxDQUFDLEVBQUU7b0JBQ2YsT0FBTyxFQUFFLENBQUM7b0JBQ1YsU0FBUztpQkFDVjthQUNGO1lBQ0QsTUFBTSxJQUFJLElBQUksQ0FBQyxnQkFBZ0IsQ0FDN0IsUUFBUSxFQUNSLE9BQU8sRUFDUCxPQUFPLEVBQ1AsT0FBTyxFQUNQLElBQUksQ0FDTCxDQUFDO1lBQ0YsT0FBTyxHQUFHLE9BQU8sRUFBRSxDQUFDLFFBQVEsQ0FBQyxDQUFDLFVBQVUsRUFBRSxJQUFJLENBQUMsQ0FBQztTQUNqRDtRQUVELE9BQU8sTUFBTSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUMsR0FBRyxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxHQUFHLE1BQU0sR0FBRyxJQUFJLENBQUMsQ0FBQyxDQUFDLEVBQUUsQ0FBQztJQUM5RSxDQUFDO0lBVVMsZ0JBQWdCLENBQ3hCLFFBQWdCLEVBQ2hCLE9BQThCLEVBQzlCLE9BQThCLEVBQzlCLE9BQWlCLEVBQ2pCLElBQXFCO1FBTXJCLE1BQU0sRUFBRSxHQUFxQixPQUFPLEVBQUUsQ0FBQyxRQUFRLEdBQUcsQ0FBQyxDQUFDLENBQUM7UUFDckQsTUFBTSxFQUFFLEdBQXFCLE9BQU8sRUFBRSxDQUFDLFFBQVEsR0FBRyxDQUFDLENBQUMsQ0FBQztRQUNyRCxNQUFNLEVBQUUsR0FBcUIsT0FBTyxFQUFFLENBQUMsUUFBUSxDQUFDLENBQUM7UUFDakQsTUFBTSxFQUFFLEdBQXFCLE9BQU8sRUFBRSxDQUFDLFFBQVEsQ0FBQyxDQUFDO1FBRWpELE1BQU0sUUFBUSxHQUFHLENBQUMsQ0FBQyxFQUFFLEVBQUUsU0FBUyxFQUFFLENBQUM7UUFDbkMsTUFBTSxRQUFRLEdBQUcsQ0FBQyxDQUFDLEVBQUUsRUFBRSxTQUFTLEVBQUUsQ0FBQztRQUNuQyxNQUFNLFFBQVEsR0FBRyxDQUFDLENBQUMsRUFBRSxFQUFFLFNBQVMsRUFBRSxDQUFDO1FBQ25DLE1BQU0sUUFBUSxHQUFHLENBQUMsQ0FBQyxFQUFFLEVBQUUsU0FBUyxFQUFFLENBQUM7UUFFbkMsTUFBTSxVQUFVLEdBQUcsQ0FBQyxJQUFzQixFQUFXLEVBQUUsQ0FDckQsQ0FBQyxJQUFJLEVBQUUsVUFBVSxFQUFFLElBQUksQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ2hDLE1BQU0sVUFBVSxHQUFHLENBQUMsSUFBc0IsRUFBVyxFQUFFLENBQ3JELENBQUMsSUFBSSxFQUFFLFVBQVUsRUFBRSxJQUFJLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQztRQUVoQyxJQUFJLE1BQU0sR0FBRyxFQUFFLENBQUM7UUFFaEIsSUFBSSxRQUFRLEtBQUssQ0FBQyxFQUFFO1lBQ2xCLElBQUksT0FBTyxDQUFDLFFBQVEsQ0FBQyxHQUFHLENBQUMsRUFBRTtnQkFDekIsSUFBSSxRQUFRLEVBQUU7b0JBQ1osTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQztpQkFDbkM7cUJBQU07b0JBQ0wsTUFBTSxJQUFJLEdBQUcsQ0FBQztpQkFDZjthQUNGO2lCQUFNLElBQUksUUFBUSxJQUFJLFFBQVEsRUFBRTtnQkFDL0IsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQzthQUN0QztpQkFBTSxJQUFJLFFBQVEsRUFBRTtnQkFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLFVBQVUsQ0FBQzthQUN6QztpQkFBTSxJQUFJLFFBQVEsRUFBRTtnQkFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQzthQUN0QztpQkFBTTtnQkFDTCxNQUFNLElBQUksR0FBRyxDQUFDO2FBQ2Y7U0FDRjthQUFNLElBQUksUUFBUSxHQUFHLElBQUksQ0FBQyxPQUFPLEVBQUU7WUFDbEMsSUFBSSxDQUFDLFFBQVEsSUFBSSxRQUFRLENBQUMsSUFBSSxDQUFDLFFBQVEsSUFBSSxRQUFRLENBQUMsRUFBRTtnQkFDcEQsTUFBTSxTQUFTLEdBQVksVUFBVSxDQUFDLEVBQUUsQ0FBQyxDQUFDO2dCQUMxQyxNQUFNLFNBQVMsR0FBWSxVQUFVLENBQUMsRUFBRSxDQUFDLENBQUM7Z0JBQzFDLE1BQU0sU0FBUyxHQUFZLFVBQVUsQ0FBQyxFQUFFLENBQUMsQ0FBQztnQkFDMUMsTUFBTSxTQUFTLEdBQVksVUFBVSxDQUFDLEVBQUUsQ0FBQyxDQUFDO2dCQUUxQyxNQUFNLFNBQVMsR0FBWSxVQUFVLENBQUMsRUFBRSxDQUFDLENBQUM7Z0JBQzFDLE1BQU0sU0FBUyxHQUFZLFVBQVUsQ0FBQyxFQUFFLENBQUMsQ0FBQztnQkFDMUMsTUFBTSxTQUFTLEdBQVksVUFBVSxDQUFDLEVBQUUsQ0FBQyxDQUFDO2dCQUMxQyxNQUFNLFNBQVMsR0FBWSxVQUFVLENBQUMsRUFBRSxDQUFDLENBQUM7Z0JBRTFDLE1BQU0sWUFBWSxHQUFHLFFBQVEsSUFBSSxRQUFRLElBQUksUUFBUSxJQUFJLFFBQVEsQ0FBQztnQkFDbEUsTUFBTSxhQUFhLEdBQUcsU0FBUyxJQUFJLFNBQVMsSUFBSSxTQUFTLElBQUksU0FBUyxDQUFDO2dCQUN2RSxNQUFNLGFBQWEsR0FBRyxTQUFTLElBQUksU0FBUyxJQUFJLFNBQVMsSUFBSSxTQUFTLENBQUM7Z0JBRXZFLElBQUksYUFBYSxJQUFJLFlBQVksRUFBRTtvQkFDakMsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE1BQU0sQ0FBQztpQkFDckM7cUJBQU0sSUFBSSxhQUFhLElBQUksWUFBWSxJQUFJLEVBQUUsS0FBSyxFQUFFLElBQUksRUFBRSxLQUFLLEVBQUUsRUFBRTtvQkFDbEUsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQztpQkFDbEM7cUJBQU0sSUFBSSxTQUFTLElBQUksU0FBUyxJQUFJLEVBQUUsS0FBSyxFQUFFLEVBQUU7b0JBQzlDLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxNQUFNLENBQUM7aUJBQ3JDO3FCQUFNLElBQUksU0FBUyxJQUFJLFNBQVMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUM5QyxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDO2lCQUN4QztxQkFBTSxJQUFJLFNBQVMsSUFBSSxTQUFTLElBQUksRUFBRSxLQUFLLEVBQUUsRUFBRTtvQkFDOUMsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQztpQkFDdEM7cUJBQU0sSUFBSSxTQUFTLElBQUksU0FBUyxJQUFJLEVBQUUsS0FBSyxFQUFFLEVBQUU7b0JBQzlDLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxRQUFRLENBQUM7aUJBQ3ZDO3FCQUFNO29CQUNMLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxNQUFNLENBQUM7aUJBQ3JDO2FBQ0Y7aUJBQU0sSUFBSSxRQUFRLElBQUksUUFBUSxFQUFFO2dCQUMvQixJQUFJLFVBQVUsQ0FBQyxFQUFFLENBQUMsSUFBSSxVQUFVLENBQUMsRUFBRSxDQUFDLElBQUksRUFBRSxLQUFLLEVBQUUsRUFBRTtvQkFDakQsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE1BQU0sQ0FBQztpQkFDckM7cUJBQU07b0JBQ0wsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLFNBQVMsQ0FBQztpQkFDeEM7YUFDRjtpQkFBTSxJQUFJLFFBQVEsSUFBSSxRQUFRLEVBQUU7Z0JBQy9CLElBQUksT0FBTyxDQUFDLFFBQVEsQ0FBQyxHQUFHLENBQUMsRUFBRTtvQkFDekIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQztpQkFDbkM7cUJBQU07b0JBQ0wsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQztpQkFDdEM7YUFDRjtpQkFBTSxJQUFJLFFBQVEsSUFBSSxRQUFRLEVBQUU7Z0JBQy9CLElBQUksVUFBVSxDQUFDLEVBQUUsQ0FBQyxJQUFJLFVBQVUsQ0FBQyxFQUFFLENBQUMsSUFBSSxFQUFFLEtBQUssRUFBRSxFQUFFO29CQUNqRCxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDO2lCQUNsQztxQkFBTTtvQkFDTCxNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDO2lCQUNyQzthQUNGO2lCQUFNLElBQUksUUFBUSxJQUFJLFFBQVEsRUFBRTtnQkFDL0IsSUFBSSxVQUFVLENBQUMsRUFBRSxDQUFDLElBQUksRUFBRSxLQUFLLEVBQUUsRUFBRTtvQkFDL0IsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEtBQUssQ0FBQztpQkFDcEM7cUJBQU07b0JBQ0wsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLFFBQVEsQ0FBQztpQkFDdkM7YUFDRjtpQkFBTSxJQUFJLFFBQVEsRUFBRTtnQkFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLFdBQVcsQ0FBQzthQUMxQztpQkFBTSxJQUFJLFFBQVEsRUFBRTtnQkFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLFVBQVUsQ0FBQzthQUN6QztpQkFBTSxJQUFJLFFBQVEsRUFBRTtnQkFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLFFBQVEsQ0FBQzthQUN2QztpQkFBTSxJQUFJLFFBQVEsRUFBRTtnQkFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQzthQUN0QztpQkFBTTtnQkFDTCxNQUFNLElBQUksR0FBRyxDQUFDO2FBQ2Y7U0FDRjtRQUVELE1BQU0sTUFBTSxHQUFHLElBQUksQ0FBQyxPQUFPLENBQUMsUUFBUSxDQUFDLEdBQUcsSUFBSSxDQUFDLEtBQUssQ0FBQyxRQUFRLENBQUM7WUFDMUQsSUFBSSxDQUFDLE9BQU8sQ0FBQyxRQUFRLENBQUMsQ0FBQztRQUV6QixJQUFJLE9BQU8sQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLElBQUksT0FBTyxFQUFFO1lBQ3BDLE1BQU0sSUFBSSxJQUFJLENBQUMsVUFBVSxDQUN2QixRQUFRLEVBQ1IsT0FBTyxFQUNQLElBQUksRUFDSixJQUFJLENBQ0wsQ0FBQztZQUNGLElBQUksT0FBTyxDQUFDLFFBQVEsQ0FBQyxLQUFLLE9BQU8sQ0FBQyxPQUFPLENBQUMsTUFBTSxHQUFHLENBQUMsQ0FBQyxFQUFFO2dCQUNyRCxJQUFJLFFBQVEsRUFBRTtvQkFDWixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsS0FBSyxDQUFDO2lCQUNwQztxQkFBTTtvQkFDTCxNQUFNLElBQUksR0FBRyxDQUFDO2lCQUNmO2dCQUNELE9BQU8sTUFBTSxDQUFDO2FBQ2Y7U0FDRjthQUFNLElBQUksUUFBUSxJQUFJLFFBQVEsRUFBRTtZQUMvQixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDLE1BQU0sQ0FBQyxNQUFNLENBQUMsQ0FBQztTQUNqRDthQUFNLElBQUksUUFBUSxFQUFFO1lBQ25CLE1BQU0sSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQyxNQUFNLENBQUMsTUFBTSxDQUFDLE1BQU0sQ0FBQyxDQUFDO1NBQ3BEO2FBQU0sSUFBSSxRQUFRLEVBQUU7WUFDbkIsTUFBTSxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxNQUFNLENBQUMsTUFBTSxDQUFDLENBQUM7U0FDakQ7YUFBTTtZQUNMLE1BQU0sSUFBSSxHQUFHLENBQUMsTUFBTSxDQUFDLE1BQU0sQ0FBQyxDQUFDO1NBQzlCO1FBRUQsSUFBSSxRQUFRLEtBQUssSUFBSSxDQUFDLE9BQU8sR0FBRyxDQUFDLEVBQUU7WUFDakMsSUFBSSxRQUFRLElBQUksUUFBUSxFQUFFO2dCQUN4QixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsUUFBUSxDQUFDO2FBQ3ZDO2lCQUFNLElBQUksUUFBUSxFQUFFO2dCQUNuQixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsV0FBVyxDQUFDO2FBQzFDO2lCQUFNLElBQUksUUFBUSxFQUFFO2dCQUNuQixNQUFNLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUMsUUFBUSxDQUFDO2FBQ3ZDO2lCQUFNO2dCQUNMLE1BQU0sSUFBSSxHQUFHLENBQUM7YUFDZjtTQUNGO1FBRUQsT0FBTyxNQUFNLENBQUM7SUFDaEIsQ0FBQztDQUNGIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgQ2VsbCwgRGlyZWN0aW9uLCBJQ2VsbCB9IGZyb20gXCIuL2NlbGwudHNcIjtcbmltcG9ydCB7IElSb3csIFJvdyB9IGZyb20gXCIuL3Jvdy50c1wiO1xuaW1wb3J0IHR5cGUgeyBJQm9yZGVyT3B0aW9ucywgSVRhYmxlU2V0dGluZ3MsIFRhYmxlIH0gZnJvbSBcIi4vdGFibGUudHNcIjtcbmltcG9ydCB7IGNvbnN1bWVXb3JkcywgbG9uZ2VzdCwgc3RyTGVuZ3RoIH0gZnJvbSBcIi4vdXRpbHMudHNcIjtcblxuLyoqIExheW91dCByZW5kZXIgc2V0dGluZ3MuICovXG5pbnRlcmZhY2UgSVJlbmRlclNldHRpbmdzIHtcbiAgcGFkZGluZzogbnVtYmVyW107XG4gIHdpZHRoOiBudW1iZXJbXTtcbiAgY29sdW1uczogbnVtYmVyO1xuICBoYXNCb3JkZXI6IGJvb2xlYW47XG4gIGhhc0hlYWRlckJvcmRlcjogYm9vbGVhbjtcbiAgaGFzQm9keUJvcmRlcjogYm9vbGVhbjtcbiAgcm93czogUm93PENlbGw+W107XG59XG5cbi8qKiBUYWJsZSBsYXlvdXQgcmVuZGVyZXIuICovXG5leHBvcnQgY2xhc3MgVGFibGVMYXlvdXQge1xuICAvKipcbiAgICogVGFibGUgbGF5b3V0IGNvbnN0cnVjdG9yLlxuICAgKiBAcGFyYW0gdGFibGUgICBUYWJsZSBpbnN0YW5jZS5cbiAgICogQHBhcmFtIG9wdGlvbnMgUmVuZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwdWJsaWMgY29uc3RydWN0b3IoXG4gICAgcHJpdmF0ZSB0YWJsZTogVGFibGUsXG4gICAgcHJpdmF0ZSBvcHRpb25zOiBJVGFibGVTZXR0aW5ncyxcbiAgKSB7fVxuXG4gIC8qKiBHZW5lcmF0ZSB0YWJsZSBzdHJpbmcuICovXG4gIHB1YmxpYyB0b1N0cmluZygpOiBzdHJpbmcge1xuICAgIGNvbnN0IG9wdHM6IElSZW5kZXJTZXR0aW5ncyA9IHRoaXMuY3JlYXRlTGF5b3V0KCk7XG4gICAgcmV0dXJuIG9wdHMucm93cy5sZW5ndGggPyB0aGlzLnJlbmRlclJvd3Mob3B0cykgOiBcIlwiO1xuICB9XG5cbiAgLyoqXG4gICAqIEdlbmVyYXRlcyB0YWJsZSBsYXlvdXQgaW5jbHVkaW5nIHJvdyBhbmQgY29sIHNwYW4sIGNvbnZlcnRzIGFsbCBub25lXG4gICAqIENlbGwvUm93IHZhbHVlcyB0byBDZWxsJ3MgYW5kIFJvdydzIGFuZCByZXR1cm5zIHRoZSBsYXlvdXQgcmVuZGVyaW5nXG4gICAqIHNldHRpbmdzLlxuICAgKi9cbiAgcHJvdGVjdGVkIGNyZWF0ZUxheW91dCgpOiBJUmVuZGVyU2V0dGluZ3Mge1xuICAgIE9iamVjdC5rZXlzKHRoaXMub3B0aW9ucy5jaGFycykuZm9yRWFjaCgoa2V5OiBzdHJpbmcpID0+IHtcbiAgICAgIGlmICh0eXBlb2YgdGhpcy5vcHRpb25zLmNoYXJzW2tleSBhcyBrZXlvZiBJQm9yZGVyT3B0aW9uc10gIT09IFwic3RyaW5nXCIpIHtcbiAgICAgICAgdGhpcy5vcHRpb25zLmNoYXJzW2tleSBhcyBrZXlvZiBJQm9yZGVyT3B0aW9uc10gPSBcIlwiO1xuICAgICAgfVxuICAgIH0pO1xuXG4gICAgY29uc3QgaGFzQm9keUJvcmRlcjogYm9vbGVhbiA9IHRoaXMudGFibGUuZ2V0Qm9yZGVyKCkgfHxcbiAgICAgIHRoaXMudGFibGUuaGFzQm9keUJvcmRlcigpO1xuICAgIGNvbnN0IGhhc0hlYWRlckJvcmRlcjogYm9vbGVhbiA9IHRoaXMudGFibGUuaGFzSGVhZGVyQm9yZGVyKCk7XG4gICAgY29uc3QgaGFzQm9yZGVyOiBib29sZWFuID0gaGFzSGVhZGVyQm9yZGVyIHx8IGhhc0JvZHlCb3JkZXI7XG5cbiAgICBjb25zdCByb3dzID0gdGhpcy4jZ2V0Um93cygpO1xuXG4gICAgY29uc3QgY29sdW1uczogbnVtYmVyID0gTWF0aC5tYXgoLi4ucm93cy5tYXAoKHJvdykgPT4gcm93Lmxlbmd0aCkpO1xuICAgIGZvciAoY29uc3Qgcm93IG9mIHJvd3MpIHtcbiAgICAgIGNvbnN0IGxlbmd0aDogbnVtYmVyID0gcm93Lmxlbmd0aDtcbiAgICAgIGlmIChsZW5ndGggPCBjb2x1bW5zKSB7XG4gICAgICAgIGNvbnN0IGRpZmYgPSBjb2x1bW5zIC0gbGVuZ3RoO1xuICAgICAgICBmb3IgKGxldCBpID0gMDsgaSA8IGRpZmY7IGkrKykge1xuICAgICAgICAgIHJvdy5wdXNoKHRoaXMuY3JlYXRlQ2VsbChudWxsLCByb3cpKTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IHBhZGRpbmc6IG51bWJlcltdID0gW107XG4gICAgY29uc3Qgd2lkdGg6IG51bWJlcltdID0gW107XG4gICAgZm9yIChsZXQgY29sSW5kZXggPSAwOyBjb2xJbmRleCA8IGNvbHVtbnM7IGNvbEluZGV4KyspIHtcbiAgICAgIGNvbnN0IG1pbkNvbFdpZHRoOiBudW1iZXIgPSBBcnJheS5pc0FycmF5KHRoaXMub3B0aW9ucy5taW5Db2xXaWR0aClcbiAgICAgICAgPyB0aGlzLm9wdGlvbnMubWluQ29sV2lkdGhbY29sSW5kZXhdXG4gICAgICAgIDogdGhpcy5vcHRpb25zLm1pbkNvbFdpZHRoO1xuICAgICAgY29uc3QgbWF4Q29sV2lkdGg6IG51bWJlciA9IEFycmF5LmlzQXJyYXkodGhpcy5vcHRpb25zLm1heENvbFdpZHRoKVxuICAgICAgICA/IHRoaXMub3B0aW9ucy5tYXhDb2xXaWR0aFtjb2xJbmRleF1cbiAgICAgICAgOiB0aGlzLm9wdGlvbnMubWF4Q29sV2lkdGg7XG4gICAgICBjb25zdCBjb2xXaWR0aDogbnVtYmVyID0gbG9uZ2VzdChjb2xJbmRleCwgcm93cywgbWF4Q29sV2lkdGgpO1xuICAgICAgd2lkdGhbY29sSW5kZXhdID0gTWF0aC5taW4obWF4Q29sV2lkdGgsIE1hdGgubWF4KG1pbkNvbFdpZHRoLCBjb2xXaWR0aCkpO1xuICAgICAgcGFkZGluZ1tjb2xJbmRleF0gPSBBcnJheS5pc0FycmF5KHRoaXMub3B0aW9ucy5wYWRkaW5nKVxuICAgICAgICA/IHRoaXMub3B0aW9ucy5wYWRkaW5nW2NvbEluZGV4XVxuICAgICAgICA6IHRoaXMub3B0aW9ucy5wYWRkaW5nO1xuICAgIH1cblxuICAgIHJldHVybiB7XG4gICAgICBwYWRkaW5nLFxuICAgICAgd2lkdGgsXG4gICAgICByb3dzLFxuICAgICAgY29sdW1ucyxcbiAgICAgIGhhc0JvcmRlcixcbiAgICAgIGhhc0JvZHlCb3JkZXIsXG4gICAgICBoYXNIZWFkZXJCb3JkZXIsXG4gICAgfTtcbiAgfVxuXG4gICNnZXRSb3dzKCkge1xuICAgIGNvbnN0IGhlYWRlcjogUm93IHwgdW5kZWZpbmVkID0gdGhpcy50YWJsZS5nZXRIZWFkZXIoKTtcbiAgICBjb25zdCByb3dzID0gaGVhZGVyID8gW2hlYWRlciwgLi4udGhpcy50YWJsZV0gOiB0aGlzLnRhYmxlLnNsaWNlKCk7XG4gICAgY29uc3QgaGFzU3BhbiA9IHJvd3Muc29tZSgocm93KSA9PlxuICAgICAgcm93LnNvbWUoKGNlbGwpID0+XG4gICAgICAgIGNlbGwgaW5zdGFuY2VvZiBDZWxsICYmIChjZWxsLmdldENvbFNwYW4oKSA+IDEgfHwgY2VsbC5nZXRSb3dTcGFuKCkgPiAxKVxuICAgICAgKVxuICAgICk7XG5cbiAgICBpZiAoaGFzU3Bhbikge1xuICAgICAgcmV0dXJuIHRoaXMuc3BhblJvd3Mocm93cyk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJvd3MubWFwKChyb3cpID0+IHtcbiAgICAgIGNvbnN0IG5ld1JvdyA9IHRoaXMuY3JlYXRlUm93KHJvdyk7XG4gICAgICByZXR1cm4gbmV3Um93Lm1hcCgoY2VsbCkgPT4gdGhpcy5jcmVhdGVDZWxsKGNlbGwsIG5ld1JvdykpO1xuICAgIH0pIGFzIEFycmF5PFJvdzxDZWxsPj47XG4gIH1cblxuICAvKipcbiAgICogRmlsbHMgcm93cyBhbmQgY29scyBieSBzcGVjaWZpZWQgcm93L2NvbCBzcGFuIHdpdGggYSByZWZlcmVuY2Ugb2YgdGhlXG4gICAqIG9yaWdpbmFsIGNlbGwuXG4gICAqXG4gICAqIEBwYXJhbSBfcm93cyAgICAgQWxsIHRhYmxlIHJvd3MuXG4gICAqIEBwYXJhbSByb3dJbmRleCAgQ3VycmVudCByb3cgaW5kZXguXG4gICAqIEBwYXJhbSBjb2xJbmRleCAgQ3VycmVudCBjb2wgaW5kZXguXG4gICAqIEBwYXJhbSByb3dTcGFuICAgQ3VycmVudCByb3cgc3Bhbi5cbiAgICogQHBhcmFtIGNvbFNwYW4gICBDdXJyZW50IGNvbCBzcGFuLlxuICAgKi9cbiAgcHJvdGVjdGVkIHNwYW5Sb3dzKFxuICAgIF9yb3dzOiBJUm93W10sXG4gICAgcm93SW5kZXggPSAwLFxuICAgIGNvbEluZGV4ID0gMCxcbiAgICByb3dTcGFuOiBudW1iZXJbXSA9IFtdLFxuICAgIGNvbFNwYW4gPSAxLFxuICApOiBSb3c8Q2VsbD5bXSB7XG4gICAgY29uc3Qgcm93czogUm93PENlbGw+W10gPSBfcm93cyBhcyBSb3c8Q2VsbD5bXTtcblxuICAgIGlmIChyb3dJbmRleCA+PSByb3dzLmxlbmd0aCAmJiByb3dTcGFuLmV2ZXJ5KChzcGFuKSA9PiBzcGFuID09PSAxKSkge1xuICAgICAgcmV0dXJuIHJvd3M7XG4gICAgfSBlbHNlIGlmIChcbiAgICAgIHJvd3Nbcm93SW5kZXhdICYmIGNvbEluZGV4ID49IHJvd3Nbcm93SW5kZXhdLmxlbmd0aCAmJlxuICAgICAgY29sSW5kZXggPj0gcm93U3Bhbi5sZW5ndGggJiYgY29sU3BhbiA9PT0gMVxuICAgICkge1xuICAgICAgcmV0dXJuIHRoaXMuc3BhblJvd3Mocm93cywgKytyb3dJbmRleCwgMCwgcm93U3BhbiwgMSk7XG4gICAgfVxuXG4gICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICBjb2xTcGFuLS07XG4gICAgICByb3dTcGFuW2NvbEluZGV4XSA9IHJvd1NwYW5bY29sSW5kZXggLSAxXTtcbiAgICAgIHJvd3Nbcm93SW5kZXhdLnNwbGljZShjb2xJbmRleCAtIDEsIDAsIHJvd3Nbcm93SW5kZXhdW2NvbEluZGV4IC0gMV0pO1xuICAgICAgcmV0dXJuIHRoaXMuc3BhblJvd3Mocm93cywgcm93SW5kZXgsICsrY29sSW5kZXgsIHJvd1NwYW4sIGNvbFNwYW4pO1xuICAgIH1cblxuICAgIGlmIChjb2xJbmRleCA9PT0gMCkge1xuICAgICAgcm93c1tyb3dJbmRleF0gPSB0aGlzLmNyZWF0ZVJvdyhyb3dzW3Jvd0luZGV4XSB8fCBbXSk7XG4gICAgfVxuXG4gICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSkge1xuICAgICAgcm93U3Bhbltjb2xJbmRleF0tLTtcbiAgICAgIHJvd3Nbcm93SW5kZXhdLnNwbGljZShjb2xJbmRleCwgMCwgcm93c1tyb3dJbmRleCAtIDFdW2NvbEluZGV4XSk7XG4gICAgICByZXR1cm4gdGhpcy5zcGFuUm93cyhyb3dzLCByb3dJbmRleCwgKytjb2xJbmRleCwgcm93U3BhbiwgY29sU3Bhbik7XG4gICAgfVxuXG4gICAgcm93c1tyb3dJbmRleF1bY29sSW5kZXhdID0gdGhpcy5jcmVhdGVDZWxsKFxuICAgICAgcm93c1tyb3dJbmRleF1bY29sSW5kZXhdIHx8IG51bGwsXG4gICAgICByb3dzW3Jvd0luZGV4XSxcbiAgICApO1xuXG4gICAgY29sU3BhbiA9IHJvd3Nbcm93SW5kZXhdW2NvbEluZGV4XS5nZXRDb2xTcGFuKCk7XG4gICAgcm93U3Bhbltjb2xJbmRleF0gPSByb3dzW3Jvd0luZGV4XVtjb2xJbmRleF0uZ2V0Um93U3BhbigpO1xuXG4gICAgcmV0dXJuIHRoaXMuc3BhblJvd3Mocm93cywgcm93SW5kZXgsICsrY29sSW5kZXgsIHJvd1NwYW4sIGNvbFNwYW4pO1xuICB9XG5cbiAgLyoqXG4gICAqIENyZWF0ZSBhIG5ldyByb3cgZnJvbSBleGlzdGluZyByb3cgb3IgY2VsbCBhcnJheS5cbiAgICogQHBhcmFtIHJvdyBPcmlnaW5hbCByb3cuXG4gICAqL1xuICBwcm90ZWN0ZWQgY3JlYXRlUm93KHJvdzogSVJvdyk6IFJvdzxDZWxsPiB7XG4gICAgcmV0dXJuIFJvdy5mcm9tKHJvdylcbiAgICAgIC5ib3JkZXIodGhpcy50YWJsZS5nZXRCb3JkZXIoKSwgZmFsc2UpXG4gICAgICAuYWxpZ24odGhpcy50YWJsZS5nZXRBbGlnbigpLCBmYWxzZSkgYXMgUm93PENlbGw+O1xuICB9XG5cbiAgLyoqXG4gICAqIENyZWF0ZSBhIG5ldyBjZWxsIGZyb20gZXhpc3RpbmcgY2VsbCBvciBjZWxsIHZhbHVlLlxuICAgKiBAcGFyYW0gY2VsbCAgT3JpZ2luYWwgY2VsbC5cbiAgICogQHBhcmFtIHJvdyAgIFBhcmVudCByb3cuXG4gICAqL1xuICBwcm90ZWN0ZWQgY3JlYXRlQ2VsbChjZWxsOiBJQ2VsbCB8IG51bGwsIHJvdzogUm93KTogQ2VsbCB7XG4gICAgcmV0dXJuIENlbGwuZnJvbShjZWxsID8/IFwiXCIpXG4gICAgICAuYm9yZGVyKHJvdy5nZXRCb3JkZXIoKSwgZmFsc2UpXG4gICAgICAuYWxpZ24ocm93LmdldEFsaWduKCksIGZhbHNlKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgdGFibGUgbGF5b3V0LlxuICAgKiBAcGFyYW0gb3B0cyBSZW5kZXIgb3B0aW9ucy5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJSb3dzKG9wdHM6IElSZW5kZXJTZXR0aW5ncyk6IHN0cmluZyB7XG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG4gICAgY29uc3Qgcm93U3BhbjogbnVtYmVyW10gPSBuZXcgQXJyYXkob3B0cy5jb2x1bW5zKS5maWxsKDEpO1xuXG4gICAgZm9yIChsZXQgcm93SW5kZXggPSAwOyByb3dJbmRleCA8IG9wdHMucm93cy5sZW5ndGg7IHJvd0luZGV4KyspIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlclJvdyhyb3dTcGFuLCByb3dJbmRleCwgb3B0cyk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdC5zbGljZSgwLCAtMSk7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIHJvdy5cbiAgICogQHBhcmFtIHJvd1NwYW4gICAgIEN1cnJlbnQgcm93IHNwYW4uXG4gICAqIEBwYXJhbSByb3dJbmRleCAgICBDdXJyZW50IHJvdyBpbmRleC5cbiAgICogQHBhcmFtIG9wdHMgICAgICAgIFJlbmRlciBvcHRpb25zLlxuICAgKiBAcGFyYW0gaXNNdWx0aWxpbmUgSXMgbXVsdGlsaW5lIHJvdy5cbiAgICovXG4gIHByb3RlY3RlZCByZW5kZXJSb3coXG4gICAgcm93U3BhbjogbnVtYmVyW10sXG4gICAgcm93SW5kZXg6IG51bWJlcixcbiAgICBvcHRzOiBJUmVuZGVyU2V0dGluZ3MsXG4gICAgaXNNdWx0aWxpbmU/OiBib29sZWFuLFxuICApOiBzdHJpbmcge1xuICAgIGNvbnN0IHJvdzogUm93PENlbGw+ID0gb3B0cy5yb3dzW3Jvd0luZGV4XTtcbiAgICBjb25zdCBwcmV2Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQgPSBvcHRzLnJvd3Nbcm93SW5kZXggLSAxXTtcbiAgICBjb25zdCBuZXh0Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQgPSBvcHRzLnJvd3Nbcm93SW5kZXggKyAxXTtcbiAgICBsZXQgcmVzdWx0ID0gXCJcIjtcblxuICAgIGxldCBjb2xTcGFuID0gMTtcblxuICAgIC8vIGJvcmRlciB0b3Agcm93XG4gICAgaWYgKCFpc011bHRpbGluZSAmJiByb3dJbmRleCA9PT0gMCAmJiByb3cuaGFzQm9yZGVyKCkpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckJvcmRlclJvdyh1bmRlZmluZWQsIHJvdywgcm93U3Bhbiwgb3B0cyk7XG4gICAgfVxuXG4gICAgbGV0IGlzTXVsdGlsaW5lUm93ID0gZmFsc2U7XG5cbiAgICByZXN1bHQgKz0gXCIgXCIucmVwZWF0KHRoaXMub3B0aW9ucy5pbmRlbnQgfHwgMCk7XG5cbiAgICBmb3IgKGxldCBjb2xJbmRleCA9IDA7IGNvbEluZGV4IDwgb3B0cy5jb2x1bW5zOyBjb2xJbmRleCsrKSB7XG4gICAgICBpZiAoY29sU3BhbiA+IDEpIHtcbiAgICAgICAgY29sU3Bhbi0tO1xuICAgICAgICByb3dTcGFuW2NvbEluZGV4XSA9IHJvd1NwYW5bY29sSW5kZXggLSAxXTtcbiAgICAgICAgY29udGludWU7XG4gICAgICB9XG5cbiAgICAgIHJlc3VsdCArPSB0aGlzLnJlbmRlckNlbGwoY29sSW5kZXgsIHJvdywgb3B0cyk7XG5cbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgaWYgKCFpc011bHRpbGluZSkge1xuICAgICAgICAgIHJvd1NwYW5bY29sSW5kZXhdLS07XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoIXByZXZSb3cgfHwgcHJldlJvd1tjb2xJbmRleF0gIT09IHJvd1tjb2xJbmRleF0pIHtcbiAgICAgICAgcm93U3Bhbltjb2xJbmRleF0gPSByb3dbY29sSW5kZXhdLmdldFJvd1NwYW4oKTtcbiAgICAgIH1cblxuICAgICAgY29sU3BhbiA9IHJvd1tjb2xJbmRleF0uZ2V0Q29sU3BhbigpO1xuXG4gICAgICBpZiAocm93U3Bhbltjb2xJbmRleF0gPT09IDEgJiYgcm93W2NvbEluZGV4XS5sZW5ndGgpIHtcbiAgICAgICAgaXNNdWx0aWxpbmVSb3cgPSB0cnVlO1xuICAgICAgfVxuICAgIH1cblxuICAgIGlmIChvcHRzLmNvbHVtbnMgPiAwKSB7XG4gICAgICBpZiAocm93W29wdHMuY29sdW1ucyAtIDFdLmdldEJvcmRlcigpKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHQ7XG4gICAgICB9IGVsc2UgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXN1bHQgKz0gXCJcXG5cIjtcblxuICAgIGlmIChpc011bHRpbGluZVJvdykgeyAvLyBza2lwIGJvcmRlclxuICAgICAgcmV0dXJuIHJlc3VsdCArIHRoaXMucmVuZGVyUm93KHJvd1NwYW4sIHJvd0luZGV4LCBvcHRzLCBpc011bHRpbGluZVJvdyk7XG4gICAgfVxuXG4gICAgLy8gYm9yZGVyIG1pZCByb3dcbiAgICBpZiAoXG4gICAgICAocm93SW5kZXggPT09IDAgJiYgb3B0cy5oYXNIZWFkZXJCb3JkZXIpIHx8XG4gICAgICAocm93SW5kZXggPCBvcHRzLnJvd3MubGVuZ3RoIC0gMSAmJiBvcHRzLmhhc0JvZHlCb3JkZXIpXG4gICAgKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJSb3cocm93LCBuZXh0Um93LCByb3dTcGFuLCBvcHRzKTtcbiAgICB9XG5cbiAgICAvLyBib3JkZXIgYm90dG9tIHJvd1xuICAgIGlmIChyb3dJbmRleCA9PT0gb3B0cy5yb3dzLmxlbmd0aCAtIDEgJiYgcm93Lmhhc0JvcmRlcigpKSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJSb3cocm93LCB1bmRlZmluZWQsIHJvd1NwYW4sIG9wdHMpO1xuICAgIH1cblxuICAgIHJldHVybiByZXN1bHQ7XG4gIH1cblxuICAvKipcbiAgICogUmVuZGVyIGNlbGwuXG4gICAqIEBwYXJhbSBjb2xJbmRleCAgQ3VycmVudCBjb2wgaW5kZXguXG4gICAqIEBwYXJhbSByb3cgICAgICAgQ3VycmVudCByb3cuXG4gICAqIEBwYXJhbSBvcHRzICAgICAgUmVuZGVyIG9wdGlvbnMuXG4gICAqIEBwYXJhbSBub0JvcmRlciAgRGlzYWJsZSBib3JkZXIuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQ2VsbChcbiAgICBjb2xJbmRleDogbnVtYmVyLFxuICAgIHJvdzogUm93PENlbGw+LFxuICAgIG9wdHM6IElSZW5kZXJTZXR0aW5ncyxcbiAgICBub0JvcmRlcj86IGJvb2xlYW4sXG4gICk6IHN0cmluZyB7XG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG4gICAgY29uc3QgcHJldkNlbGw6IENlbGwgfCB1bmRlZmluZWQgPSByb3dbY29sSW5kZXggLSAxXTtcblxuICAgIGNvbnN0IGNlbGw6IENlbGwgPSByb3dbY29sSW5kZXhdO1xuXG4gICAgaWYgKCFub0JvcmRlcikge1xuICAgICAgaWYgKGNvbEluZGV4ID09PSAwKSB7XG4gICAgICAgIGlmIChjZWxsLmdldEJvcmRlcigpKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0O1xuICAgICAgICB9IGVsc2UgaWYgKG9wdHMuaGFzQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBpZiAoY2VsbC5nZXRCb3JkZXIoKSB8fCBwcmV2Q2VsbD8uZ2V0Qm9yZGVyKCkpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLm1pZGRsZTtcbiAgICAgICAgfSBlbHNlIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cblxuICAgIGxldCBtYXhMZW5ndGg6IG51bWJlciA9IG9wdHMud2lkdGhbY29sSW5kZXhdO1xuXG4gICAgY29uc3QgY29sU3BhbjogbnVtYmVyID0gY2VsbC5nZXRDb2xTcGFuKCk7XG4gICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICBmb3IgKGxldCBvID0gMTsgbyA8IGNvbFNwYW47IG8rKykge1xuICAgICAgICAvLyBhZGQgcGFkZGluZyBhbmQgd2l0aCBvZiBuZXh0IGNlbGxcbiAgICAgICAgbWF4TGVuZ3RoICs9IG9wdHMud2lkdGhbY29sSW5kZXggKyBvXSArIG9wdHMucGFkZGluZ1tjb2xJbmRleCArIG9dO1xuICAgICAgICBpZiAob3B0cy5oYXNCb3JkZXIpIHtcbiAgICAgICAgICAvLyBhZGQgcGFkZGluZyBhZ2FpbiBhbmQgYm9yZGVyIHdpdGhcbiAgICAgICAgICBtYXhMZW5ndGggKz0gb3B0cy5wYWRkaW5nW2NvbEluZGV4ICsgb10gKyAxO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgY29uc3QgeyBjdXJyZW50LCBuZXh0IH0gPSB0aGlzLnJlbmRlckNlbGxWYWx1ZShjZWxsLCBtYXhMZW5ndGgpO1xuXG4gICAgcm93W2NvbEluZGV4XS5zZXRWYWx1ZShuZXh0KTtcblxuICAgIGlmIChvcHRzLmhhc0JvcmRlcikge1xuICAgICAgcmVzdWx0ICs9IFwiIFwiLnJlcGVhdChvcHRzLnBhZGRpbmdbY29sSW5kZXhdKTtcbiAgICB9XG5cbiAgICByZXN1bHQgKz0gY3VycmVudDtcblxuICAgIGlmIChvcHRzLmhhc0JvcmRlciB8fCBjb2xJbmRleCA8IG9wdHMuY29sdW1ucyAtIDEpIHtcbiAgICAgIHJlc3VsdCArPSBcIiBcIi5yZXBlYXQob3B0cy5wYWRkaW5nW2NvbEluZGV4XSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHJlc3VsdDtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgc3BlY2lmaWVkIGxlbmd0aCBvZiBjZWxsLiBSZXR1cm5zIHRoZSByZW5kZXJlZCB2YWx1ZSBhbmQgYSBuZXcgY2VsbFxuICAgKiB3aXRoIHRoZSByZXN0IHZhbHVlLlxuICAgKiBAcGFyYW0gY2VsbCAgICAgIENlbGwgdG8gcmVuZGVyLlxuICAgKiBAcGFyYW0gbWF4TGVuZ3RoIE1heCBsZW5ndGggb2YgY29udGVudCB0byByZW5kZXIuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQ2VsbFZhbHVlKFxuICAgIGNlbGw6IENlbGwsXG4gICAgbWF4TGVuZ3RoOiBudW1iZXIsXG4gICk6IHsgY3VycmVudDogc3RyaW5nOyBuZXh0OiBDZWxsIH0ge1xuICAgIGNvbnN0IGxlbmd0aDogbnVtYmVyID0gTWF0aC5taW4oXG4gICAgICBtYXhMZW5ndGgsXG4gICAgICBzdHJMZW5ndGgoY2VsbC50b1N0cmluZygpKSxcbiAgICApO1xuICAgIGxldCB3b3Jkczogc3RyaW5nID0gY29uc3VtZVdvcmRzKGxlbmd0aCwgY2VsbC50b1N0cmluZygpKTtcblxuICAgIC8vIGJyZWFrIHdvcmQgaWYgd29yZCBpcyBsb25nZXIgdGhhbiBtYXggbGVuZ3RoXG4gICAgY29uc3QgYnJlYWtXb3JkID0gc3RyTGVuZ3RoKHdvcmRzKSA+IGxlbmd0aDtcbiAgICBpZiAoYnJlYWtXb3JkKSB7XG4gICAgICB3b3JkcyA9IHdvcmRzLnNsaWNlKDAsIGxlbmd0aCk7XG4gICAgfVxuXG4gICAgLy8gZ2V0IG5leHQgY29udGVudCBhbmQgcmVtb3ZlIGxlYWRpbmcgc3BhY2UgaWYgYnJlYWtXb3JkIGlzIG5vdCB0cnVlXG4gICAgY29uc3QgbmV4dCA9IGNlbGwudG9TdHJpbmcoKS5zbGljZSh3b3Jkcy5sZW5ndGggKyAoYnJlYWtXb3JkID8gMCA6IDEpKTtcbiAgICBjb25zdCBmaWxsTGVuZ3RoID0gbWF4TGVuZ3RoIC0gc3RyTGVuZ3RoKHdvcmRzKTtcblxuICAgIC8vIEFsaWduIGNvbnRlbnRcbiAgICBjb25zdCBhbGlnbjogRGlyZWN0aW9uID0gY2VsbC5nZXRBbGlnbigpO1xuICAgIGxldCBjdXJyZW50OiBzdHJpbmc7XG4gICAgaWYgKGZpbGxMZW5ndGggPT09IDApIHtcbiAgICAgIGN1cnJlbnQgPSB3b3JkcztcbiAgICB9IGVsc2UgaWYgKGFsaWduID09PSBcImxlZnRcIikge1xuICAgICAgY3VycmVudCA9IHdvcmRzICsgXCIgXCIucmVwZWF0KGZpbGxMZW5ndGgpO1xuICAgIH0gZWxzZSBpZiAoYWxpZ24gPT09IFwiY2VudGVyXCIpIHtcbiAgICAgIGN1cnJlbnQgPSBcIiBcIi5yZXBlYXQoTWF0aC5mbG9vcihmaWxsTGVuZ3RoIC8gMikpICsgd29yZHMgK1xuICAgICAgICBcIiBcIi5yZXBlYXQoTWF0aC5jZWlsKGZpbGxMZW5ndGggLyAyKSk7XG4gICAgfSBlbHNlIGlmIChhbGlnbiA9PT0gXCJyaWdodFwiKSB7XG4gICAgICBjdXJyZW50ID0gXCIgXCIucmVwZWF0KGZpbGxMZW5ndGgpICsgd29yZHM7XG4gICAgfSBlbHNlIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihcIlVua25vd24gZGlyZWN0aW9uOiBcIiArIGFsaWduKTtcbiAgICB9XG5cbiAgICByZXR1cm4ge1xuICAgICAgY3VycmVudCxcbiAgICAgIG5leHQ6IGNlbGwuY2xvbmUobmV4dCksXG4gICAgfTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZW5kZXIgYm9yZGVyIHJvdy5cbiAgICogQHBhcmFtIHByZXZSb3cgUHJldmlvdXMgcm93LlxuICAgKiBAcGFyYW0gbmV4dFJvdyBOZXh0IHJvdy5cbiAgICogQHBhcmFtIHJvd1NwYW4gQ3VycmVudCByb3cgc3Bhbi5cbiAgICogQHBhcmFtIG9wdHMgICAgUmVuZGVyIG9wdGlvbnMuXG4gICAqL1xuICBwcm90ZWN0ZWQgcmVuZGVyQm9yZGVyUm93KFxuICAgIHByZXZSb3c6IFJvdzxDZWxsPiB8IHVuZGVmaW5lZCxcbiAgICBuZXh0Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQsXG4gICAgcm93U3BhbjogbnVtYmVyW10sXG4gICAgb3B0czogSVJlbmRlclNldHRpbmdzLFxuICApOiBzdHJpbmcge1xuICAgIGxldCByZXN1bHQgPSBcIlwiO1xuXG4gICAgbGV0IGNvbFNwYW4gPSAxO1xuICAgIGZvciAobGV0IGNvbEluZGV4ID0gMDsgY29sSW5kZXggPCBvcHRzLmNvbHVtbnM7IGNvbEluZGV4KyspIHtcbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgaWYgKCFuZXh0Um93KSB7XG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiaW52YWxpZCBsYXlvdXRcIik7XG4gICAgICAgIH1cbiAgICAgICAgaWYgKGNvbFNwYW4gPiAxKSB7XG4gICAgICAgICAgY29sU3Bhbi0tO1xuICAgICAgICAgIGNvbnRpbnVlO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJCb3JkZXJDZWxsKFxuICAgICAgICBjb2xJbmRleCxcbiAgICAgICAgcHJldlJvdyxcbiAgICAgICAgbmV4dFJvdyxcbiAgICAgICAgcm93U3BhbixcbiAgICAgICAgb3B0cyxcbiAgICAgICk7XG4gICAgICBjb2xTcGFuID0gbmV4dFJvdz8uW2NvbEluZGV4XS5nZXRDb2xTcGFuKCkgPz8gMTtcbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0Lmxlbmd0aCA/IFwiIFwiLnJlcGVhdCh0aGlzLm9wdGlvbnMuaW5kZW50KSArIHJlc3VsdCArIFwiXFxuXCIgOiBcIlwiO1xuICB9XG5cbiAgLyoqXG4gICAqIFJlbmRlciBib3JkZXIgY2VsbC5cbiAgICogQHBhcmFtIGNvbEluZGV4ICBDdXJyZW50IGluZGV4LlxuICAgKiBAcGFyYW0gcHJldlJvdyAgIFByZXZpb3VzIHJvdy5cbiAgICogQHBhcmFtIG5leHRSb3cgICBOZXh0IHJvdy5cbiAgICogQHBhcmFtIHJvd1NwYW4gICBDdXJyZW50IHJvdyBzcGFuLlxuICAgKiBAcGFyYW0gb3B0cyAgICAgIFJlbmRlciBvcHRpb25zLlxuICAgKi9cbiAgcHJvdGVjdGVkIHJlbmRlckJvcmRlckNlbGwoXG4gICAgY29sSW5kZXg6IG51bWJlcixcbiAgICBwcmV2Um93OiBSb3c8Q2VsbD4gfCB1bmRlZmluZWQsXG4gICAgbmV4dFJvdzogUm93PENlbGw+IHwgdW5kZWZpbmVkLFxuICAgIHJvd1NwYW46IG51bWJlcltdLFxuICAgIG9wdHM6IElSZW5kZXJTZXR0aW5ncyxcbiAgKTogc3RyaW5nIHtcbiAgICAvLyBhMSB8IGIxXG4gICAgLy8gLS0tLS0tLVxuICAgIC8vIGEyIHwgYjJcblxuICAgIGNvbnN0IGExOiBDZWxsIHwgdW5kZWZpbmVkID0gcHJldlJvdz8uW2NvbEluZGV4IC0gMV07XG4gICAgY29uc3QgYTI6IENlbGwgfCB1bmRlZmluZWQgPSBuZXh0Um93Py5bY29sSW5kZXggLSAxXTtcbiAgICBjb25zdCBiMTogQ2VsbCB8IHVuZGVmaW5lZCA9IHByZXZSb3c/Lltjb2xJbmRleF07XG4gICAgY29uc3QgYjI6IENlbGwgfCB1bmRlZmluZWQgPSBuZXh0Um93Py5bY29sSW5kZXhdO1xuXG4gICAgY29uc3QgYTFCb3JkZXIgPSAhIWExPy5nZXRCb3JkZXIoKTtcbiAgICBjb25zdCBhMkJvcmRlciA9ICEhYTI/LmdldEJvcmRlcigpO1xuICAgIGNvbnN0IGIxQm9yZGVyID0gISFiMT8uZ2V0Qm9yZGVyKCk7XG4gICAgY29uc3QgYjJCb3JkZXIgPSAhIWIyPy5nZXRCb3JkZXIoKTtcblxuICAgIGNvbnN0IGhhc0NvbFNwYW4gPSAoY2VsbDogQ2VsbCB8IHVuZGVmaW5lZCk6IGJvb2xlYW4gPT5cbiAgICAgIChjZWxsPy5nZXRDb2xTcGFuKCkgPz8gMSkgPiAxO1xuICAgIGNvbnN0IGhhc1Jvd1NwYW4gPSAoY2VsbDogQ2VsbCB8IHVuZGVmaW5lZCk6IGJvb2xlYW4gPT5cbiAgICAgIChjZWxsPy5nZXRSb3dTcGFuKCkgPz8gMSkgPiAxO1xuXG4gICAgbGV0IHJlc3VsdCA9IFwiXCI7XG5cbiAgICBpZiAoY29sSW5kZXggPT09IDApIHtcbiAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0O1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChiMUJvcmRlciAmJiBiMkJvcmRlcikge1xuICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnRNaWQ7XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTGVmdDtcbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BMZWZ0O1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgfVxuICAgIH0gZWxzZSBpZiAoY29sSW5kZXggPCBvcHRzLmNvbHVtbnMpIHtcbiAgICAgIGlmICgoYTFCb3JkZXIgJiYgYjJCb3JkZXIpIHx8IChiMUJvcmRlciAmJiBhMkJvcmRlcikpIHtcbiAgICAgICAgY29uc3QgYTFDb2xTcGFuOiBib29sZWFuID0gaGFzQ29sU3BhbihhMSk7XG4gICAgICAgIGNvbnN0IGEyQ29sU3BhbjogYm9vbGVhbiA9IGhhc0NvbFNwYW4oYTIpO1xuICAgICAgICBjb25zdCBiMUNvbFNwYW46IGJvb2xlYW4gPSBoYXNDb2xTcGFuKGIxKTtcbiAgICAgICAgY29uc3QgYjJDb2xTcGFuOiBib29sZWFuID0gaGFzQ29sU3BhbihiMik7XG5cbiAgICAgICAgY29uc3QgYTFSb3dTcGFuOiBib29sZWFuID0gaGFzUm93U3BhbihhMSk7XG4gICAgICAgIGNvbnN0IGEyUm93U3BhbjogYm9vbGVhbiA9IGhhc1Jvd1NwYW4oYTIpO1xuICAgICAgICBjb25zdCBiMVJvd1NwYW46IGJvb2xlYW4gPSBoYXNSb3dTcGFuKGIxKTtcbiAgICAgICAgY29uc3QgYjJSb3dTcGFuOiBib29sZWFuID0gaGFzUm93U3BhbihiMik7XG5cbiAgICAgICAgY29uc3QgaGFzQWxsQm9yZGVyID0gYTFCb3JkZXIgJiYgYjJCb3JkZXIgJiYgYjFCb3JkZXIgJiYgYTJCb3JkZXI7XG4gICAgICAgIGNvbnN0IGhhc0FsbFJvd1NwYW4gPSBhMVJvd1NwYW4gJiYgYjFSb3dTcGFuICYmIGEyUm93U3BhbiAmJiBiMlJvd1NwYW47XG4gICAgICAgIGNvbnN0IGhhc0FsbENvbFNwYW4gPSBhMUNvbFNwYW4gJiYgYjFDb2xTcGFuICYmIGEyQ29sU3BhbiAmJiBiMkNvbFNwYW47XG5cbiAgICAgICAgaWYgKGhhc0FsbFJvd1NwYW4gJiYgaGFzQWxsQm9yZGVyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWRkbGU7XG4gICAgICAgIH0gZWxzZSBpZiAoaGFzQWxsQ29sU3BhbiAmJiBoYXNBbGxCb3JkZXIgJiYgYTEgPT09IGIxICYmIGEyID09PSBiMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkO1xuICAgICAgICB9IGVsc2UgaWYgKGExQ29sU3BhbiAmJiBiMUNvbFNwYW4gJiYgYTEgPT09IGIxKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BNaWQ7XG4gICAgICAgIH0gZWxzZSBpZiAoYTJDb2xTcGFuICYmIGIyQ29sU3BhbiAmJiBhMiA9PT0gYjIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmJvdHRvbU1pZDtcbiAgICAgICAgfSBlbHNlIGlmIChhMVJvd1NwYW4gJiYgYTJSb3dTcGFuICYmIGExID09PSBhMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubGVmdE1pZDtcbiAgICAgICAgfSBlbHNlIGlmIChiMVJvd1NwYW4gJiYgYjJSb3dTcGFuICYmIGIxID09PSBiMikge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMucmlnaHRNaWQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5taWRNaWQ7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoYTFCb3JkZXIgJiYgYjFCb3JkZXIpIHtcbiAgICAgICAgaWYgKGhhc0NvbFNwYW4oYTEpICYmIGhhc0NvbFNwYW4oYjEpICYmIGExID09PSBiMSkge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyICYmIGIyQm9yZGVyKSB7XG4gICAgICAgIGlmIChyb3dTcGFuW2NvbEluZGV4XSA+IDEpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLmxlZnQ7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5sZWZ0TWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGIyQm9yZGVyICYmIGEyQm9yZGVyKSB7XG4gICAgICAgIGlmIChoYXNDb2xTcGFuKGEyKSAmJiBoYXNDb2xTcGFuKGIyKSAmJiBhMiA9PT0gYjIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnRvcE1pZDtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChhMUJvcmRlciAmJiBhMkJvcmRlcikge1xuICAgICAgICBpZiAoaGFzUm93U3BhbihhMSkgJiYgYTEgPT09IGEyKSB7XG4gICAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodDtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0TWlkO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGExQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tUmlnaHQ7XG4gICAgICB9IGVsc2UgaWYgKGIxQm9yZGVyKSB7XG4gICAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tTGVmdDtcbiAgICAgIH0gZWxzZSBpZiAoYTJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BSaWdodDtcbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BMZWZ0O1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmVzdWx0ICs9IFwiIFwiO1xuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IGxlbmd0aCA9IG9wdHMucGFkZGluZ1tjb2xJbmRleF0gKyBvcHRzLndpZHRoW2NvbEluZGV4XSArXG4gICAgICBvcHRzLnBhZGRpbmdbY29sSW5kZXhdO1xuXG4gICAgaWYgKHJvd1NwYW5bY29sSW5kZXhdID4gMSAmJiBuZXh0Um93KSB7XG4gICAgICByZXN1bHQgKz0gdGhpcy5yZW5kZXJDZWxsKFxuICAgICAgICBjb2xJbmRleCxcbiAgICAgICAgbmV4dFJvdyxcbiAgICAgICAgb3B0cyxcbiAgICAgICAgdHJ1ZSxcbiAgICAgICk7XG4gICAgICBpZiAobmV4dFJvd1tjb2xJbmRleF0gPT09IG5leHRSb3dbbmV4dFJvdy5sZW5ndGggLSAxXSkge1xuICAgICAgICBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgICByZXN1bHQgKz0gdGhpcy5vcHRpb25zLmNoYXJzLnJpZ2h0O1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgICAgfVxuICAgICAgICByZXR1cm4gcmVzdWx0O1xuICAgICAgfVxuICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIgJiYgYjJCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMubWlkLnJlcGVhdChsZW5ndGgpO1xuICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMuYm90dG9tLnJlcGVhdChsZW5ndGgpO1xuICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgIHJlc3VsdCArPSB0aGlzLm9wdGlvbnMuY2hhcnMudG9wLnJlcGVhdChsZW5ndGgpO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXN1bHQgKz0gXCIgXCIucmVwZWF0KGxlbmd0aCk7XG4gICAgfVxuXG4gICAgaWYgKGNvbEluZGV4ID09PSBvcHRzLmNvbHVtbnMgLSAxKSB7XG4gICAgICBpZiAoYjFCb3JkZXIgJiYgYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5yaWdodE1pZDtcbiAgICAgIH0gZWxzZSBpZiAoYjFCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy5ib3R0b21SaWdodDtcbiAgICAgIH0gZWxzZSBpZiAoYjJCb3JkZXIpIHtcbiAgICAgICAgcmVzdWx0ICs9IHRoaXMub3B0aW9ucy5jaGFycy50b3BSaWdodDtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJlc3VsdCArPSBcIiBcIjtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4gcmVzdWx0O1xuICB9XG59XG4iXX0=